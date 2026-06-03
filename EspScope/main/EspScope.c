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
#include "hal/adc_types.h"

#define WIFI_SSID       "ESP_SCOPE"
#define WIFI_PASS       "12345678"
#define WIFI_CHANNEL    6
#define MAX_STA_CONN    1

#define ADC_UNIT        ADC_UNIT_1
#define ADC_CHANNEL     ADC_CHANNEL_0   // GPIO0 on ESP32-C3
#define ADC_ATTEN       ADC_ATTEN_DB_12
#define ADC_BIT_WIDTH   SOC_ADC_DIGI_MAX_BITWIDTH

#define SAMPLE_RATE_HZ  50000           // start conservative
#define ADC_FRAME_BYTES 1024
#define UDP_SAMPLES     512

#define MAGIC           0x53434F50u      // "SCOP"

static const char *TAG = "esp_scope";

static adc_continuous_handle_t adc_handle;
static volatile bool streaming = false;

static int udp_sock = -1;
static struct sockaddr_in client_addr;
static uint32_t packet_seq = 0;

typedef struct __attribute__((packed)) {
    uint32_t magic;
    uint32_t seq;
    uint32_t sample_rate;
    uint16_t sample_count;
    uint16_t samples[UDP_SAMPLES];
} scope_packet_t;

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

static void adc_init_continuous(void)
{
    adc_continuous_handle_cfg_t adc_config = {
        .max_store_buf_size = 4096,
        .conv_frame_size = ADC_FRAME_BYTES,
    };

    ESP_ERROR_CHECK(adc_continuous_new_handle(&adc_config, &adc_handle));

    adc_digi_pattern_config_t adc_pattern = {
        .atten = ADC_ATTEN,
        .channel = ADC_CHANNEL,
        .unit = ADC_UNIT,
        .bit_width = ADC_BIT_WIDTH,
    };

    adc_continuous_config_t dig_cfg = {
        .sample_freq_hz = SAMPLE_RATE_HZ,
        .conv_mode = ADC_CONV_SINGLE_UNIT_1,
        .format = ADC_DIGI_OUTPUT_FORMAT_TYPE2,
        .pattern_num = 1,
        .adc_pattern = &adc_pattern,
    };

    ESP_ERROR_CHECK(adc_continuous_config(adc_handle, &dig_cfg));
}

static esp_err_t start_handler(httpd_req_t *req)
{
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
        "  /start?ip=192.168.4.2&port=5000\n"
        "  /stop\n";
    httpd_resp_sendstr(req, msg);
    return ESP_OK;
}

static void http_server_start(void)
{
    httpd_config_t config = HTTPD_DEFAULT_CONFIG();
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

    ESP_ERROR_CHECK(httpd_register_uri_handler(server, &root_uri));
    ESP_ERROR_CHECK(httpd_register_uri_handler(server, &start_uri));
    ESP_ERROR_CHECK(httpd_register_uri_handler(server, &stop_uri));
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
        pkt.sample_rate = SAMPLE_RATE_HZ;
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
    adc_init_continuous();
    http_server_start();

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
