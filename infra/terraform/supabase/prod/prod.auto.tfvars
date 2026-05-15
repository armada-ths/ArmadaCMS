project_ref    = "rsdjnixgxqauonaofrwr"
project_name   = "ArmadaCMS"
project_region = "eu-north-1"

database_name = "postgres"
database_user = "postgres"

api_db_schema            = ""
api_db_extra_search_path = "public,extensions"
api_max_rows             = 1000

# Pooler host is region-specific and cannot be derived from project_ref alone.
# Consumed by gcp/prod via tfe_outputs so it does not need to be set there.
pooler_host = "aws-1-eu-north-1.pooler.supabase.com"

# Staging branch DB connection details.
# gcp/staging doesn't use a pooler, since it supports direct DB connections via ipv6.
# Consumed by gcp/staging via tfe_outputs.
staging_db_host = "db.dqeikqjiztvmifmnbzbf.supabase.co"
