# ── Security group ──────────────────────────────────────────────────────────────

resource "aws_security_group" "rds" {
  name        = "armadacms-rds-sg"
  description = "PostgreSQL access for ArmadaCMS from Cloud Run NAT"
  vpc_id      = local.vpc_id

  ingress {
    description = "PostgreSQL from Cloud Run NAT"
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = [local.nat_cidr]
  }

  egress {
    description = "Allow all outbound"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = "armadacms-rds-sg"
  })
}

# ── RDS PostgreSQL instance ─────────────────────────────────────────────────────

resource "aws_db_instance" "main" {
  identifier = local.rds_identifier

  engine         = "postgres"
  engine_version = "17.5"
  instance_class = "db.t4g.micro"

  db_name  = "armadacms"
  username = "postgres"
  password = var.db_password

  allocated_storage = 20
  storage_type      = "gp2"

  publicly_accessible    = true
  multi_az               = false
  availability_zone      = "eu-north-1a"
  db_subnet_group_name   = "default"
  vpc_security_group_ids = [aws_security_group.rds.id]

  backup_retention_period = 0
  deletion_protection     = true

  tags = merge(local.common_tags, {
    Name = "armadacms-prod-db"
  })

  lifecycle {
    # password is write-only in the AWS API; importing the instance will leave it
    # blank in state. Ignore changes so Terraform never resets the password unless
    # you explicitly want to rotate it.
    ignore_changes = [password, engine_version]
  }
}
