# Rede: duas zonas. Sub-redes públicas para o balanceador e as tarefas (com IP público e grupo de
# segurança que só aceita o balanceador — assim não há NAT Gateway cobrando por hora parado);
# sub-redes privadas, sem saída para a internet, para o banco.

data "aws_availability_zones" "zonas" {
  state = "available"
}

locals {
  zonas = slice(data.aws_availability_zones.zonas.names, 0, 2)
}

resource "aws_vpc" "principal" {
  cidr_block           = "10.40.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true
  tags                 = { Name = var.nome }
}

resource "aws_internet_gateway" "saida" {
  vpc_id = aws_vpc.principal.id
  tags   = { Name = var.nome }
}

resource "aws_subnet" "publica" {
  count             = 2
  vpc_id            = aws_vpc.principal.id
  cidr_block        = cidrsubnet(aws_vpc.principal.cidr_block, 8, count.index)
  availability_zone = local.zonas[count.index]
  tags              = { Name = "${var.nome}-publica-${count.index}" }
}

resource "aws_subnet" "privada" {
  count             = 2
  vpc_id            = aws_vpc.principal.id
  cidr_block        = cidrsubnet(aws_vpc.principal.cidr_block, 8, 10 + count.index)
  availability_zone = local.zonas[count.index]
  tags              = { Name = "${var.nome}-privada-${count.index}" }
}

resource "aws_route_table" "publica" {
  vpc_id = aws_vpc.principal.id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.saida.id
  }
  tags = { Name = "${var.nome}-publica" }
}

resource "aws_route_table_association" "publica" {
  count          = 2
  subnet_id      = aws_subnet.publica[count.index].id
  route_table_id = aws_route_table.publica.id
}

# As privadas ficam com a tabela principal da VPC, que só tem a rota local: o banco não sai.

# ---- grupos de segurança ----

# Só o CloudFront fala com o balanceador (lista de endereços mantida pela AWS); um cabeçalho secreto,
# conferido no balanceador, barra quem tentar entrar por outra distribuição CloudFront.
data "aws_ec2_managed_prefix_list" "cloudfront" {
  name = "com.amazonaws.global.cloudfront.origin-facing"
}

resource "aws_security_group" "balanceador" {
  name        = "${var.nome}-balanceador"
  description = "Entrada do CloudFront"
  vpc_id      = aws_vpc.principal.id
}

resource "aws_vpc_security_group_ingress_rule" "balanceador_cloudfront" {
  security_group_id = aws_security_group.balanceador.id
  prefix_list_id    = data.aws_ec2_managed_prefix_list.cloudfront.id
  ip_protocol       = "tcp"
  from_port         = 80
  to_port           = 80
}

resource "aws_vpc_security_group_egress_rule" "balanceador_tarefas" {
  security_group_id            = aws_security_group.balanceador.id
  referenced_security_group_id = aws_security_group.tarefas.id
  ip_protocol                  = "tcp"
  from_port                    = 8080
  to_port                      = 8082
}

resource "aws_security_group" "tarefas" {
  name        = "${var.nome}-tarefas"
  description = "Tarefas do ECS: API, front, PSP simulado, k6"
  vpc_id      = aws_vpc.principal.id
}

resource "aws_vpc_security_group_ingress_rule" "tarefas_do_balanceador" {
  security_group_id            = aws_security_group.tarefas.id
  referenced_security_group_id = aws_security_group.balanceador.id
  ip_protocol                  = "tcp"
  from_port                    = 8080
  to_port                      = 8082
}

# API ↔ PSP simulado (cobrança e aviso de pagamento), pelo DNS interno do Cloud Map.
resource "aws_vpc_security_group_ingress_rule" "tarefas_entre_si" {
  security_group_id            = aws_security_group.tarefas.id
  referenced_security_group_id = aws_security_group.tarefas.id
  ip_protocol                  = "tcp"
  from_port                    = 8081
  to_port                      = 8082
}

# Saída livre: baixar imagens do ECR, falar com S3, Secrets Manager, CloudWatch e (k6) o CloudFront.
resource "aws_vpc_security_group_egress_rule" "tarefas_saida" {
  security_group_id = aws_security_group.tarefas.id
  cidr_ipv4         = "0.0.0.0/0"
  ip_protocol       = "-1"
}

resource "aws_security_group" "banco" {
  name        = "${var.nome}-banco"
  description = "Aurora: so as tarefas"
  vpc_id      = aws_vpc.principal.id
}

resource "aws_vpc_security_group_ingress_rule" "banco_das_tarefas" {
  security_group_id            = aws_security_group.banco.id
  referenced_security_group_id = aws_security_group.tarefas.id
  ip_protocol                  = "tcp"
  from_port                    = 5432
  to_port                      = 5432
}
