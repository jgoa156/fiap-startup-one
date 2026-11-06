# Registro das imagens (ECR) e quem as constrói (CodeBuild). A construção roda na AWS porque esta
# máquina não tem Docker para Linux; e assim qualquer pessoa reproduz a imagem só com a AWS CLI.

locals {
  repos = {
    go  = "${var.nome}/go"
    web = "${var.nome}/web"
    k6  = "${var.nome}/k6"
  }
  registro = "${data.aws_caller_identity.conta.account_id}.dkr.ecr.${data.aws_region.atual.region}.amazonaws.com"
}

resource "aws_ecr_repository" "imagem" {
  for_each             = local.repos
  name                 = each.value
  force_delete         = true
  image_tag_mutability = "MUTABLE"
  image_scanning_configuration {
    scan_on_push = true
  }
}

resource "aws_ecr_lifecycle_policy" "imagem" {
  for_each   = aws_ecr_repository.imagem
  repository = each.value.name
  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "guarda as 5 mais recentes"
      selection    = { tagStatus = "any", countType = "imageCountMoreThan", countNumber = 5 }
      action       = { type = "expire" }
    }]
  })
}

data "aws_iam_policy_document" "codebuild_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["codebuild.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "codebuild" {
  name               = "${var.nome}-codebuild"
  assume_role_policy = data.aws_iam_policy_document.codebuild_assume.json
}

resource "aws_cloudwatch_log_group" "codebuild" {
  name              = "/${var.nome}/codebuild"
  retention_in_days = 7
}

data "aws_iam_policy_document" "codebuild" {
  statement {
    actions   = ["logs:CreateLogStream", "logs:PutLogEvents"]
    resources = ["${aws_cloudwatch_log_group.codebuild.arn}:*"]
  }
  statement {
    actions   = ["s3:GetObject", "s3:GetObjectVersion"]
    resources = ["${aws_s3_bucket.arquivos.arn}/fonte/*"]
  }
  statement {
    actions   = ["ecr:GetAuthorizationToken"]
    resources = ["*"]
  }
  statement {
    actions = [
      "ecr:BatchCheckLayerAvailability", "ecr:BatchGetImage", "ecr:GetDownloadUrlForLayer",
      "ecr:InitiateLayerUpload", "ecr:UploadLayerPart", "ecr:CompleteLayerUpload", "ecr:PutImage",
    ]
    resources = [for r in aws_ecr_repository.imagem : r.arn]
  }
}

resource "aws_iam_role_policy" "codebuild" {
  role   = aws_iam_role.codebuild.id
  policy = data.aws_iam_policy_document.codebuild.json
}

resource "aws_codebuild_project" "imagens" {
  name          = "${var.nome}-imagens"
  service_role  = aws_iam_role.codebuild.arn
  build_timeout = 30

  source {
    type      = "S3"
    location  = "${aws_s3_bucket.arquivos.id}/fonte/fonte.zip"
    buildspec = "infra/buildspec.yml"
  }

  artifacts {
    type = "NO_ARTIFACTS"
  }

  environment {
    type            = "LINUX_CONTAINER"
    compute_type    = "BUILD_GENERAL1_MEDIUM"
    image           = "aws/codebuild/amazonlinux-x86_64-standard:5.0"
    privileged_mode = true # docker build
    environment_variable {
      name  = "REGISTRO"
      value = local.registro
    }
    environment_variable {
      name  = "REPO_GO"
      value = aws_ecr_repository.imagem["go"].repository_url
    }
    environment_variable {
      name  = "REPO_WEB"
      value = aws_ecr_repository.imagem["web"].repository_url
    }
    environment_variable {
      name  = "REPO_K6"
      value = aws_ecr_repository.imagem["k6"].repository_url
    }
    environment_variable {
      name  = "TAG"
      value = var.tag_imagem
    }
  }

  logs_config {
    cloudwatch_logs {
      group_name = aws_cloudwatch_log_group.codebuild.name
    }
  }
}
