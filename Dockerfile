FROM golang:1.27-alpine AS source

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY cmd ./cmd
COPY internal ./internal

FROM ubuntu:24.04 AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
     sudo ca-certificates software-properties-common gpg-agent \
    && rm -rf /var/lib/apt/lists/* \
    && rm /etc/apt/apt.conf.d/docker-clean

# pvm itself is not in the image: it is built into .local/bin, which the container
# sees through the (read-only) project mount, so code changes don't recreate the
# container. `make pvm` / `make shell` copy it to /usr/local/bin, a writable
# location, so `pvm self-upgrade` can replace it.
ENV PATH=/root/.pvm/bin:$PATH
