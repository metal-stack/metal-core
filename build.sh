#!/usr/bin/env bash

set -e

docker build -t metalcore .
docker save -o metalcore.tar.gz metalcore
