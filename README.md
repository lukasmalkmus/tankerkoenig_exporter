# lukasmalkmus/tankerkoenig_exporter
> An Tankerkoenig API Exporter for Prometheus. - by **[Lukas Malkmus](https://github.com/lukasmalkmus)**

[![Travis Status][travis_badge]][travis]
[![Coverage Status][coverage_badge]][coverage]
[![Go Report][report_badge]][report]
[![Latest Release][release_badge]][release]
[![License][license_badge]][license]

---

## Table of Contents
1. [Introduction](#introduction)
2. [Usage](#usage)
3. [Contributing](#contributing)
4. [License](#license)

### Introduction
The *tankerkoenig_exporter* is a simple server that scrapes the Tankerkoenig API
for gas station prices and exports them via HTTP for Prometheus consumption.

It is so simple, that I haven't even followed the best practices (everything is
squeezed into one metric). After the initial boilerplate code writing, it took
only a few minutes to get the exporter up and running. And I'm far from being a
go and/or prometheus expert. But if someone has an actual use case for this
exporter I might agree to do some refactoring.

### Usage
#### Installation
The easiest way to run the *tankerkoenig_exporter* is by grabbing the latest
binary from the [release page][release].

##### Building from source
This project uses [glide](http://glide.sh) for vendoring.
```bash
git clone https://github.com/lukasmalkmus/tankerkoenig_exporter
cd tankerkoenig_exporter
glide install
go build
# or promu build
```

#### Usage
```bash
./tankerkoenig_exporter [flags]
```

Help on flags:

```bash
./tankerkoenig_exporter --help
```

The first step is to grab an API key from the [Tankerkoenig site](https://creativecommons.tankerkoenig.de/#usage).
Then grab the IDs of the gas stations you want to track. You can use the
Tankerkoenig Demo site for this.
The next step is to set the correct flags while running the exporter.
Use `-api.key` flag to provide your API key and pass a comma seperated list of
gas station IDs to `-api.stations`.

**Important:** Keep in mind that the Tankerkoenig API shouldn't be accessed to
often or your API key might get suspended. Set a high scrape interval, e.g. `10m`.

**Note:** Since *tankerkoenig* isn't a very handy word, metrics have been
shortened to `tk_`.

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
[release]: https://github.com/lukasmalkmus/tankerkoenig_exporter/releases
[release_badge]: https://img.shields.io/github/release/lukasmalkmus/tankerkoenig_exporter.svg
[license]: https://opensource.org/licenses/Apache-2.0
[license_badge]: https://img.shields.io/badge/license-Apache-blue.svg
