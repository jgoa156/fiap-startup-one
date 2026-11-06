# CloudFront é a porta única do candidato, como na arquitetura alvo (lá, atrás do Cloudflare):
#   /publico/*        → S3, com cache (o pico de leitura não chega ao backend)
#   /_next/static/*   → front, com cache (arquivos com hash no nome, nunca mudam)
#   /api/*            → balanceador → API Go, sem cache
#   resto             → balanceador → front Next, sem cache

data "aws_cloudfront_cache_policy" "sem_cache" {
  name = "Managed-CachingDisabled"
}

data "aws_cloudfront_cache_policy" "com_cache" {
  name = "Managed-CachingOptimized"
}

# Repassa cookies, busca e cabeçalhos, inclusive CloudFront-Forwarded-Proto (é por ele que a API
# sabe que o navegador veio por HTTPS e marca os cookies como Secure).
data "aws_cloudfront_origin_request_policy" "tudo" {
  name = "Managed-AllViewerAndCloudFrontHeaders-2022-06"
}

resource "aws_cloudfront_origin_access_control" "s3" {
  name                              = "${var.nome}-s3"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_distribution" "entrada" {
  enabled         = true
  comment         = var.nome
  http_version    = "http2and3"
  is_ipv6_enabled = true
  price_class     = "PriceClass_All" # inclui os pontos de presença da América do Sul

  origin {
    origin_id   = "balanceador"
    domain_name = aws_lb.entrada.dns_name
    custom_origin_config {
      http_port                = 80
      https_port               = 443
      origin_protocol_policy   = "http-only" # sem domínio próprio não há certificado no balanceador (ver README)
      origin_ssl_protocols     = ["TLSv1.2"]
      origin_read_timeout      = 30
      origin_keepalive_timeout = 60
    }
    custom_header {
      name  = "X-Origem-Cloudfront"
      value = random_password.origem.result
    }
  }

  origin {
    origin_id                = "arquivos"
    domain_name              = aws_s3_bucket.arquivos.bucket_regional_domain_name
    origin_access_control_id = aws_cloudfront_origin_access_control.s3.id
  }

  ordered_cache_behavior {
    path_pattern           = "/publico/*"
    target_origin_id       = "arquivos"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]
    cache_policy_id        = data.aws_cloudfront_cache_policy.com_cache.id # respeita o max-age=60 do objeto
    compress               = true
  }

  ordered_cache_behavior {
    path_pattern             = "/api/*"
    target_origin_id         = "balanceador"
    viewer_protocol_policy   = "redirect-to-https"
    allowed_methods          = ["GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"]
    cached_methods           = ["GET", "HEAD"]
    cache_policy_id          = data.aws_cloudfront_cache_policy.sem_cache.id
    origin_request_policy_id = data.aws_cloudfront_origin_request_policy.tudo.id
    compress                 = true
  }

  ordered_cache_behavior {
    path_pattern           = "/_next/static/*"
    target_origin_id       = "balanceador"
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]
    cache_policy_id        = data.aws_cloudfront_cache_policy.com_cache.id
    compress               = true
  }

  default_cache_behavior {
    target_origin_id         = "balanceador"
    viewer_protocol_policy   = "redirect-to-https"
    allowed_methods          = ["GET", "HEAD", "OPTIONS", "PUT", "POST", "PATCH", "DELETE"]
    cached_methods           = ["GET", "HEAD"]
    cache_policy_id          = data.aws_cloudfront_cache_policy.sem_cache.id
    origin_request_policy_id = data.aws_cloudfront_origin_request_policy.tudo.id
    compress                 = true
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
  }
}
