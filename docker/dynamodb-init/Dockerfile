FROM amazon/aws-cli:2.17.57

# Corporate CA notes:
# - AWS CLI honors AWS_CA_BUNDLE.
# - Copy internal CA files to a stable path such as:
#   /usr/local/share/ca-certificates/internal-ca.crt
# - Then set:
#   ENV AWS_CA_BUNDLE=/usr/local/share/ca-certificates/internal-ca.crt
# - Depending on the aws-cli base image version, install/update CA packages with
#   the available package manager before running update-ca-certificates or the
#   equivalent command for the base OS.
