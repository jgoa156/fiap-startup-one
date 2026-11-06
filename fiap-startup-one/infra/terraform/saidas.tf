output "endereco" {
  description = "Onde o candidato (e o k6) entra."
  value       = "https://${aws_cloudfront_distribution.entrada.domain_name}"
}

output "cluster" {
  value = aws_ecs_cluster.principal.name
}

output "painel_operacao" {
  description = "O painel do CloudWatch para acompanhar o ensaio: tarefas no ar, CPU, balanceador, p95 e o Aurora."
  value       = "https://${data.aws_region.atual.region}.console.aws.amazon.com/cloudwatch/home#dashboards/dashboard/${aws_cloudwatch_dashboard.operacao.dashboard_name}"
}

output "bucket" {
  value = aws_s3_bucket.arquivos.id
}

output "projeto_codebuild" {
  value = aws_codebuild_project.imagens.name
}

# Para `aws ecs run-task` (seed, auditoria, k6): sub-redes e grupo de segurança das tarefas.
output "rede_tarefas" {
  value = {
    subnets         = aws_subnet.publica[*].id
    security_groups = [aws_security_group.tarefas.id]
  }
}

output "segredo_app_arn" {
  description = "Onde está o segredo de operação (POST /api/operacao). O valor não sai aqui."
  value       = aws_secretsmanager_secret.app.arn
}
