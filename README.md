# go-sensors/cubicpm1003

Go library for interacting with the [Cubic PM1003][cubicpm1003] particulate matter sensor for measuring indoor air quality.

## Quickstart

Take a look at [rpi-sensor-exporter][rpi-sensor-exporter] for an example implementation that makes use of this sensor (and others).

[rpi-sensor-exporter]: https://github.com/go-sensors/rpi-sensor-exporter

## Sensor Details

The [Cubic PM1003][cubicpm1003] particulate matter sensor is primarily used for measuring indoor air quality, per [vendor specifications][specs]. This [go-sensors] implementation makes use of the sensor's UART-based protocol for obtaining PM2.5 measurements on a [1-second interval](./defaults.go).

[cubicpm1003]: https://en.gassensor.com.cn/ParticulateMatterSensor/info_itemid_104.html
[specs]: ./docs/Cubic_PM1003_DS.pdf
[go-sensors]: https://github.com/go-sensors

## Building

This software doesn't have any compiled assets.

## Code of Conduct

We are committed to fostering an open and welcoming environment. Please read our [code of conduct](CODE_OF_CONDUCT.md) before participating in or contributing to this project.

## Contributing

We welcome contributions and collaboration on this project. Please read our [contributor's guide](CONTRIBUTING.md) to understand how best to work with us.

## License and Authors

[![Daniel James logo](https://secure.gravatar.com/avatar/eaeac922b9f3cc9fd18cb9629b9e79f6.png?size=16) Daniel James](https://github.com/thzinc)

[![license](https://img.shields.io/github/license/go-sensors/cubic-pm1003.svg)](https://github.com/go-sensors/cubic-pm1003/blob/master/LICENSE)
[![GitHub contributors](https://img.shields.io/github/contributors/go-sensors/cubic-pm1003.svg)](https://github.com/go-sensors/cubic-pm1003/graphs/contributors)

This software is made available by Daniel James under the MIT license.
