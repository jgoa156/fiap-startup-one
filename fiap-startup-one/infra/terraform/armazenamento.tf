# Um bucket, três prefixos:
#   publico/     cartões, relação e dados do certame — lidos SÓ pelo CloudFront
#   documentos/  laudos e comprovantes — só a API lê e grava; nunca públicos
#   fonte/       o zip que o CodeBuild constrói

resource "aws_s3_bucket" "arquivos" {
  bucket        = "${var.nome}-${data.aws_caller_identity.conta.account_id}-arquivos"
  force_destroy = true # dados sintéticos: destruir o ambiente apaga tudo
}

resource "aws_s3_bucket_public_access_block" "arquivos" {
  bucket                  = aws_s3_bucket.arquivos.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_ownership_controls" "arquivos" {
  bucket = aws_s3_bucket.arquivos.id
  rule {
    object_ownership = "BucketOwnerEnforced"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "arquivos" {
  bucket = aws_s3_bucket.arquivos.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

data "aws_iam_policy_document" "arquivos" {
  statement {
    sid       = "CloudFrontLeSoPublico"
    actions   = ["s3:GetObject"]
    resources = ["${aws_s3_bucket.arquivos.arn}/publico/*"]
    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }
    condition {
      test     = "StringEquals"
      variable = "AWS:SourceArn"
      values   = [aws_cloudfront_distribution.entrada.arn]
    }
  }
  statement {
    sid       = "SoHTTPS"
    effect    = "Deny"
    actions   = ["s3:*"]
    resources = [aws_s3_bucket.arquivos.arn, "${aws_s3_bucket.arquivos.arn}/*"]
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

resource "aws_s3_bucket_policy" "arquivos" {
  bucket     = aws_s3_bucket.arquivos.id
  policy     = data.aws_iam_policy_document.arquivos.json
  depends_on = [aws_s3_bucket_public_access_block.arquivos]
}
