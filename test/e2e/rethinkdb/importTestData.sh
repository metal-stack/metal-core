#!/usr/bin/env bash

rethinkdb --bind all &

sleep 7

rethinkdb import -f /facilities.json --table metalapi.facility
rethinkdb import -f /sizes.json --table metalapi.size
rethinkdb import -f /images.json --table metalapi.image
rethinkdb import -f /testdevices.json --table metalapi.device
