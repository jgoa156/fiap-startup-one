# Aurora PostgreSQL Serverless v2, como na arquitetura alvo. A senha do usuário mestre é criada e
# guardada pelo próprio RDS no Secrets Manager: não passa pelo Terraform nem fica no estado.

data "aws_rds_engine_version" "postgres" {
  engine  = "aurora-postgresql"
  version = "17"
  latest  = true
}

resource "aws_db_subnet_group" "banco" {
  name       = "${var.nome}-banco"
  subnet_ids = aws_subnet.privada[*].id
}

resource "aws_rds_cluster" "banco" {
  cluster_identifier          = "${var.nome}-banco"
  engine                      = "aurora-postgresql"
  engine_mode                 = "provisioned"
  engine_version              = data.aws_rds_engine_version.postgres.version_actual
  database_name               = "lisura"
  master_username             = "lisura"
  manage_master_user_password = true
  db_subnet_group_name        = aws_db_subnet_group.banco.name
  vpc_security_group_ids      = [aws_security_group.banco.id]
  storage_encrypted           = true

  serverlessv2_scaling_configuration {
    min_capacity             = var.banco_acu_min
    max_capacity             = var.banco_acu_max
    seconds_until_auto_pause = var.banco_acu_min == 0 ? 300 : null
  }

  # Dados sintéticos, ambiente recriável: sem cópia final e sem proteção contra exclusão.
  skip_final_snapshot     = true
  deletion_protection     = false
  backup_retention_period = 1
}

resource "aws_rds_cluster_instance" "escrita" {
  identifier          = "${var.nome}-banco-1"
  cluster_identifier  = aws_rds_cluster.banco.id
  instance_class      = "db.serverless"
  engine              = aws_rds_cluster.banco.engine
  engine_version      = aws_rds_cluster.banco.engine_version
  publicly_accessible = false
  # Performance Insights mostra onde o banco gasta o tempo: é a ferramenta para achar o gargalo.
  performance_insights_enabled          = true
  performance_insights_retention_period = 7
}
