FROM golang:1.15.2-alpine3.12

WORKDIR /go/src/app
#COPY . .

RUN apk add git

#containernet network utilities
RUN apk add iputils \
    iproute2 \
    net-tools \
    iperf \
    xterm \
    busybox-extras 

RUN apk add bash

RUN go get -d -v github.com/edoardesd/mqtt-benchmark

RUN go install -v github.com/edoardesd/mqtt-benchmark

