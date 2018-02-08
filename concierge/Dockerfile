FROM golang:1.9.1


ADD . /go/src/github.com/last-ent/goophr/concierge

WORKDIR /go/src/github.com/last-ent/goophr/concierge

RUN go install github.com/last-ent/goophr/concierge

ENTRYPOINT /go/bin/concierge

