FROM golang:1.25-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o /out/quantriskd ./cmd/quantriskd && \
    go build -o /out/quantriskcli ./cmd/quantriskcli

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates curl sqlite3 && \
    rm -rf /var/lib/apt/lists/*

COPY --from=build /out/ /usr/local/bin/
COPY test/integration/seed.sh /usr/local/bin/seed.sh
RUN chmod +x /usr/local/bin/seed.sh && mkdir -p /data

WORKDIR /data
