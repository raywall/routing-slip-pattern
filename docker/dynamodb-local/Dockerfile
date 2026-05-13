FROM amazon/dynamodb-local:2.5.2

# Corporate CA notes:
# - This image is used only for local DynamoDB emulation.
# - If it must call internal HTTPS endpoints in a customized setup, inspect the
#   base image package manager before installing CA tooling.
# - For Java truststores, an internal CA may need to be imported with keytool
#   into the JVM truststore instead of only using Linux CA paths.
# - For Linux CA bundles, prefer /usr/local/share/ca-certificates when available.
