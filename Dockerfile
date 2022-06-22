FROM metalstack/builder:latest as builder

FROM alpine:3.16

RUN apk add \
    ca-certificates
COPY --from=builder /work/bin/metal-core /

ENTRYPOINT ["/metal-core"]
