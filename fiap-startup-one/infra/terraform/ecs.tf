# ECS Fargate: API, front e PSP simulado como serviços; seed, auditoria e k6 como tarefas avulsas.
# (A arquitetura alvo usa EKS; ver infra/README.md pelo porquê de Fargate na POC.)

resource "aws_ecs_cluster" "principal" {
  name = var.nome
  setting {
    name  = "containerInsights"
    value = "enabled" # CPU, memória e rede por tarefa: entra no relatório do ensaio
  }
}

resource "aws_ecs_cluster_capacity_providers" "principal" {
  cluster_name       = aws_ecs_cluster.principal.name
  capacity_providers = ["FARGATE", "FARGATE_SPOT"]
  default_capacity_provider_strategy {
    capacity_provider = "FARGATE"
    weight            = 1
  }
}

# DNS interno: a API acha o PSP simulado em psp.lisura.local e o PSP avisa em api.lisura.local.
resource "aws_service_discovery_private_dns_namespace" "interno" {
  name = "lisura.local"
  vpc  = aws_vpc.principal.id
}

resource "aws_service_discovery_service" "servico" {
  for_each = toset(["api", "psp"])
  name     = each.key
  dns_config {
    namespace_id   = aws_service_discovery_private_dns_namespace.interno.id
    routing_policy = "MULTIVALUE"
    dns_records {
      type = "A"
      ttl  = 10
    }
  }
}

resource "aws_cloudwatch_log_group" "tarefa" {
  for_each          = toset(["api", "web", "psp", "seed", "auditoria", "k6"])
  name              = "/${var.nome}/${each.key}"
  retention_in_days = 7
}

# ---- papéis ----

data "aws_iam_policy_document" "ecs_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

# Execução: o ECS baixa a imagem, escreve o log e entrega os segredos à tarefa.
resource "aws_iam_role" "execucao" {
  name               = "${var.nome}-execucao"
  assume_role_policy = data.aws_iam_policy_document.ecs_assume.json
}

resource "aws_iam_role_policy_attachment" "execucao" {
  role       = aws_iam_role.execucao.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

data "aws_iam_policy_document" "execucao_segredos" {
  statement {
    actions   = ["secretsmanager:GetSecretValue"]
    resources = [local.segredo_app, local.segredo_banco]
  }
}

resource "aws_iam_role_policy" "execucao_segredos" {
  role   = aws_iam_role.execucao.id
  policy = data.aws_iam_policy_document.execucao_segredos.json
}

# A API: grava publicação e documentos no bucket; não lê a fonte nem nada fora dele.
resource "aws_iam_role" "api" {
  name               = "${var.nome}-api"
  assume_role_policy = data.aws_iam_policy_document.ecs_assume.json
}

data "aws_iam_policy_document" "api" {
  statement {
    actions = ["s3:PutObject", "s3:GetObject"]
    resources = [
      "${aws_s3_bucket.arquivos.arn}/publico/*",
      "${aws_s3_bucket.arquivos.arn}/documentos/*",
    ]
  }
  statement {
    actions   = ["s3:ListBucket"] # HeadBucket na subida; e HeadObject devolve 404 (não 403) quando falta
    resources = [aws_s3_bucket.arquivos.arn]
  }
  # Avisos pelo SES, só com var.email_de: enviar do remetente verificado, e ler a cota na subida.
  dynamic "statement" {
    for_each = var.email_de == "" ? [] : [1]
    content {
      actions = ["ses:SendEmail"]
      # A identidade verificada pode ser o endereço ou o domínio dele; e o envio pelo conjunto de
      # configuração também é autorizado no conjunto.
      resources = concat([for i in [var.email_de, element(split("@", var.email_de), 1)] :
        "arn:aws:ses:${data.aws_region.atual.region}:${data.aws_caller_identity.conta.account_id}:identity/${i}"],
      aws_sesv2_configuration_set.avisos[*].arn)
    }
  }
  dynamic "statement" {
    for_each = var.email_de == "" ? [] : [1]
    content {
      actions   = ["ses:GetAccount"]
      resources = ["*"] # a ação não aceita recurso
    }
  }
}

resource "aws_iam_role_policy" "api" {
  role   = aws_iam_role.api.id
  policy = data.aws_iam_policy_document.api.json
}

# PSP, front e k6 não tocam em nada da AWS.
resource "aws_iam_role" "sem_acesso" {
  name               = "${var.nome}-sem-acesso"
  assume_role_policy = data.aws_iam_policy_document.ecs_assume.json
}

# ---- definições de tarefa ----

locals {
  img_go  = "${aws_ecr_repository.imagem["go"].repository_url}:${var.tag_imagem}"
  img_web = "${aws_ecr_repository.imagem["web"].repository_url}:${var.tag_imagem}"
  img_k6  = "${aws_ecr_repository.imagem["k6"].repository_url}:${var.tag_imagem}"

  log = { for k, g in aws_cloudwatch_log_group.tarefa : k => {
    logDriver = "awslogs"
    options = {
      "awslogs-group"         = g.name
      "awslogs-region"        = data.aws_region.atual.region
      "awslogs-stream-prefix" = k
    }
  } }

  # Banco: o pgx lê PGHOST/PGUSER/PGPASSWORD do ambiente e o resto desta linha.
  banco_env = [
    { name = "PGHOST", value = aws_rds_cluster.banco.endpoint },
    { name = "PGUSER", value = aws_rds_cluster.banco.master_username },
    { name = "DATABASE_URL", value = "dbname=lisura sslmode=require pool_max_conns=${var.api_pool_conexoes}" },
  ]
  banco_segredos = [{ name = "PGPASSWORD", valueFrom = "${local.segredo_banco}:password::" }]
}

resource "aws_ecs_task_definition" "api" {
  family                   = "${var.nome}-api"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.api_cpu
  memory                   = var.api_memoria
  execution_role_arn       = aws_iam_role.execucao.arn
  task_role_arn            = aws_iam_role.api.arn
  runtime_platform {
    operating_system_family = "LINUX"
    cpu_architecture        = "X86_64"
  }
  container_definitions = jsonencode([{
    name         = "api"
    image        = local.img_go
    command      = ["/lisura"]
    essential    = true
    portMappings = [{ containerPort = 8082, protocol = "tcp" }]
    environment = concat(local.banco_env, [
      { name = "LISURA_ADDR", value = ":8082" },
      { name = "LISURA_S3_BUCKET", value = aws_s3_bucket.arquivos.id },
      { name = "LISURA_COFRE_BUCKET", value = aws_s3_bucket.cofre.id },
      { name = "LISURA_PSP_URL", value = "http://psp.lisura.local:8081" },
      { name = "LISURA_SANDBOX", value = "1" }, # PSP simulado: a rota de "pagar no banco" fica ligada
      { name = "LISURA_TRABALHADORES", value = tostring(var.api_trabalhadores) },
      # Os avisos ao candidato passam pela fila como em produção. Sem var.email_de, o provedor aceita
      # e conta: o ensaio mede a fila com eles dentro, sem um provedor de e-mail de verdade no caminho.
      { name = "LISURA_EMAIL", value = var.email_de == "" ? "descarte" : "ses" },
      { name = "LISURA_SES_DE", value = var.email_de },
      # Vazios sem var.email_de (as listas têm zero ou um elemento).
      { name = "LISURA_SES_CONJUNTO", value = join("", aws_sesv2_configuration_set.avisos[*].configuration_set_name) },
      { name = "LISURA_SES_TOPICO", value = join("", aws_sns_topic.eventos_ses[*].arn) },
    ])
    secrets = concat(local.banco_segredos, [
      { name = "LISURA_TOKEN_KEY", valueFrom = "${local.segredo_app}:token_key::" },
      { name = "LISURA_OPERACAO_SEGREDO", valueFrom = "${local.segredo_app}:operacao::" },
      { name = "LISURA_PSP_SEGREDO", valueFrom = "${local.segredo_app}:psp::" },
    ])
    logConfiguration = local.log["api"]
  }])
}

resource "aws_ecs_task_definition" "psp" {
  family                   = "${var.nome}-psp"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 512
  memory                   = 1024
  execution_role_arn       = aws_iam_role.execucao.arn
  task_role_arn            = aws_iam_role.sem_acesso.arn
  runtime_platform {
    operating_system_family = "LINUX"
    cpu_architecture        = "X86_64"
  }
  container_definitions = jsonencode([{
    name             = "psp"
    image            = local.img_go
    command          = ["/psp-simulado", "-addr", ":8081", "-aviso", "http://api.lisura.local:8082/api/psp/aviso", "-repetir", tostring(var.psp_repetir), "-perder", tostring(var.psp_perder)]
    essential        = true
    portMappings     = [{ containerPort = 8081, protocol = "tcp" }]
    secrets          = [{ name = "LISURA_PSP_SEGREDO", valueFrom = "${local.segredo_app}:psp::" }]
    logConfiguration = local.log["psp"]
  }])
}

resource "aws_ecs_task_definition" "web" {
  family                   = "${var.nome}-web"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 512
  memory                   = 1024
  execution_role_arn       = aws_iam_role.execucao.arn
  task_role_arn            = aws_iam_role.sem_acesso.arn
  runtime_platform {
    operating_system_family = "LINUX"
    cpu_architecture        = "X86_64"
  }
  container_definitions = jsonencode([{
    name             = "web"
    image            = local.img_web
    essential        = true
    portMappings     = [{ containerPort = 8080, protocol = "tcp" }]
    logConfiguration = local.log["web"]
  }])
}

# Tarefas avulsas com a imagem Go: o comando é trocado no run-task (/lisura-seed, /lisura-auditoria).
resource "aws_ecs_task_definition" "ferramenta" {
  for_each                 = toset(["seed", "auditoria"])
  family                   = "${var.nome}-${each.key}"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 512
  memory                   = 1024
  execution_role_arn       = aws_iam_role.execucao.arn
  task_role_arn            = each.key == "auditoria" ? aws_iam_role.auditoria.arn : aws_iam_role.sem_acesso.arn
  runtime_platform {
    operating_system_family = "LINUX"
    cpu_architecture        = "X86_64"
  }
  container_definitions = jsonencode([{
    name      = each.key
    image     = local.img_go
    command   = ["/lisura-${each.key}"]
    essential = true
    environment = concat(local.banco_env, each.key == "auditoria" ? [
      { name = "LISURA_COFRE_BUCKET", value = aws_s3_bucket.cofre.id },
    ] : [])
    secrets          = local.banco_segredos
    logConfiguration = local.log[each.key]
  }])
}

resource "aws_ecs_task_definition" "k6" {
  family                   = "${var.nome}-k6"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.k6_cpu
  memory                   = var.k6_memoria
  execution_role_arn       = aws_iam_role.execucao.arn
  task_role_arn            = aws_iam_role.sem_acesso.arn
  runtime_platform {
    operating_system_family = "LINUX"
    cpu_architecture        = "X86_64"
  }
  container_definitions = jsonencode([{
    name      = "k6"
    image     = local.img_k6
    command   = ["run", "ultimo-dia.js"]
    essential = true
    # O k6 entra pela mesma porta que o candidato: o CloudFront.
    environment = [
      { name = "BASE_URL", value = "https://${aws_cloudfront_distribution.entrada.domain_name}" },
    ]
    secrets          = [{ name = "LISURA_OPERACAO_SEGREDO", valueFrom = "${local.segredo_app}:operacao::" }]
    ulimits          = [{ name = "nofile", softLimit = 65535, hardLimit = 65535 }]
    logConfiguration = local.log["k6"]
  }])
}

# ---- serviços ----

locals {
  rede_tarefas = {
    subnets          = aws_subnet.publica[*].id
    security_groups  = [aws_security_group.tarefas.id]
    assign_public_ip = true # sem NAT: a saída para ECR/S3/CloudWatch é pelo IP público da tarefa
  }
}

resource "aws_ecs_service" "api" {
  name                              = "api"
  cluster                           = aws_ecs_cluster.principal.id
  task_definition                   = aws_ecs_task_definition.api.arn
  desired_count                     = var.ligado ? var.api_tarefas : 0
  launch_type                       = "FARGATE"
  health_check_grace_period_seconds = 30
  network_configuration {
    subnets          = local.rede_tarefas.subnets
    security_groups  = local.rede_tarefas.security_groups
    assign_public_ip = local.rede_tarefas.assign_public_ip
  }
  load_balancer {
    target_group_arn = aws_lb_target_group.api.arn
    container_name   = "api"
    container_port   = 8082
  }
  service_registries {
    registry_arn = aws_service_discovery_service.servico["api"].arn
  }
  depends_on = [aws_lb_listener_rule.api]
}

resource "aws_ecs_service" "psp" {
  name            = "psp"
  cluster         = aws_ecs_cluster.principal.id
  task_definition = aws_ecs_task_definition.psp.arn
  desired_count   = var.ligado ? 1 : 0
  launch_type     = "FARGATE"
  network_configuration {
    subnets          = local.rede_tarefas.subnets
    security_groups  = local.rede_tarefas.security_groups
    assign_public_ip = local.rede_tarefas.assign_public_ip
  }
  service_registries {
    registry_arn = aws_service_discovery_service.servico["psp"].arn
  }
}

resource "aws_ecs_service" "web" {
  name                              = "web"
  cluster                           = aws_ecs_cluster.principal.id
  task_definition                   = aws_ecs_task_definition.web.arn
  desired_count                     = var.ligado ? var.web_tarefas : 0
  launch_type                       = "FARGATE"
  health_check_grace_period_seconds = 30
  network_configuration {
    subnets          = local.rede_tarefas.subnets
    security_groups  = local.rede_tarefas.security_groups
    assign_public_ip = local.rede_tarefas.assign_public_ip
  }
  load_balancer {
    target_group_arn = aws_lb_target_group.web.arn
    container_name   = "web"
    container_port   = 8080
  }
  depends_on = [aws_lb_listener_rule.web]
}
