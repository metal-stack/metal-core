FROM rethinkdb as builder

RUN apt update -qq \
 && apt install -y \
    python \
    python-pip \
 && pip install \
    rethinkdb

COPY testdata/* /
COPY importTestData.sh /data

RUN ./importTestData.sh \
 && cp -r /data/rethinkdb_data /

FROM rethinkdb

COPY --from=builder /rethinkdb_data/ /data/rethinkdb_data/
