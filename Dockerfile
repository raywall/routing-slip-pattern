FROM golang:1.25-alpine

# Corporate CA notes:
# - Alpine trusts certificates installed under /usr/local/share/ca-certificates.
# - To enable an internal CA, copy a .crt file to that directory and run
#   update-ca-certificates.
# - For AWS SDK/CLI calls that need a custom bundle, set:
#   ENV AWS_CA_BUNDLE=/usr/local/share/ca-certificates/internal-ca.crt
# - Example:
#   COPY certs/internal-ca.crt /usr/local/share/ca-certificates/internal-ca.crt
#   RUN update-ca-certificates
RUN apk add --no-cache ca-certificates curl && update-ca-certificates

ARG APP_CMD=.
ENV APP_CMD=${APP_CMD}
ENV GOCACHE=/workspace/.gocache

WORKDIR /workspace/app
EXPOSE 8088 8090 8091
CMD ["sh", "-c", "go run ${APP_CMD}"]
