FROM golang:1.27-alpine AS source

# The code is not copied in: compose mounts the project at /src, so a code
# change never needs an image rebuild. Modules and build cache live in .local/.
RUN apk add --no-cache git su-exec
ENV GOPATH=/src/.local/go \
    GOCACHE=/src/.local/go-cache \
    GOFLAGS=-modcacherw \
    HOME=/tmp
COPY cli/go-entrypoint /usr/local/bin/go-entrypoint
WORKDIR /src
ENTRYPOINT ["sh", "/usr/local/bin/go-entrypoint"]

FROM ubuntu:24.04 AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
     sudo ca-certificates \
     software-properties-common \
     gpg-agent \
     zip \
     unzip \
    && rm -rf /var/lib/apt/lists/* \
    && rm /etc/apt/apt.conf.d/docker-clean

# pvm itself is not in the image: it is built into .local/bin, which the container
# sees through the (read-only) project mount, so code changes don't recreate the
# container. `make pvm` / `make shell` copy it to ~/.pvm/bin, where the README
# installs it, so `pvm self-upgrade` can replace it.
ENV PATH=/root/.pvm/bin:$PATH
