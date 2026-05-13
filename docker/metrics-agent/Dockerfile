FROM golang:1.22-alpine AS build

# Corporate CA notes:
# - golang:1.22-alpine uses Alpine conventions.
# - Install ca-certificates, copy internal CAs to
#   /usr/local/share/ca-certificates, then run update-ca-certificates.
# - Use AWS_CA_BUNDLE=/usr/local/share/ca-certificates/internal-ca.crt when AWS
#   clients need to trust that CA.
RUN apk add --no-cache ca-certificates && update-ca-certificates

WORKDIR /src
COPY go.mod ./
COPY . .
RUN go build -o /out/agent ./cmd/agent

FROM alpine:3.20

# Corporate CA notes for runtime:
# - Same Alpine path: /usr/local/share/ca-certificates.
# - Add curl/openssl here if troubleshooting outbound TLS is needed.
RUN apk add --no-cache ca-certificates && update-ca-certificates

WORKDIR /app
COPY --from=build /out/agent /app/agent
EXPOSE 8125/udp
ENTRYPOINT ["/app/agent"]
