FROM golang:1.27-alpine AS source

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY cmd ./cmd
COPY internal ./internal

FROM source AS builder

ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=${VERSION}" -o /pvm .

FROM ubuntu:24.04 AS runtime

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
     sudo ca-certificates software-properties-common gpg-agent \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /pvm /usr/local/bin/pvm

ENV PATH=/root/.pvm/bin:$PATH

ENTRYPOINT ["pvm"]
