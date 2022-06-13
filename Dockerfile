FROM metalstack/builder:latest as builder

FROM r.metal-stack.io/metal/supermicro:2.5.2 as sum

FROM debian:10-slim

RUN apt update \
 && apt install --yes --no-install-recommends \
    ca-certificates \
    ipmitool \
 # /usr/bin/sum is provided by busybox
 && rm /usr/bin/sum 

COPY --from=builder /work/bin/metal-core /
COPY --from=sum /usr/bin/sum /usr/bin/sum

ENTRYPOINT ["/metal-core"]
