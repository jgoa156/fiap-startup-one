variable "regiao" {
  description = "Região AWS. São Paulo, como na arquitetura alvo."
  type        = string
  default     = "sa-east-1"
}

variable "nome" {
  description = "Prefixo de todos os recursos."
  type        = string
  default     = "lisura-poc"
}

variable "tag_imagem" {
  description = "Tag das imagens no ECR (a que infra/scripts/publicar-imagens.ps1 publicou)."
  type        = string
  default     = "inicial"
}

# ---- ligar e desligar: "subir capacidade, medir e devolver" (04-escopo-poc §5) ----

variable "ligado" {
  description = "false deixa tudo com zero tarefas e o banco pausável: fora do ensaio, só armazenamento custa."
  type        = bool
  default     = false
}

# ---- aplicação ----

variable "api_tarefas" {
  description = "Instâncias da API atrás do balanceador."
  type        = number
  default     = 2
}

variable "api_cpu" {
  description = "CPU por tarefa da API, em unidades do ECS (1024 = 1 vCPU)."
  type        = number
  default     = 1024
}

variable "api_memoria" {
  description = "Memória por tarefa da API, em MiB."
  type        = number
  default     = 2048
}

variable "api_pool_conexoes" {
  description = "Conexões ao banco por tarefa da API. Total = api_tarefas × isto: tem de caber no max_connections do Aurora no ACU mínimo."
  type        = number
  default     = 20
}

variable "api_trabalhadores" {
  description = "Trabalhadores da fila de tarefas por instância da API."
  type        = number
  default     = 4
}

variable "email_de" {
  description = "Remetente dos avisos, verificado no SES desta região (o endereço ou o domínio dele). Vazio: o ensaio, com o provedor que aceita e conta."
  type        = string
  default     = ""
  validation {
    condition     = var.email_de == "" || can(regex("^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$", var.email_de))
    error_message = "email_de é um endereço de e-mail, ou vazio."
  }
}

variable "web_tarefas" {
  description = "Instâncias do front Next."
  type        = number
  default     = 2
}

variable "psp_repetir" {
  description = "PSP simulado: quantas vezes manda cada aviso de pagamento (aviso repetido de propósito)."
  type        = number
  default     = 2
}

variable "psp_perder" {
  description = "PSP simulado: chance de um aviso nunca chegar (a conciliação tem de recuperar)."
  type        = number
  default     = 0.02
}

# ---- banco ----

variable "banco_acu_min" {
  description = "Capacidade mínima do Aurora Serverless v2. 0 = pausa sozinho sem uso. No ensaio, subir ANTES (o Serverless sozinho sobe tarde num pico de 90 s)."
  type        = number
  default     = 0
}

variable "banco_acu_max" {
  description = "Capacidade máxima do Aurora Serverless v2."
  type        = number
  default     = 16
}

# ---- gerador de carga ----

variable "k6_cpu" {
  description = "CPU da tarefa do k6 (4096 = 4 vCPU)."
  type        = number
  default     = 4096
}

variable "k6_memoria" {
  description = "Memória da tarefa do k6, em MiB."
  type        = number
  default     = 8192
}
