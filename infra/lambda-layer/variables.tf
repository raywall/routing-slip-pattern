variable "aws_region" {
  description = "Regiao AWS onde a layer sera publicada."
  type        = string
  default     = "us-east-1"
}

variable "layer_name" {
  description = "Nome da Lambda Layer do routing-slip-pattern."
  type        = string
  default     = "routing-slip-pattern-framework"
}

variable "layer_description" {
  description = "Descricao da Lambda Layer."
  type        = string
  default     = "Routing Slip Pattern framework assets, configuration and workflow templates."
}

variable "layer_source_dir" {
  description = "Diretorio local que sera empacotado dentro da layer. O conteudo fica disponivel em /opt na Lambda."
  type        = string
  default     = null
}

variable "compatible_runtimes" {
  description = "Runtimes compativeis com a layer."
  type        = list(string)
  default     = ["provided.al2023"]
}

variable "compatible_architectures" {
  description = "Arquiteturas compativeis com a layer."
  type        = list(string)
  default     = ["arm64", "x86_64"]
}

variable "extra_layer_arns" {
  description = "ARNs de layers adicionais para anexar a Lambda exemplo. Preencha depois de criar dependencias internas, extensoes ou certificados."
  type        = list(string)
  default     = []
}

variable "create_example_lambda" {
  description = "Cria uma Lambda exemplo anexando a layer publicada."
  type        = bool
  default     = false
}

variable "example_lambda_name" {
  description = "Nome da Lambda exemplo."
  type        = string
  default     = "routing-slip-pattern-example"
}

variable "example_lambda_package_file" {
  description = "Arquivo .zip da Lambda Go compilada com bootstrap. Obrigatorio quando create_example_lambda=true."
  type        = string
  default     = ""
}

variable "example_lambda_role_arn" {
  description = "Role existente para a Lambda exemplo. Se vazio e create_example_lambda=true, uma role minima sera criada."
  type        = string
  default     = ""
}

variable "example_lambda_timeout" {
  description = "Timeout da Lambda exemplo em segundos."
  type        = number
  default     = 30
}

variable "example_lambda_memory_size" {
  description = "Memoria da Lambda exemplo em MB."
  type        = number
  default     = 256
}

variable "example_lambda_environment" {
  description = "Variaveis de ambiente adicionais da Lambda exemplo."
  type        = map(string)
  default     = {}
}
