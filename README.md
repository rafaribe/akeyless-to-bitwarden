# Akeyless to Bitwarden Secrets Manager Migration Tool

Migrate and sync all static secrets from Akeyless vault to Bitwarden Secrets Manager.

## Features

- **Recursive Secret Discovery** - Automatically traverses the entire Akeyless folder structure to find all nested secrets
- **Flat Secret Structure** - Creates individual Bitwarden secrets with full path names (e.g., `/hass/ssh` → `hass/ssh`)
- **Smart Sync** - Creates new secrets or updates existing ones based on current state
- **Sync Tracking** - Adds timestamp notes to track when secrets were last synced
- **Rate Limit Handling** - Automatically retries with backoff when hitting API rate limits
- **Error Resilience** - Failed migrations are logged but don't stop the entire sync process
- **Clean Architecture** - Modular codebase following SOLID principles for easy maintenance and testing

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

## Running as a Cron Job

To keep secrets in sync, run the tool periodically:

```bash
# Every hour
0 * * * * cd /path/to/akeyless-to-bitwarden && docker-compose up
```

## How It Works

1. Authenticates with both Akeyless and Bitwarden
2. Recursively discovers all static secrets in Akeyless (including nested paths)
3. Lists existing secrets in the target Bitwarden project
4. For each Akeyless secret:
   - Creates a new Bitwarden secret if it doesn't exist
   - Updates the existing Bitwarden secret if it already exists
5. Adds sync timestamp to secret notes: `Synced from Akeyless @ <timestamp>`

## Project Structure

```
.
├── main.go          # Application entry point
├── config.go        # Configuration loading
├── akeyless.go      # Akeyless client wrapper
├── bitwarden.go     # Bitwarden client wrapper
├── syncer.go        # Sync orchestration logic
├── config.yaml      # Configuration file (not in git)
└── Dockerfile       # Container image definition
```

## Notes

- Only static secrets are migrated
- Secrets with non-string values are skipped
- Nested secrets are created as flat keys (e.g., `/hass/ssh` becomes `hass/ssh`)
- Sync timestamps overwrite previous notes on each run
- Rate limiting is handled automatically with 60-second retry delays
