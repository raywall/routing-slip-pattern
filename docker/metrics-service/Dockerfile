FROM golang:1.24-alpine AS build

# Corporate CA notes for the Go build image:
# - Alpine CA path: /usr/local/share/ca-certificates/*.crt
# - Install the CA bundle package with apk add --no-cache ca-certificates.
# - Copy internal CA files and run update-ca-certificates before go build if
#   private modules, proxies, or artifact registries use internal TLS.
# - If AWS calls use a custom CA, set AWS_CA_BUNDLE to the copied .crt path.
RUN apk add --no-cache ca-certificates && update-ca-certificates

WORKDIR /src
COPY go.mod ./
COPY . .
RUN go build -o /out/service ./cmd/service

FROM alpine:3.20

# Corporate CA notes for the runtime image:
# - Put internal CA files in /usr/local/share/ca-certificates.
# - Run update-ca-certificates after copying them.
# - Example:
#   COPY certs/internal-ca.crt /usr/local/share/ca-certificates/internal-ca.crt
#   RUN update-ca-certificates
#   ENV AWS_CA_BUNDLE=/usr/local/share/ca-certificates/internal-ca.crt
RUN apk add --no-cache ca-certificates && update-ca-certificates

WORKDIR /app
COPY --from=build /out/service /app/service
EXPOSE 8080
ENTRYPOINT ["/app/service"]
