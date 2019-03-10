# FROM golang:1.10.1-alpine3.7 as build
# RUN apk update && apk add ca-certificates
# RUN pwd

# RUN mkdir -p ./src/github.com/justinbarrick/fluxcloud
# ADD . ./src/github.com/justinbarrick/fluxcloud
# WORKDIR ./src/github.com/justinbarrick/fluxcloud
# RUN go build cmd/fluxcloud.go

# FROM scratch
# COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
# COPY --from=build /go/src/github.com/justinbarrick/fluxcloud/fluxcloud /fluxcloud
# EXPOSE 3031
# RUN ["/bin/ls"]
# ENTRYPOINT ["/fluxcloud"]


FROM golang:1.10.1-alpine3.7
RUN apk update && apk add ca-certificates

RUN mkdir -p ./src/github.com/justinbarrick/fluxcloud
ADD . ./src/github.com/justinbarrick/fluxcloud
WORKDIR ./src/github.com/justinbarrick/fluxcloud
RUN go build cmd/fluxcloud.go