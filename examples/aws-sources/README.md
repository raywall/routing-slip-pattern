# AWS configuration sources

`source.AWS` supports `s3`, `secretsmanager`, `ssm` and `dynamodb`. Configure
`Region`, resource identifiers and, when testing with LocalStack, `Endpoint`.

Both runtime configuration and workflow origins are selected during
`framework.New`. Referenced workflows use the same origin and resolve relative
to the parent key/name.
