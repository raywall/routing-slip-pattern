FROM golang:1.26-alpine AS build

WORKDIR /src/app

RUN apk add --no-cache ca-certificates git && update-ca-certificates

COPY ./app .
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/routing-slip .

# Corporate CA notes:
# - Alpine trusts certificates installed under /usr/local/share/ca-certificates.
# - To enable an internal CA, copy a .crt file to that directory and run
#   update-ca-certificates.
# - For AWS SDK/CLI calls that need a custom bundle, set:
#   ENV AWS_CA_BUNDLE=/usr/local/share/ca-certificates/internal-ca.crt
# - Example:
#   COPY certs/internal-ca.crt /usr/local/share/ca-certificates/internal-ca.crt
#   RUN update-ca-certificates
FROM alpine:3.22

WORKDIR /opt/routing-slip

RUN apk add --no-cache ca-certificates curl && update-ca-certificates
RUN mkdir -p /opt/routing-slip/config

ENV ROUTING_SLIP_CONFIG_PATH=/opt/routing-slip/config/config.yaml
ENV SELECTED_WORKFLOW=payment-fulfillment.yaml

COPY --from=build /out/routing-slip /usr/local/bin/routing-slip
COPY config.yaml /opt/routing-slip/config/config.yaml
COPY workflows /opt/routing-slip/workflows

EXPOSE 8088 9091

CMD ["/usr/local/bin/routing-slip"]
