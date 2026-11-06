# Observar o sistema mudando na AWS, durante o ensaio: o mesmo papel do Grafana local
# (observabilidade/ no repositório), com o que a AWS mede sozinha. Tarefas no ar por serviço (o
# servidor escalando ou caindo), CPU e memória, o tráfego e os erros no balanceador, o tempo de
# resposta, e a capacidade do Aurora Serverless subindo (ACU). Atualiza a cada minuto (métricas
# padrão) ou a cada 10 s nas que o Container Insights publica.
#
# As métricas da aplicação (fila, sala de espera, conexões do pool) estão em GET /metrics de cada
# tarefa, dentro da VPC. Levá-las a um Prometheus e Grafana gerenciados (Amazon Managed Prometheus e
# Managed Grafana) ainda não está aqui.

resource "aws_cloudwatch_dashboard" "operacao" {
  dashboard_name = "${var.nome}-operacao"
  dashboard_body = jsonencode({
    widgets = [
      {
        type = "metric", x = 0, y = 0, width = 8, height = 6
        properties = {
          title  = "Tarefas no ar, por serviço (escala)"
          region = data.aws_region.atual.region
          stat   = "Average", period = 60, view = "timeSeries"
          metrics = [for s in ["api", "web", "psp"] :
          ["ECS/ContainerInsights", "RunningTaskCount", "ClusterName", aws_ecs_cluster.principal.name, "ServiceName", s]]
        }
      },
      {
        type = "metric", x = 8, y = 0, width = 8, height = 6
        properties = {
          title  = "CPU por serviço (%)"
          region = data.aws_region.atual.region
          stat   = "Average", period = 60, view = "timeSeries"
          metrics = [for s in ["api", "web"] :
          ["AWS/ECS", "CPUUtilization", "ClusterName", aws_ecs_cluster.principal.name, "ServiceName", s]]
        }
      },
      {
        type = "metric", x = 16, y = 0, width = 8, height = 6
        properties = {
          title  = "Memória por serviço (%)"
          region = data.aws_region.atual.region
          stat   = "Average", period = 60, view = "timeSeries"
          metrics = [for s in ["api", "web"] :
          ["AWS/ECS", "MemoryUtilization", "ClusterName", aws_ecs_cluster.principal.name, "ServiceName", s]]
        }
      },
      {
        type = "metric", x = 0, y = 6, width = 8, height = 6
        properties = {
          title  = "Requisições e erros 5xx no balanceador"
          region = data.aws_region.atual.region
          stat   = "Sum", period = 60, view = "timeSeries"
          metrics = [
            ["AWS/ApplicationELB", "RequestCount", "LoadBalancer", aws_lb.entrada.arn_suffix],
            ["AWS/ApplicationELB", "HTTPCode_Target_5XX_Count", "LoadBalancer", aws_lb.entrada.arn_suffix],
            ["AWS/ApplicationELB", "HTTPCode_ELB_5XX_Count", "LoadBalancer", aws_lb.entrada.arn_suffix],
          ]
        }
      },
      {
        type = "metric", x = 8, y = 6, width = 8, height = 6
        properties = {
          title       = "Tempo de resposta da API, p95 (s) — o H1 pede < 2 s"
          region      = data.aws_region.atual.region
          stat        = "p95", period = 60, view = "timeSeries"
          metrics     = [["AWS/ApplicationELB", "TargetResponseTime", "TargetGroup", aws_lb_target_group.api.arn_suffix, "LoadBalancer", aws_lb.entrada.arn_suffix]]
          annotations = { horizontal = [{ label = "H1", value = 2 }] }
        }
      },
      {
        type = "metric", x = 16, y = 6, width = 8, height = 6
        properties = {
          title  = "Aurora: capacidade (ACU) e conexões"
          region = data.aws_region.atual.region
          stat   = "Average", period = 60, view = "timeSeries"
          metrics = [
            ["AWS/RDS", "ServerlessDatabaseCapacity", "DBClusterIdentifier", aws_rds_cluster.banco.cluster_identifier],
            ["AWS/RDS", "DatabaseConnections", "DBClusterIdentifier", aws_rds_cluster.banco.cluster_identifier, { yAxis = "right" }],
          ]
        }
      },
    ]
  })
}
