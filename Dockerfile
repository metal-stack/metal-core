FROM metalstack/builder:latest as builder

FROM registry.fi-ts.io/metal/supermicro:2.5.0 as sum

FROM letsdeal/redoc-cli:latest as docbuilder
COPY --from=builder /work/spec/metal-core.json /spec/metal-core.json
RUN redoc-cli bundle -o /generate/index.html /spec/metal-core.json

FROM alpine:3.12

RUN apk add -U \
    ca-certificates \
    ipmitool \
    libpcap-dev

COPY --from=builder /work/bin/metal-core /
COPY --from=docbuilder /generate/index.html /generate/index.html
COPY --from=sum /usr/bin/sum /usr/bin/sum

ENTRYPOINT ["/metal-core"]
