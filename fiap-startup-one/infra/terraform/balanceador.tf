# Balanceador de aplicação: só o CloudFront entra (grupo de segurança + cabeçalho secreto).

resource "aws_lb" "entrada" {
  name                       = var.nome
  load_balancer_type         = "application"
  internal                   = false
  subnets                    = aws_subnet.publica[*].id
  security_groups            = [aws_security_group.balanceador.id]
  idle_timeout               = 60
  drop_invalid_header_fields = true
}

resource "aws_lb_target_group" "api" {
  name                 = "${var.nome}-api"
  port                 = 8082
  protocol             = "HTTP"
  target_type          = "ip"
  vpc_id               = aws_vpc.principal.id
  deregistration_delay = 15
  health_check {
    path                = "/api/saude" # isenta da sala de espera e dos níveis
    matcher             = "200"
    interval            = 10
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }
}

resource "aws_lb_target_group" "web" {
  name                 = "${var.nome}-web"
  port                 = 8080
  protocol             = "HTTP"
  target_type          = "ip"
  vpc_id               = aws_vpc.principal.id
  deregistration_delay = 15
  health_check {
    path                = "/entrar"
    matcher             = "200"
    interval            = 10
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }
}

resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.entrada.arn
  port              = 80
  protocol          = "HTTP"
  default_action {
    type = "fixed-response"
    fixed_response {
      content_type = "text/plain"
      message_body = "acesso direto recusado"
      status_code  = "403"
    }
  }
}

resource "aws_lb_listener_rule" "api" {
  listener_arn = aws_lb_listener.http.arn
  priority     = 10
  condition {
    http_header {
      http_header_name = "X-Origem-Cloudfront"
      values           = [random_password.origem.result]
    }
  }
  condition {
    path_pattern {
      values = ["/api/*"]
    }
  }
  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.api.arn
  }
}

resource "aws_lb_listener_rule" "web" {
  listener_arn = aws_lb_listener.http.arn
  priority     = 20
  condition {
    http_header {
      http_header_name = "X-Origem-Cloudfront"
      values           = [random_password.origem.result]
    }
  }
  action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.web.arn
  }
}
