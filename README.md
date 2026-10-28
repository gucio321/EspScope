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

This project is a DIY osciloscope based on ESP32 C3 microcontroller.
It uses an external machine (PC) as a display and also for performing heavy calculations.

## Project components

- [firmware](./firmware) contains a ESP-IDF project flashed to the microcontroller
- [case](./case) contains 3D-printed case for the device.
- [PCB](./PCB) PCB production files
- The rest of the repo contains a go module for the desktop application.


## How it works?

![Schematic](./PCB/EspScope_schematic.pdf)
