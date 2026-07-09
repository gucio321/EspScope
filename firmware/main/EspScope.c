#define API_VERSION_MAJOR 1
#define API_VERSION_MINOR 1

#include <string.h>
#include <stdio.h>
#include <errno.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>

#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "freertos/event_groups.h"

#include "esp_log.h"
#include "esp_wifi.h"
#include "esp_event.h"
#include "esp_netif.h"
#include "nvs_flash.h"

#include "esp_http_server.h"

#include "esp_adc/adc_continuous.h"
#include "esp_adc/adc_cali.h"
#include "esp_adc/adc_cali_scheme.h"
#include "hal/adc_types.h"

#include "esp_app_desc.h"

#include "esp_ota_ops.h"

#ifdef CONFIG_AP_NAME
        #define WIFI_SSID       CONFIG_AP_NAME
#else
        #define WIFI_SSID       "ESP_SCOPE"
#endif // CONFIG_AP_NAME

#define WIFI_PASS       CONFIG_AP_PASSWORD
#define WIFI_CHANNEL    6
#define MAX_STA_CONN    1

#define ADC_UNIT        ADC_UNIT_1

#if CONFIG_ADC_CHAN_CUSTOM
        #define ADC_CHANNEL     CONFIG_ADC_CHAN
#else
        #define ADC_CHANNEL     ADC_CHANNEL_0   // GPIO0 on ESP32-C3
#endif // CONFIG_ADC_CHAN


#define ADC_ATTEN       ADC_ATTEN_DB_12
#define ADC_BIT_WIDTH   SOC_ADC_DIGI_MAX_BITWIDTH

#define DEFAULT_SAMPLE_RATE_HZ  50000           // start conservative
#define ABSOLUTE_MAX_SAMPLE_RATE_HZ SOC_ADC_SAMPLE_FREQ_THRES_HIGH
#define ADC_FRAME_BYTES 1024
#define UDP_SAMPLES     512

#define MAGIC           0x53434F50u      // "SCOP"

static const char *TAG = "esp_scope";

static uint32_t sample_rate_hz = DEFAULT_SAMPLE_RATE_HZ;
static adc_cali_handle_t cali_handle;
static adc_continuous_handle_t adc_handle;

static volatile bool streaming = false;

static int udp_sock = -1;
static struct sockaddr_in client_addr;
static uint32_t packet_seq = 0;

#include "esp_adc/adc_cali.h"
#include "esp_adc/adc_oneshot.h"

#define ADC_MAX 4095

uint16_t adc_lut[ADC_MAX + 1]; // adc lookup table. see table_handler

static esp_ota_handle_t ota_handle = 0;
static const esp_partition_t *ota_partition = NULL;
static bool ota_in_progress = false;

typedef struct __attribute__((packed)) {
    uint32_t magic;
    uint32_t seq;
    uint32_t sample_rate;
    uint16_t sample_count;
    uint16_t samples[UDP_SAMPLES];
} scope_packet_t;

void build_adc_lut()
{
    for (int raw = 0; raw <= ADC_MAX; raw++) {
        int mv = 0;

        adc_cali_raw_to_voltage(cali_handle, raw, &mv);

        if (mv < 0) mv = 0;
        if (mv > 3300) mv = 3300;  // optional clamp

        adc_lut[raw] = (uint16_t)mv;
    }
}

void adc_init_fit(void)
{
    adc_cali_curve_fitting_config_t cali_config = {
        .unit_id = ADC_UNIT,
        .atten = ADC_ATTEN,
        .bitwidth = ADC_BIT_WIDTH,
    };

    ESP_ERROR_CHECK(
        adc_cali_create_scheme_curve_fitting(&cali_config, &cali_handle)
    );

    build_adc_lut();
}

static void wifi_init_softap(void)
{
    ESP_ERROR_CHECK(esp_netif_init());
    ESP_ERROR_CHECK(esp_event_loop_create_default());

    esp_netif_create_default_wifi_ap();

    wifi_init_config_t cfg = WIFI_INIT_CONFIG_DEFAULT();
    ESP_ERROR_CHECK(esp_wifi_init(&cfg));

    wifi_config_t wifi_config = {
        .ap = {
            .ssid = WIFI_SSID,
            .ssid_len = strlen(WIFI_SSID),
            .channel = WIFI_CHANNEL,
            .password = WIFI_PASS,
            .max_connection = MAX_STA_CONN,
            .authmode = WIFI_AUTH_WPA_WPA2_PSK,
            .pmf_cfg = {
                .required = false,
            },
        },
    };

    // handle empty password configuration
    if (strlen(WIFI_PASS) == 0) {
        wifi_config.ap.authmode = WIFI_AUTH_OPEN;
    }

    ESP_ERROR_CHECK(esp_wifi_set_mode(WIFI_MODE_AP));
    ESP_ERROR_CHECK(esp_wifi_set_config(WIFI_IF_AP, &wifi_config));
    ESP_ERROR_CHECK(esp_wifi_start());

    ESP_LOGI(TAG, "SoftAP started: SSID=%s password=%s IP=192.168.4.1",
             WIFI_SSID, WIFI_PASS);
}

static void adc_reconfigure_continuous(uint32_t new_sample_rate_hz) {
    adc_digi_pattern_config_t adc_pattern = {
        .atten = ADC_ATTEN,
        .channel = ADC_CHANNEL,
        .unit = ADC_UNIT,
        .bit_width = ADC_BIT_WIDTH,
    };

    adc_continuous_config_t dig_cfg = {
        .sample_freq_hz = new_sample_rate_hz,
        .conv_mode = ADC_CONV_SINGLE_UNIT_1,
        .format = ADC_DIGI_OUTPUT_FORMAT_TYPE2,
        .pattern_num = 1,
        .adc_pattern = &adc_pattern,
    };

    sample_rate_hz = new_sample_rate_hz;

    ESP_ERROR_CHECK(adc_continuous_config(adc_handle, &dig_cfg));
}

static void adc_init_continuous(void)
{
    adc_continuous_handle_cfg_t adc_config = {
        .max_store_buf_size = 4096,
        .conv_frame_size = ADC_FRAME_BYTES,
    };

    ESP_ERROR_CHECK(adc_continuous_new_handle(&adc_config, &adc_handle));

    adc_reconfigure_continuous(DEFAULT_SAMPLE_RATE_HZ);
}

static esp_err_t start_handler(httpd_req_t *req)
{
    if (ota_in_progress) {
        httpd_resp_set_status(req, "503 Service Unavailable");
        return httpd_resp_sendstr(req, "OTA update in progress!\n");
    }
    
    char query[128] = {0};
    char ip_str[32] = "192.168.4.2";
    char port_str[16] = "5000";

    if (httpd_req_get_url_query_str(req, query, sizeof(query)) == ESP_OK) {
        httpd_query_key_value(query, "ip", ip_str, sizeof(ip_str));
        httpd_query_key_value(query, "port", port_str, sizeof(port_str));
    }

    int port = atoi(port_str);

    memset(&client_addr, 0, sizeof(client_addr));
    client_addr.sin_family = AF_INET;
    client_addr.sin_port = htons(port);
    inet_pton(AF_INET, ip_str, &client_addr.sin_addr);

    if (udp_sock < 0) {
        udp_sock = socket(AF_INET, SOCK_DGRAM, IPPROTO_IP);
        if (udp_sock < 0) {
            ESP_LOGE(TAG, "socket failed: errno=%d", errno);
            httpd_resp_send_500(req);
            return ESP_FAIL;
        }
    }

    packet_seq = 0;
    streaming = true;
    ESP_ERROR_CHECK(adc_continuous_start(adc_handle));

    ESP_LOGI(TAG, "Streaming started to %s:%d", ip_str, port);
    httpd_resp_sendstr(req, "OK streaming started\n");
    return ESP_OK;
}

static esp_err_t stop_handler(httpd_req_t *req)
{
    if (ota_in_progress) {
        httpd_resp_set_status(req, "503 Service Unavailable");
        return httpd_resp_sendstr(req, "OTA update in progress!\n");
    }

    streaming = false;
    adc_continuous_stop(adc_handle);

    ESP_LOGI(TAG, "Streaming stopped");
    httpd_resp_sendstr(req, "OK streaming stopped\n");
    return ESP_OK;
}

static esp_err_t root_handler(httpd_req_t *req)
{
    const char *msg =
        "ESP Scope\n"
        "Use:\n"
        "  /start?ip=IP&port=PORT - start streaming to IP (default: 192.168.4.1) on port PORT (default: 5000)\n"
        "  /adc_table - ADC calibration data\n"
        "  /ota - OTA update request\n"
        "  /version - Get firmware version\n"
        "  /stop - Stop streaming\n"
        "  /sample_rate?set=RATE - Set sample rate (Hz). Default is 50k.\n";

    httpd_resp_sendstr(req, msg);

    if (ota_in_progress) {
        httpd_resp_set_status(req, "503 Service Unavailable");
        return httpd_resp_sendstr(req, "OTA update in progress!\n");
    }

    return ESP_OK;
}

static esp_err_t table_handler(httpd_req_t *req)
{
    if (ota_in_progress) {
        httpd_resp_set_status(req, "503 Service Unavailable");
        return httpd_resp_sendstr(req, "OTA update in progress!\n");
    }

    const size_t table_size_bytes = sizeof(adc_lut);

    httpd_resp_set_type(req, "application/octet-stream");
    httpd_resp_set_hdr(req, "Content-Disposition",
                        "attachment; filename=\"adc_lut.bin\"");

    size_t sent = 0;
    const size_t chunk_size = 1024;

    while (sent < table_size_bytes) {
        size_t to_send = table_size_bytes - sent;
        if (to_send > chunk_size) {
            to_send = chunk_size;
        }

        int ret = httpd_resp_send_chunk(
            req,
            (const char *)((uint8_t *)adc_lut + sent),
            to_send
        );

        if (ret != ESP_OK) {
            return ESP_FAIL;
        }

        sent += to_send;
    }

    httpd_resp_send_chunk(req, NULL, 0); // finish transfer

    return ESP_OK;
}

static esp_err_t ota_upload_handler(httpd_req_t *req)
{
    char buf[1024];
    int received;

    if (!ota_in_progress)
    {
        ESP_LOGI(TAG, "Starting OTA update");

        if (streaming) {
            streaming = false;
            adc_continuous_stop(adc_handle);
            ESP_LOGI(TAG, "Streaming stopped due to OTA request");
        }

        ota_partition = esp_ota_get_next_update_partition(NULL);
        if (!ota_partition) {
            ESP_LOGE(TAG, "No OTA partition found");
            return ESP_FAIL;
        }

        ESP_ERROR_CHECK(esp_ota_begin(ota_partition, OTA_SIZE_UNKNOWN, &ota_handle));
        ota_in_progress = true;
    }

    // Stream data from HTTP request
    while ((received = httpd_req_recv(req, buf, sizeof(buf))) > 0)
    {
        esp_err_t err = esp_ota_write(ota_handle, buf, received);
        if (err != ESP_OK) {
            ESP_LOGE(TAG, "OTA write failed");
            esp_ota_abort(ota_handle);
            ota_in_progress = false;
            return ESP_FAIL;
        }
    }

    // Client closed connection OR finished upload
    if (received == HTTPD_SOCK_ERR_TIMEOUT) {
        return ESP_OK;
    }

    if (received < 0) {
        ESP_LOGE(TAG, "Receive error");
        esp_ota_abort(ota_handle);
        ota_in_progress = false;
        return ESP_FAIL;
    }

    // Finalize OTA
    ESP_ERROR_CHECK(esp_ota_end(ota_handle));
    ESP_ERROR_CHECK(esp_ota_set_boot_partition(ota_partition));

    ESP_LOGI(TAG, "OTA complete, restarting...");

    ota_in_progress = false;

    // Kindly say "thank you" to the OTA provider and let it close before rebooting the server.
    httpd_resp_sendstr(req, "OTA OK, rebooting...\n");
    httpd_resp_send(req, NULL, 0);   // flush
    vTaskDelay(pdMS_TO_TICKS(100));  // allow TCP to send

    esp_restart();
    return ESP_OK;
}

static esp_err_t version_handler(httpd_req_t *req)
{
    if (ota_in_progress) {
        httpd_resp_set_status(req, "503 Service Unavailable");
        return httpd_resp_sendstr(req, "OTA update in progress!\n");
    }

    const esp_app_desc_t *app = esp_app_get_description();

    char msg[256];

    snprintf(msg, sizeof(msg),
         "{\n"
         "\t\"ProjectName\": \"%s\",\n"
         "\t\"CommitID\": \"%s\",\n"
         "\t\"BuildDate\": \"%s %s\",\n"
         "\t\"APIVersionMajor\": %d,\n"
         "\t\"APIVersionMinor\": %d\n"
         "}\n",
         app->project_name,
         app->version,
         app->date,
         app->time,
         API_VERSION_MAJOR,
         API_VERSION_MINOR);

    httpd_resp_sendstr(req, msg);
    return ESP_OK;
}

static esp_err_t sample_rate_handler(httpd_req_t *req)
{
    // make sure no OTA is in progress.
    if (ota_in_progress) {
        httpd_resp_set_status(req, "503 Service Unavailable");
        return httpd_resp_sendstr(req, "OTA update in progress!\n");
    }
    
    char query[128] = {0};
    char new_sample_rate_str[16];
    sprintf(new_sample_rate_str, "%ld", sample_rate_hz);

    if (httpd_req_get_url_query_str(req, query, sizeof(query)) == ESP_OK) {
        httpd_query_key_value(query, "set", new_sample_rate_str, sizeof(new_sample_rate_str));
    }

    if (strlen(new_sample_rate_str) == 0) {
        char msg[128];
        sprintf(msg, "{\"SampleRateHz\": %ld}\n", sample_rate_hz);
        return httpd_resp_sendstr(req, msg);
    }

    int new_sample_rate = atoi(new_sample_rate_str);

    if (new_sample_rate <= SOC_ADC_SAMPLE_FREQ_THRES_LOW) {
        httpd_resp_set_status(req, "400 Bad Request");
        return httpd_resp_sendstr(req, "{\"Message\":\"Invalid sample rate. Must be more than SOC_ADC_SAMPLE_FREQ_THRES_HIGH.\"}\n");
    } else if (new_sample_rate > ABSOLUTE_MAX_SAMPLE_RATE_HZ) {
        httpd_resp_set_status(req, "400 Bad Request");
        return httpd_resp_sendstr(req, "{\"Message\":\"Invalid sample rate. Must be less than ABSOLUTE_MAX_SAMPLE_RATE_HZ (see firmware).\"}\n");
    } else if (streaming) {
        httpd_resp_set_status(req, "400 Bad Request");
        return httpd_resp_sendstr(req, "{\"Message\":\"Cannot change sample rate while streaming.\"}\n");
    }

    if (new_sample_rate == sample_rate_hz) {
        char msg[128];
        sprintf(msg, "{\"SampleRateHz\": %ld,\n\"Message\": \"Same sample rate as already set - aborting\"}\n", sample_rate_hz);
        return httpd_resp_sendstr(req, msg);
    }

    char msg[256];
    sprintf(msg, "{\"SampleRateHz\": %d,\n\"Message\": \"Changing sample rate to %ld=>%d Hz\"}\n", new_sample_rate, sample_rate_hz, new_sample_rate);

    adc_reconfigure_continuous(new_sample_rate);

    httpd_resp_sendstr(req, msg);
    return ESP_OK;
}

static void http_server_start(void)
{
    httpd_config_t config = HTTPD_DEFAULT_CONFIG();
    config.recv_wait_timeout = 10;
    httpd_handle_t server = NULL;

    ESP_ERROR_CHECK(httpd_start(&server, &config));

    httpd_uri_t root_uri = {
        .uri = "/",
        .method = HTTP_GET,
        .handler = root_handler,
    };

    httpd_uri_t start_uri = {
        .uri = "/start",
        .method = HTTP_GET,
        .handler = start_handler,
    };

    httpd_uri_t stop_uri = {
        .uri = "/stop",
        .method = HTTP_GET,
        .handler = stop_handler,
    };

    httpd_uri_t adc_table_uri = {
        .uri      = "/adc_table",
        .method   = HTTP_GET,
        .handler  = table_handler,
        .user_ctx = NULL
    };

    httpd_uri_t ota_upload_uri = {
        .uri      = "/ota",
        .method   = HTTP_POST,
        .handler  = ota_upload_handler,
        .user_ctx = NULL
    };

    httpd_uri_t version_uri = {
        .uri      = "/version",
        .method   = HTTP_GET,
        .handler  = version_handler,
        .user_ctx = NULL
    };

    httpd_uri_t sample_rate_uri = {
        .uri      = "/sample_rate",
        .method   = HTTP_GET,
        .handler  = sample_rate_handler,
        .user_ctx = NULL
    };

    ESP_ERROR_CHECK(httpd_register_uri_handler(server, &root_uri));
    ESP_ERROR_CHECK(httpd_register_uri_handler(server, &start_uri));
    ESP_ERROR_CHECK(httpd_register_uri_handler(server, &stop_uri));
    ESP_ERROR_CHECK(httpd_register_uri_handler(server, &adc_table_uri));
    ESP_ERROR_CHECK(httpd_register_uri_handler(server, &ota_upload_uri));
    ESP_ERROR_CHECK(httpd_register_uri_handler(server, &version_uri));
    ESP_ERROR_CHECK(httpd_register_uri_handler(server, &sample_rate_uri));
}

static void udp_stream_task(void *arg)
{
    uint8_t adc_buf[ADC_FRAME_BYTES];
    scope_packet_t pkt;

    while (1) {
        if (!streaming || udp_sock < 0) {
            vTaskDelay(pdMS_TO_TICKS(20));
            continue;
        }

        uint32_t bytes_read = 0;
        esp_err_t ret = adc_continuous_read(
            adc_handle,
            adc_buf,
            sizeof(adc_buf),
            &bytes_read,
            pdMS_TO_TICKS(100)
        );

        if (ret != ESP_OK || bytes_read == 0) {
            continue;
        }

        pkt.magic = MAGIC;
        pkt.seq = packet_seq++;
        pkt.sample_rate = sample_rate_hz;
        pkt.sample_count = 0;

        for (int i = 0;
             i + sizeof(adc_digi_output_data_t) <= bytes_read && pkt.sample_count < UDP_SAMPLES;
             i += sizeof(adc_digi_output_data_t)) {

            adc_digi_output_data_t *sample = (adc_digi_output_data_t *)&adc_buf[i];

            if (sample->type2.unit == ADC_UNIT_1 &&
                sample->type2.channel == ADC_CHANNEL) {
                pkt.samples[pkt.sample_count++] = sample->type2.data;
            }
        }

        if (pkt.sample_count > 0) {
            size_t packet_size =
                sizeof(pkt.magic) +
                sizeof(pkt.seq) +
                sizeof(pkt.sample_rate) +
                sizeof(pkt.sample_count) +
                pkt.sample_count * sizeof(uint16_t);

            int sent = sendto(
                udp_sock,
                &pkt,
                packet_size,
                0,
                (struct sockaddr *)&client_addr,
                sizeof(client_addr)
            );

            if (sent < 0) {
                ESP_LOGW(TAG, "UDP send failed: errno=%d", errno);
            }
        }
    }
}

void app_main(void)
{
    ESP_ERROR_CHECK(nvs_flash_init());

    wifi_init_softap();
    adc_init_fit();
    adc_init_continuous();

    http_server_start();

    esp_ota_mark_app_valid_cancel_rollback();

    xTaskCreate(
        udp_stream_task,
        "udp_stream_task",
        4096,
        NULL,
        5,
        NULL
    );

    ESP_LOGI(TAG, "Ready. Connect to Wi-Fi '%s', then open http://192.168.4.1/",
             WIFI_SSID);
}

