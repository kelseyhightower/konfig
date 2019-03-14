FROM golang:1.12beta1-stretch as builder
ENV GO111MODULE=on
WORKDIR /module
COPY . /module/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -a -tags netgo \
      -ldflags '-w -extldflags "-static"' \
      -mod vendor \
      -o env

FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
COPY --from=builder /module/env .
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENTRYPOINT ["/env"]
