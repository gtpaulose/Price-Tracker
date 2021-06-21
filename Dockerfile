FROM golang:1.15.0-alpine3.12 AS builder
 
RUN apk add --no-cache \
    git

WORKDIR /src
COPY . .

ENV CGO_ENABLED=0
RUN	go build -o tracker -a -installsuffix nocgo ./cmd/main.go

ENTRYPOINT ["./tracker"]