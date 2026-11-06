# O que acontece com os avisos depois de sair, só com var.email_de: o SES publica devolução
# (bounce), reclamação e entrega num tópico do SNS, que entrega por HTTPS à API
# (POST /api/ses/eventos). A API só aceita mensagem assinada pelo SNS deste tópico, grava a
# ocorrência e suprime o endereço que devolveu de vez ou reclamou.
#
# A identidade do remetente (o domínio, com DKIM) não está aqui: verificar um domínio pede registros
# no DNS dele, que é de quem tem o domínio.

locals {
  com_ses = var.email_de != ""
}

resource "aws_sesv2_configuration_set" "avisos" {
  count                  = local.com_ses ? 1 : 0
  configuration_set_name = "${var.nome}-avisos"
  reputation_options {
    reputation_metrics_enabled = true # taxa de devolução e de reclamação no CloudWatch
  }
}

resource "aws_sns_topic" "eventos_ses" {
  count = local.com_ses ? 1 : 0
  name  = "${var.nome}-eventos-ses"
}

data "aws_iam_policy_document" "eventos_ses" {
  count = local.com_ses ? 1 : 0
  statement {
    actions   = ["sns:Publish"]
    resources = [aws_sns_topic.eventos_ses[0].arn]
    principals {
      type        = "Service"
      identifiers = ["ses.amazonaws.com"]
    }
    condition {
      test     = "StringEquals"
      variable = "AWS:SourceAccount"
      values   = [data.aws_caller_identity.conta.account_id]
    }
  }
}

resource "aws_sns_topic_policy" "eventos_ses" {
  count  = local.com_ses ? 1 : 0
  arn    = aws_sns_topic.eventos_ses[0].arn
  policy = data.aws_iam_policy_document.eventos_ses[0].json
}

resource "aws_sesv2_configuration_set_event_destination" "avisos" {
  count                  = local.com_ses ? 1 : 0
  configuration_set_name = aws_sesv2_configuration_set.avisos[0].configuration_set_name
  event_destination_name = "sns"
  event_destination {
    enabled              = true
    matching_event_types = ["BOUNCE", "COMPLAINT", "DELIVERY"]
    sns_destination {
      topic_arn = aws_sns_topic.eventos_ses[0].arn
    }
  }
  depends_on = [aws_sns_topic_policy.eventos_ses]
}

# A inscrição da API no tópico. O SNS manda a confirmação ao endereço, e a própria API a confirma:
# por isso só existe com a API no ar (var.ligado) e depois do serviço dela. Se a API ainda não
# respondia quando a confirmação chegou, o apply espera o prazo e falha; rodar o apply de novo manda
# outra confirmação.
resource "aws_sns_topic_subscription" "eventos_ses" {
  count                           = local.com_ses && var.ligado ? 1 : 0
  topic_arn                       = aws_sns_topic.eventos_ses[0].arn
  protocol                        = "https"
  endpoint                        = "https://${aws_cloudfront_distribution.entrada.domain_name}/api/ses/eventos"
  endpoint_auto_confirms          = true
  confirmation_timeout_in_minutes = 5
  depends_on                      = [aws_ecs_service.api]
}
