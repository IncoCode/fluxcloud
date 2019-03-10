FROM golang:1.10.1-alpine3.7
RUN apk update && apk add ca-certificates

RUN mkdir -p ./src/github.com/justinbarrick/fluxcloud
ADD . ./src/github.com/justinbarrick/fluxcloud
WORKDIR ./src/github.com/justinbarrick/fluxcloud
RUN go build -o /go/bin/fluxcloud cmd/fluxcloud.go
RUN rm -rf /go/src/github.com/justinbarrick

EXPOSE 3031
ENTRYPOINT ["/go/bin/fluxcloud"]