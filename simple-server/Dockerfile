FROM golang:1.9.1


ADD . /go/src/littlefs

WORKDIR /go/src/littlefs

RUN go install littlefs

ENTRYPOINT /go/bin/littlefs

