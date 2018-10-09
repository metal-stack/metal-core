FROM rethinkdb as builder

RUN apt update -qq \
 && apt install -y \
    python \
    python-pip \
 && pip install \
    rethinkdb

WORKDIR /data

COPY testdata/* ./

RUN /bin/bash -c "rethinkdb --bind all &" \
 && sleep 7 \
 && rethinkdb import -f facilities.json --table metalapi.facility \
 && rethinkdb import -f sizes.json --table metalapi.size \
 && rethinkdb import -f images.json --table metalapi.image \
 && rethinkdb import -f testdevices.json --table metalapi.device \
 && cp -r rethinkdb_data /

FROM rethinkdb

COPY --from=builder /rethinkdb_data/ /data/rethinkdb_data/
