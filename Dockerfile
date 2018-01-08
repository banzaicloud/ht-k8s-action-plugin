FROM golang:1.9-alpine

ADD . /go/src/github.com/banzaicloud/ht-k8s-action-plugin
WORKDIR /go/src/github.com/banzaicloud/ht-k8s-action-plugin
RUN go build -o /ht-k8s-action-plugin .

FROM alpine:3.6
RUN apk add --no-cache ca-certificates
COPY --from=0 /ht-k8s-action-plugin /
#ADD conf/plugin-config.toml /
ENTRYPOINT ["/ht-k8s-action-plugin"]
