ARG ARCH="amd64"
ARG OS="linux"
FROM  quay.io/prometheus/busybox:latest
LABEL maintainer="Lukas Malkmus <mail@lukasmalkmus.com>"

ARG ARCH="amd64"
ARG OS="linux"
COPY .build/${OS}-${ARCH}/tankerkoenig_exporter /bin/tankerkoenig_exporter

ENTRYPOINT ["/bin/tankerkoenig_exporter"]
EXPOSE     9386