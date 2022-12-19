# Tankerkoenig API Exporter for Prometheus

[![Workflow][workflow_badge]][workflow]
[![Latest Release][release_badge]][release]
[![License][license_badge]][license]

---

## Introduction

The _Tankerkoenig API Exporter_ is a simple server that scrapes the
[Tankerkoenig API] for gas station prices and exports them via HTTP for
Prometheus consumption.

## Usage

The first step is to grab an API key from the [Tankerkoenig site].

The exporter supports two different modes of operation:

1. **Geo-Mode**: The exporter will scrape all stations in a given radius around
   a given location.
2. **Station-Mode**: The exporter will scrape only the stations given by their
   IDs. To get station IDs, either use the API yourself or checkout the
   [Tankstellen Finder].

**Important:** Be advised to set a high scrape interval (e.g. 5 minutes). Each
scrape performs an API call and to frequent requests can lead to the
**deauthorization** of your API key!

**Note:** Since _tankerkoenig_ isn't a very handy word, the metric namespace is
`tk`.

### Installation

The easiest way to run the exporter is by grabbing the latest binary from the
[release page][release].

### Using the application

Run the application with the `--help` flag to see all available options with
their descriptions and default values (if any).

#### Geo-Mode

```bash
export TANKERKOENIG_API_KEY="YOUR_API_KEY"
./tankerkoenig --tankerkoenig.location=u0yjje785f4 --tankerkoenig.radius=5
```

**Note:** The `--tankerkoenig.product` flag is currently not implemented.

#### Station-Mode

```bash
export TANKERKOENIG_API_KEY="YOUR_API_KEY"
./tankerkoenig --tankerkoenig.stations="51d4b55e-a095-1aa0-e100-80009459e03a"
```

**Note**: The `--tankerkoenig.stations` flag can be used multiple times to add multiple
stations to scrape.

### Using docker

Docker images are available on the [GitHub Package Registry].

```bash
# .env file contains TANKERKOENIG_API_KEY="YOUR_API_KEY"
docker run -p9386:9386/tcp --env-file=.env ghcr.io/lukasmalkmus/tankerkoenig-exporter:0.10.0 --tankerkoenig.stations="51d4b55e-a095-1aa0-e100-80009459e03a"
```

## Contributing

Feel free to submit PRs or to fill Issues. Every kind of help is appreciated.

## License

© Lukas Malkmus, 2023

Distributed under Apache License (`Apache License, Version 2.0`).

See [LICENSE](LICENSE) for more information.

<!-- Links -->

[tankerkoenig api]: https://creativecommons.tankerkoenig.de/#usage
[tankerkoenig site]: https://creativecommons.tankerkoenig.de/#usage
[tankstellen finder]: https://creativecommons.tankerkoenig.de/TankstellenFinder/index.html
[github package registry]: https://github.com/users/lukasmalkmus/packages?repo_name=tankerkoenig_exporter

<!-- Badges -->

[workflow]: https://github.com/lukasmalkmus/tankerkoenig_exporter/actions/workflows/ci.yaml
[workflow_badge]: https://img.shields.io/github/actions/workflow/status/lukasmalkmus/tankerkoenig_exporter/ci.yaml?branch=main
[release]: https://github.com/lukasmalkmus/tankerkoenig_exporter/releases
[release_badge]: https://img.shields.io/github/release/lukasmalkmus/tankerkoenig_exporter.svg
[license]: https://opensource.org/licenses/Apache-2.0
[license_badge]: https://img.shields.io/badge/license-Apache-blue.svg
