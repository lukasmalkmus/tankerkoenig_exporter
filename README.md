# lukasmalkmus/tankerkoenig_exporter

> An Tankerkoenig API Exporter for Prometheus. - by **[Lukas Malkmus](https://github.com/lukasmalkmus)**

[![Travis Status][travis_badge]][travis]
[![Coverage Status][coverage_badge]][coverage]
[![Go Report][report_badge]][report]
[![GoDoc][docs_badge]][docs]
[![Docker Repository on Quay][quay_badge]][quay]
[![Docker Pulls][hub_badge]][hub]
[![Latest Release][release_badge]][release]
[![License][license_badge]][license]

---

## Table of Contents

1. [Introduction](#introduction)
1. [Usage](#usage)
1. [Contributing](#contributing)
1. [License](#license)

### Introduction

The *tankerkoenig_exporter* is a simple server that scrapes the Tankerkoenig API
for gas station prices and exports them via HTTP for Prometheus consumption.

### Usage

The first step is to grab an API key from the [Tankerkoenig site](https://creativecommons.tankerkoenig.de/#usage).
After that grab some station IDs. Either use the API yourself or the [TankstellenFinder](https://creativecommons.tankerkoenig.de/TankstellenFinder/index.html).

**Important:** Be advised set a high scrape interval (e.g. 5min). Each scrape
performs a direct API call so to frequent requests can lead to the
_deauthorization_ of your API key!

**Note:** Since *tankerkoenig* isn't a very handy word, metrics have been
shortened to `tk_`.

#### Installation

The easiest way to run the *tankerkoenig_exporter* is by grabbing the latest
binary from the [release page][release].

##### Building from source

This project uses [dep](https://github.com/golang/dep) for vendoring.

```bash
git clone https://github.com/lukasmalkmus/tankerkoenig_exporter
cd tankerkoenig_exporter
dep ensure # Install dependencies
make # Build application (needs the prometheus utility `promu` installed)
```

#### Using the exporter

```bash
./tankerkoenig_exporter [flags]
```

Help on flags:

```bash
./tankerkoenig_exporter --help
```

#### Using docker

```bash
docker run -p 9386:9386 quay.io/lukasmalkmus/tankerkoenig-exporter:v0.4.0 \
        --apiKey="YOUR_API_TOKEN" \
        --apiStations="9646eb5e-b7ae-4205-bdbd-0a64abc46c20,7566fb7a-b7cc-5214-bcad-0a53abd46d14"
```

### Contributing

Feel free to submit PRs or to fill Issues. Every kind of help is appreciated.

### License

© Lukas Malkmus, 2017

Distributed under Apache License (`Apache License, Version 2.0`).

See [LICENSE](LICENSE) for more information.

[travis]: https://travis-ci.org/lukasmalkmus/tankerkoenig_exporter
[travis_badge]: https://travis-ci.org/lukasmalkmus/tankerkoenig_exporter.svg
[coverage]: https://coveralls.io/github/lukasmalkmus/tankerkoenig_exporter?branch=master
[coverage_badge]: https://coveralls.io/repos/github/lukasmalkmus/tankerkoenig_exporter/badge.svg?branch=master
[report]: https://goreportcard.com/report/github.com/lukasmalkmus/tankerkoenig_exporter
[report_badge]: https://goreportcard.com/badge/github.com/lukasmalkmus/tankerkoenig_exporter
[docs]: https://godoc.org/github.com/lukasmalkmus/tankerkoenig_exporter
[docs_badge]: https://godoc.org/github.com/lukasmalkmus/tankerkoenig_exporter?status.svg
[quay]: https://quay.io/repository/lukasmalkmus/tankerkoenig-exporter
[quay_badge]: https://quay.io/repository/lukasmalkmus/tankerkoenig-exporter/status
[hub]: https://hub.docker.com/r/lukasmalkmus/tankerkoenig-exporter
[hub_badge]: https://img.shields.io/docker/pulls/lukasmalkmus/tankerkoenig-exporter.svg
[release]: https://github.com/lukasmalkmus/tankerkoenig_exporter/releases
[release_badge]: https://img.shields.io/github/release/lukasmalkmus/tankerkoenig_exporter.svg
[license]: https://opensource.org/licenses/Apache-2.0
[license_badge]: https://img.shields.io/badge/license-Apache-blue.svg