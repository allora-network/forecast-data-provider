# node build
from golang:1.22-bookworm as gobuilder
WORKDIR /
# RUN apt-get update && apt-get install -y curl

COPY . .
RUN go build .

# final image
FROM debian:bookworm-slim

RUN apt update && \
    apt -y dist-upgrade && \
    apt install -y --no-install-recommends \
        curl jq \
        tzdata \
        bc \
        ca-certificates && \
    echo "deb http://deb.debian.org/debian testing main" >> /etc/apt/sources.list && \
    apt update && \
    apt install -y --no-install-recommends -t testing \
      zlib1g \
      libgnutls30 \
      perl-base && \
    rm -rf /var/cache/apt/*

# Install PostgreSQL client
RUN apt-get update && apt-get install -y postgresql-client && rm -rf /var/lib/apt/lists/*

WORKDIR /
# Detect the architecture and download the appropriate binary
ARG TARGETARCH

RUN mkdir -p /usr/local/bin/previous/v2
RUN if [ "$TARGETARCH" = "arm64" ]; then \
        curl -L https://github.com/allora-network/allora-chain/releases/download/v0.2.11/allorad_linux_arm64 -o /usr/local/bin/previous/v2/allorad; \
        curl -L https://github.com/allora-network/allora-chain/releases/download/v0.3.0/allorad_linux_arm64 -o /usr/local/bin/allorad; \
    else \
        curl -L https://github.com/allora-network/allora-chain/releases/download/v0.2.11/allorad_linux_amd64 -o /usr/local/bin/previous/v2/allorad; \
        curl -L https://github.com/allora-network/allora-chain/releases/download/v0.3.0/allorad_linux_amd64 -o /usr/local/bin/allorad; \
    fi

RUN chmod -R 777 /usr/local/bin/allorad
RUN chmod -R 777 /usr/local/bin/previous/v2/allorad

COPY --from=gobuilder forecast-data-provider /usr/local/bin/forecast-data-provider
WORKDIR /usr/local/bin
# EXPOSE 8080
ENTRYPOINT ["forecast-data-provider"]
