# Production image based on alpine.
FROM alpine
LABEL maintainer="Lukas Malkmus <mail@lukasmalkmus.com>"

# Upgrade packages and install ca-certificates.
RUN apk update --no-cache \
    apk upgrade --no-cache \
    apk add --no-cache ca-certificates

# Copy binary into image.
COPY tankerkoenig_exporter /usr/bin/tankerkoenig_exporter

# Use the project name as working directory.
WORKDIR /tankerkoenig_exporter

# Set the binary as entrypoint.
ENTRYPOINT [ "/usr/bin/tankerkoenig_exporter" ]
