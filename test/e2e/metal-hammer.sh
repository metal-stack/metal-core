#!/usr/bin/env sh

set -e

echo "METAL_CORE_ADDRESS=${METAL_CORE_ADDRESS}" > /cmdline
mount -n --bind -o ro /cmdline /proc/cmdline
/metal-hammer
