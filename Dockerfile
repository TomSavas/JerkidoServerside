FROM golang:alpine

WORKDIR /go/src/jerkido
COPY ./src ../jerkido

RUN apk update && apk upgrade && apk add --no-cache git
RUN go get -d ./...
RUN go install -v ./...

ENTRYPOINT jerkido | tee log
