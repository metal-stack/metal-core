FROM metalstack/builder:latest as builder

FROM letsdeal/redoc-cli:latest as docbuilder
COPY --from=builder /work/metal-api.json /spec/metal-api.json
COPY --from=builder /work/spec/metal-core.json /spec/metal-core.json
RUN redoc-cli bundle -o /generate/index.html /spec/metal-api.json /spec/metal-core.json

FROM alpine:3.10

RUN apk add -U \
    ca-certificates \
    ipmitool \
    libpcap-dev

COPY --from=builder /work/bin/metal-core /
COPY --from=docbuilder /generate/index.html /generate/index.html

ENTRYPOINT ["/metal-core"]
