FROM appropriate/curl as binaries
ENV TINI_VERSION v0.18.0
RUN curl --fail -sLo /tini https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini-static-amd64

FROM debian:bookworm-slim as base
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/bin/satokens"]
RUN apt-get update && apt-get install -y ca-certificates liblz4-1 && rm -rf /var/lib/apt/lists/*
RUN useradd -r -u 999 -d /home/satokens satokens
COPY --from=binaries /tini /usr/bin/tini
RUN chmod +x /usr/bin/tini

FROM base as goreleaser
COPY satokens /usr/bin/satokens
RUN chmod +x /usr/bin/satokens
USER satokens