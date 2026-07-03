FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends msitools gcab && \
    rm -rf /var/lib/apt/lists/*

COPY linux/amd64/gomsi /usr/bin/gomsi

ENTRYPOINT ["gomsi"]
