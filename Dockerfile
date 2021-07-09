FROM metalstack/builder:latest as builder

FROM r.metal-stack.io/metal/supermicro:2.5.2 as sum

FROM letsdeal/redoc-cli:latest as docbuilder
COPY --from=builder /work/spec/metal-core.json /spec/metal-core.json
RUN redoc-cli bundle -o /generate/index.html /spec/metal-core.json

FROM debian:10-slim

RUN apt update \
 && apt install --yes --no-install-recommends \
    ca-certificates \
    ipmitool \
#     libpcap-dev \
 # /usr/bin/sum is provided by busybox
 && rm /usr/bin/sum 

COPY --from=builder /work/bin/metal-core /
COPY --from=docbuilder /generate/index.html /generate/index.html
COPY --from=sum /usr/bin/sum /usr/bin/sum

ENTRYPOINT ["/metal-core"]
