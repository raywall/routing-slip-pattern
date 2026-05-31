output "routing_slip_layer_arn" {
  description = "ARN base da Lambda Layer publicada."
  value       = aws_lambda_layer_version.routing_slip.layer_arn
}

output "routing_slip_layer_version_arn" {
  description = "ARN versionado da Lambda Layer publicada."
  value       = aws_lambda_layer_version.routing_slip.arn
}

output "routing_slip_layer_version" {
  description = "Versao numerica da Lambda Layer publicada."
  value       = aws_lambda_layer_version.routing_slip.version
}

output "example_lambda_arn" {
  description = "ARN da Lambda exemplo, quando criada."
  value       = try(aws_lambda_function.example[0].arn, null)
}
