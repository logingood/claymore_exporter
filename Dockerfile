FROM golang:1.8

MAINTAINER Murat Mukhtarov <muhtarov.mr@gmail.com>

LABEL version="1.0"
LABEL description="Claymore miner prometheus exporeter"

ADD . /go/src/github.com/murat1985/claymore_exporter

RUN go get github.com/prometheus/client_golang/prometheus
RUN go install github.com/murat1985/claymore_exporter

ENTRYPOINT /go/bin/claymore_exporter
