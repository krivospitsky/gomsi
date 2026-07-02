FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends msitools lcab && \
    rm -rf /var/lib/apt/lists/*

COPY gomsi /usr/bin/gomsi

ENTRYPOINT ["gomsi"]
