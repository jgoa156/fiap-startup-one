# O cofre das âncoras da trilha (poc/internal/ancora): um bucket à parte, com bloqueio de objeto
# (Object Lock). Cada âncora é gravada uma vez e fica retida: nem a API nem quem tem o banco
# consegue trocá-la ou apagá-la antes do prazo.
#
# Modo e prazo: no ensaio, GOVERNANCE por 1 dia — os dados são sintéticos e o ambiente precisa poder
# ser destruído (quem tem s3:BypassGovernanceRetention apaga). Em produção, COMPLIANCE pelo prazo de
# guarda do certame: nesse modo ninguém apaga antes do fim do prazo, nem o usuário raiz da conta, e
# o `terraform destroy` falha até lá. Por isso o padrão NÃO é o de produção.

variable "cofre_modo" {
  description = "Modo do bloqueio de objeto do cofre: GOVERNANCE (ensaio) ou COMPLIANCE (produção)."
  type        = string
  default     = "GOVERNANCE"
  validation {
    condition     = contains(["GOVERNANCE", "COMPLIANCE"], var.cofre_modo)
    error_message = "cofre_modo é GOVERNANCE ou COMPLIANCE."
  }
}

variable "cofre_dias" {
  description = "Dias de retenção de cada âncora."
  type        = number
  default     = 1
}

resource "aws_s3_bucket" "cofre" {
  bucket              = "${var.nome}-${data.aws_caller_identity.conta.account_id}-cofre"
  object_lock_enabled = true # só pode ser ligado na criação do bucket; liga também o versionamento
  force_destroy       = var.cofre_modo == "GOVERNANCE"
}

resource "aws_s3_bucket_versioning" "cofre" {
  bucket = aws_s3_bucket.cofre.id
  versioning_configuration {
    status = "Enabled" # exigido pelo bloqueio de objeto
  }
}

resource "aws_s3_bucket_object_lock_configuration" "cofre" {
  bucket = aws_s3_bucket.cofre.id
  rule {
    default_retention {
      mode = var.cofre_modo
      days = var.cofre_dias
    }
  }
  depends_on = [aws_s3_bucket_versioning.cofre]
}

resource "aws_s3_bucket_public_access_block" "cofre" {
  bucket                  = aws_s3_bucket.cofre.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "cofre" {
  bucket = aws_s3_bucket.cofre.id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "cofre" {
  bucket = aws_s3_bucket.cofre.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

data "aws_iam_policy_document" "cofre" {
  statement {
    sid       = "SoHTTPS"
    effect    = "Deny"
    actions   = ["s3:*"]
    resources = [aws_s3_bucket.cofre.arn, "${aws_s3_bucket.cofre.arn}/*"]
    principals {
      type        = "*"
      identifiers = ["*"]
    }
    condition {
      test     = "Bool"
      variable = "aws:SecureTransport"
      values   = ["false"]
    }
  }
}

resource "aws_s3_bucket_policy" "cofre" {
  bucket     = aws_s3_bucket.cofre.id
  policy     = data.aws_iam_policy_document.cofre.json
  depends_on = [aws_s3_bucket_public_access_block.cofre]
}

# A API grava e lê âncoras; nunca apaga, nunca mexe na retenção. A negação explícita vale mesmo que
# alguém amplie a permissão de gravação depois.
data "aws_iam_policy_document" "api_cofre" {
  statement {
    actions   = ["s3:PutObject", "s3:GetObject"]
    resources = ["${aws_s3_bucket.cofre.arn}/ancoras/*"]
  }
  statement {
    actions   = ["s3:ListBucket"]
    resources = [aws_s3_bucket.cofre.arn]
  }
  statement {
    sid    = "NuncaApagaNemAfrouxa"
    effect = "Deny"
    actions = [
      "s3:DeleteObject", "s3:DeleteObjectVersion", "s3:BypassGovernanceRetention",
      "s3:PutObjectRetention", "s3:PutObjectLegalHold", "s3:PutBucketObjectLockConfiguration",
      "s3:PutBucketVersioning", "s3:DeleteBucket", "s3:PutBucketPolicy",
    ]
    resources = [aws_s3_bucket.cofre.arn, "${aws_s3_bucket.cofre.arn}/*"]
  }
}

resource "aws_iam_role_policy" "api_cofre" {
  role   = aws_iam_role.api.id
  policy = data.aws_iam_policy_document.api_cofre.json
}

# A auditoria só lê o cofre.
resource "aws_iam_role" "auditoria" {
  name               = "${var.nome}-auditoria"
  assume_role_policy = data.aws_iam_policy_document.ecs_assume.json
}

data "aws_iam_policy_document" "auditoria" {
  statement {
    actions   = ["s3:GetObject"]
    resources = ["${aws_s3_bucket.cofre.arn}/ancoras/*"]
  }
  statement {
    actions   = ["s3:ListBucket"]
    resources = [aws_s3_bucket.cofre.arn]
  }
}

resource "aws_iam_role_policy" "auditoria" {
  role   = aws_iam_role.auditoria.id
  policy = data.aws_iam_policy_document.auditoria.json
}

output "cofre" {
  value = aws_s3_bucket.cofre.id
}
