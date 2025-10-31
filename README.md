# Akeyless to Bitwarden Secrets Manager Migration Tool

Migrate all static secrets from Akeyless vault to Bitwarden Secrets Manager.

## Prerequisites

- Go 1.23+
- Akeyless access credentials (Access ID and Access Key)
- Bitwarden Secrets Manager access token
- Bitwarden organization ID and project ID

## Setup

1. Install dependencies:
```bash
go mod tidy
```

2. Create a `config.yaml` file:
```bash
cp config.yaml.example config.yaml
```

3. Edit `config.yaml` with your credentials:
```yaml
akeyless:
  url: https://api.akeyless.io
  access_id: your-access-id
  access_key: your-access-key

bitwarden:
  api_url: https://api.bitwarden.com
  identity_url: https://identity.bitwarden.com
  access_token: your-access-token
  org_id: your-org-id
  project_id: your-project-id
```

## Usage

### Local

```bash
CGO_LDFLAGS="-lm" go run main.go

# Or specify config path
CGO_LDFLAGS="-lm" go run main.go -config /path/to/config.yaml

# Or via environment variable
CONFIG_PATH=/path/to/config.yaml CGO_LDFLAGS="-lm" go run main.go
```

Or build first:
```bash
CGO_LDFLAGS="-lm" go build -o akeyless-to-bitwarden
./akeyless-to-bitwarden -config config.yaml
```

### Docker

```bash
docker build -t akeyless-to-bitwarden .
docker run -v $(pwd)/config.yaml:/app/config.yaml:ro akeyless-to-bitwarden
```

Or with docker-compose:
```bash
docker-compose up
```

The tool will:
1. Authenticate with Akeyless and Bitwarden
2. List all static secrets from Akeyless
3. Compare with existing Bitwarden secrets
4. Create new secrets or update existing ones
5. Group nested paths into JSON objects
6. Add sync timestamp to notes

## Running as a Cron Job

To keep secrets in sync, run the tool periodically:

```bash
# Every hour
0 * * * * cd /path/to/akeyless-to-bitwarden && docker-compose up
```

## Notes

- Only static secrets are migrated
- Secrets with non-string values are skipped
- Failed migrations are logged but don't stop the process
- Existing secrets in Bitwarden may cause conflicts
