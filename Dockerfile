FROM  quay.io/prometheus/busybox:latest
LABEL maintainer="Lukas Malkmus <mail@lukasmalkmus.com>"

COPY tankerkoenig_exporter /bin/tankerkoenig_exporter

ENTRYPOINT ["/bin/tankerkoenig_exporter"]
EXPOSE     9386
