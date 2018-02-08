FROM golang:1.9.1


ADD . /go/src/github.com/last-ent/goophr/librarian

RUN go install github.com/last-ent/goophr/librarian

ENTRYPOINT /go/bin/librarian

