# lukasmalkmus/tankerkoenig_exporter

> A Tankerkoenig API Exporter for Prometheus. - by **[Lukas Malkmus]**

[![Travis Status][travis_badge]][travis]
[![Go Report][report_badge]][report]
[![Latest Release][release_badge]][release]
[![License][license_badge]][license]
[![Docker][docker_badge]][docker]

---

## Table of Contents

1. [Introduction](#introduction)
2. [Usage](#usage)
3. [Contributing](#contributing)
4. [License](#license)

### Introduction

The *tankerkoenig_exporter* is a simple server that scrapes the Tankerkoenig API
for gas station prices and exports them via HTTP for Prometheus consumption.

### Usage

The first step is to grab an API key from the [Tankerkoenig site]. After that
find some station IDs. Either use the API yourself or the [TankstellenFinder].

**Important:** Be advised to set a high scrape interval (e.g. 5min). Each scrape
performs a direct API call and to frequent requests can lead to the
_deauthorization_ of your API key!

**Note:** Since *tankerkoenig* isn't a very handy word, the metric namespace is
`tk_`.

#### Installation

The easiest way to run the *tankerkoenig* is by grabbing the latest binary from
the [release page][release].

##### Building from source

This project uses [go mod] for vendoring.

```bash
git clone https://github.com/lukasmalkmus/tankerkoenig.git
cd tankerkoenig
make build
```

#### Using the application

```bash
./tankerkoenig [flags]
```

Help on flags:

```bash
./tankerkoenig --help
```

#### Using docker

Docker images are now available on [DockerHub]!

```bash
docker run -p9386:9386/tcp -e TANKERKOENIG_API_KEY="YOUR_API_TOKEN" lukasmalkmus/tankerkoenig-exporter:v0.7.0 \
        --api.stations="9646eb5e-b7ae-4205-bdbd-0a64abc46c20,7566fb7a-b7cc-5214-bcad-0a53abd46d14"
```

### Contributing

Feel free to submit PRs or to fill Issues. Every kind of help is appreciated.

### License

© Lukas Malkmus, 2019

Distributed under Apache License (`Apache License, Version 2.0`).

See [LICENSE](LICENSE) for more information.

<!-- Links -->
[go mod]: https://golang.org/cmd/go/#hdr-Module_maintenance
[Lukas Malkmus]: https://github.com/lukasmalkmus
[Tankerkoenig site]: https://creativecommons.tankerkoenig.de/#usage
[TankstellenFinder]: https://creativecommons.tankerkoenig.de/TankstellenFinder/index.html
[DockerHub]: https://hub.docker.com/r/lukasmalkmus/tankerkoenig-exporter

<!-- Badges -->
[travis]: https://travis-ci.com/lukasmalkmus/tankerkoenig_exporter
[travis_badge]: https://travis-ci.com/lukasmalkmus/tankerkoenig_exporter.svg
[report]: https://goreportcard.com/report/github.com/lukasmalkmus/tankerkoenig_exporter
[report_badge]: https://goreportcard.com/badge/github.com/lukasmalkmus/tankerkoenig_exporter
[release]: https://github.com/lukasmalkmus/tankerkoenig_exporter/releases
[release_badge]: https://img.shields.io/github/release/lukasmalkmus/tankerkoenig_exporter.svg
[license]: https://opensource.org/licenses/Apache-2.0
[license_badge]: https://img.shields.io/badge/license-Apache-blue.svg
[docker]: https://hub.docker.com/r/lukasmalkmus/tankerkoenig-exporter
[docker_badge]: https://img.shields.io/docker/pulls/lukasmalkmus/tankerkoenig-exporter.svg
