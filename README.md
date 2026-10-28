<h1>EspScope</h1>

<div>
  <img
    align="right"
    src="./pkg/assets/logo.png"
    alt="EspScope logo"
    style="width: 50%; height: auto;"
  >
</div>

## Description

This project is a DIY, low cost osciloscope based on ESP32 C3 microcontroller.
It uses an external machine (PC) as a display and also for performing heavy calculations.

## Project components

- [firmware](./firmware) contains a ESP-IDF project flashed to the microcontroller
- [case](./case) contains 3D-printed case for the device.
- [PCB](./PCB) PCB production files
- The rest of the repo contains a go module for the desktop application.


## How it works?

- A simple voltage divider and an amplifier are converting the input voltage so that it is suitable for the EPS32's ADC input.
- ESP32 reads adc value and sends it to the Host PC.
- EspScope Application receives raw ADC data, converts it back to the appropiate units and presents on the UI.

## Voltage range

> [!note]
> The circuit contains overvoltage protection (a set of diods) so it **should** not be damaged by the out-of-range voltage.
> HOWEVER:
> - this was not verified
> - The amplifier (U1.1) outputting overvoltage (5V) should not be the problem due to a shotkey diode which will push it to the ESP's 3.3V regulator (should be save).
> - (This is only a thoery): When voltage on the amplifier (U1.1) is out of its range, it could damage the amplifier.

According to the [schematic](PCB/EspScope_schematic.pdf) the following formula gives $V_{min}$ and $V_{max}$:

$$
V_o = \left(\frac{V_i}{R_1} + \frac{3.3 V}{R_2}\right) * \left(\frac{R_1 R_2 R_3}{R_1 R_2 + R_2 R_3 + R_1 R_3}\right)
$$

Then you can calculate $V_{min}$ and $V_{max}$ with the following formulas:

$$
\begin{aligned}
V_{\min} &= -3.3\,\text{V}\frac{R_1}{R_2} \\
V_{\max} &= 3.3\,\text{V}\left(1+\frac{R_1}{R_3}\right)
\end{aligned}
$$

> [!tip]
> With those formulas, you can choose $R_1$, $R_2$, $R_3$ resistors so that the scope range fits your needs.
> Project recommended values are: $R_1 = 1M \Omega$, $R_2 = 240 k\Omega$ and $R_3 = 316k\Omega$. This setting allows for **input voltage range** $V_i = \pm 13.7V$

> [!important]
> Please keep your resistors values larger than smaller.
> You **don't want** your scope to draw a significient amount of current.
> For the recommended settings, the scope draws $I_{max} \approx 10 \mu A$.

## Bill Of Material (BOM)

| Qty | Item | Notes |
|----|---|---|
| **__Electronics__** |  |  |
| 1  | ESP32-C3 module | Main ESP module. Something like [this from AliExpress](https://pl.aliexpress.com/item/1005012143614522.html) |
| 1 | $R_1$ Resistor for the `R1206` footprint | Recommended: $1 M\Omega$ resistor. I've used [AR1206-1M-0.1%](https://www.tme.eu/pl/details/ar1206-1m-0.1%25/rezystory-smd/viking/ar06btcv1004/) |
| 1 | $R_2$ Resistor for the `R0402` footprint | Recommended: $240 k\Omega$ resistor. |
| 1 | $R_3$ Resistor for the `R0603` footprint | Recommended: $316 k\Omega$ resistor. I've used [ERA3AEB3163V](https://www.tme.eu/pl/details/era3aeb3163v/rezystory-smd/panasonic/) |
| 1 | $R_4$ Resistor for the `R0603` footprint | It should be something around `1 k\Omega` (doesn't matter that much). I've used [ERA3AEB102V](https://www.tme.eu/pl/details/era3aeb102v/rezystory-smd/panasonic/) |
| 1 | Operational Amplifier fo rhe `PDIP-8_L9.3-W6.4-P2.54-LS7.9-BL-1` footprint | I've used [this one from TME: MCP6002-E/P](https://www.tme.eu/pl/details/mcp6002-e_p/wzmacniacze-operacyjne-tht/microchip-technology/) |
| 1 | Double Schottky diode for the `SOT-23-3_L2.9-W1.3-P1.90-LS2.4-BR` | For security to clamp voltage to esp32-compatible range. I've used this: [BAT54S-DIO](https://www.tme.eu/pl/details/bat54s-dio/diody-schottky-smd/diotec-semiconductor/bat54s/) |
| 1 | USB-A socket for the `USB-A-TH__USB-AF-90` footprint | I've used this one from LCSC electronics: [C2345](https://www.lcsc.com/product-detail/C2345.html) as I had one from some of my past projects, but you can use anything compatible. |
| 1 | Switch | I've used [those from TEMU](https://share.temu.com/ZqqPkEJkTWB). Generally you can use similar one, or something else but that might require case modification |
| 2 | banana socket | To comfortably connect wires. [You can use those](https://share.temu.com/lIIGhAWa1IB). |
| 1 | Around 100 g of the PLA (or other) filament. | Note that for the case you need a 3D printer :smile: |
| 1 | PCB from [here](./PCB) | For PCB version. Alternatively you can use some prototyping board (you'll need to solder elements anyway). I've ordered mine on [JLCPCB](https://jlcpcb.com) but any other vendor should be fine. |
|   | ** Extra parts** | |
| 1 | USB-A -> USB-A cable | |
| 2 | Banana probe wires |  |
