# Segredos da aplicação, gerados aqui e entregues às tarefas pelo Secrets Manager (nunca como texto
# na definição da tarefa). Eles ficam no estado do Terraform: o estado é sensível e não vai para o
# repositório (ver infra/README.md).

resource "random_password" "token" {
  length  = 64
  special = false
}

resource "random_password" "operacao" {
  length  = 40
  special = false
}

resource "random_password" "psp" {
  length  = 40
  special = false
}

# Cabeçalho que o CloudFront manda ao balanceador; quem não manda, recebe 403.
resource "random_password" "origem" {
  length  = 40
  special = false
}

resource "aws_secretsmanager_secret" "app" {
  name                    = "${var.nome}/app"
  recovery_window_in_days = 0 # recriar o ambiente logo depois de destruir
}

resource "aws_secretsmanager_secret_version" "app" {
  secret_id = aws_secretsmanager_secret.app.id
  secret_string = jsonencode({
    token_key = random_password.token.result
    operacao  = random_password.operacao.result
    psp       = random_password.psp.result
  })
}

locals {
  segredo_app   = aws_secretsmanager_secret.app.arn
  segredo_banco = aws_rds_cluster.banco.master_user_secret[0].secret_arn
}
