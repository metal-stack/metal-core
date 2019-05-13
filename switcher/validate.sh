#!/bin/sh

set -e

if [ ! -d "frr" ]; then
    git clone --depth 1 https://github.com/FRRouting/frr
fi

echo "building frr container"
cd frr/docker/debian
docker build \
    --rm \
    --build-arg http_proxy="${HTTP_PROXY}" \
    --build-arg https_proxy="${HTTP_PROXY}" \
    -t frr:latest .
cd -

echo "validate frr.conf items in test_data directory"
cd test_data
for i in $(find . -type f -name frr.conf); do
    echo "validating: $i"
    docker run -i \
        --name frr \
        --entrypoint vtysh \
        --rm \
        --volume "$PWD/$i":/frr.conf \
        frr:latest \
        -C -f /frr.conf
done