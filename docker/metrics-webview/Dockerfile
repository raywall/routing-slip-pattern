FROM nginx:1.27-alpine

# Corporate CA notes:
# - nginx:alpine also uses /usr/local/share/ca-certificates for internal CAs.
# - Install ca-certificates and run update-ca-certificates after copying .crt
#   files when this container needs to call internal HTTPS backends.
# - Example:
#   COPY certs/internal-ca.crt /usr/local/share/ca-certificates/internal-ca.crt
#   RUN update-ca-certificates
RUN apk add --no-cache ca-certificates && update-ca-certificates

COPY . /usr/share/nginx/html
EXPOSE 80
