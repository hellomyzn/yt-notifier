# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project does

A scheduled job that polls YouTube RSS feeds (and optionally the YouTube Data API) for 100+ channels every few hours, then dispatches new video notifications as Discord embeds via webhooks. State is tracked in lightweight CSV files stored in a separate "footprints" repository.

## Commands

All Go commands run from the `src/` directory:

```bash
# Run the job locally
cd src && go run ./cmd/job

# Run tests
cd src && go test ./...

# Run a specific test
cd src && go test ./internal/repository/...

# Build
cd src && go build ./cmd/job
```

Docker-based workflow (uses external CSV via volume mount):

```bash
make up      # Start container
make exec    # Shell into workspace container
make logs    # View logs
make run     # Full cycle: up → run job → down
make down    # Stop containers
make destroy # Destroy containers and volumes
```

## Architecture

Clean layered architecture with no external dependencies (stdlib only):

```
cmd/job/main.go              → wires up dependencies, calls RunOnce()
internal/controller/         → orchestrates the job loop
internal/service/            → business logic (video discovery, notifications)
internal/repository/         → data access (CSV files, RSS XML, YouTube API)
internal/notifier/           → Discord webhook HTTP client
internal/model/dto.go        → ChannelDTO, VideoDTO
config/                      → YAML + .env file loaders
```

**Data flow**: `job_controller.RunOnce()` reads channels from `csv/channels.csv`, fetches RSS (or YouTube Data API for channels with `fetch_limit >= 15`), deduplicates against `csv/notified.csv`, dispatches Discord embeds, then appends newly notified video IDs back to `notified.csv`.

**Feed strategy**: RSS is the default. The YouTube Data API is used when a channel's `fetch_limit >= 15` and an API key is configured, because RSS caps at 15 items. If the API fails, it falls back to RSS. If RSS is saturated (returns exactly 15 items), the code escalates to the API.

**Notification retry**: Up to 5 retries per webhook post, exponential backoff (base 2s, max 30s), respects Discord `Retry-After` headers on 429s.

## Configuration

**`src/config/app.yaml`** — main config: category-to-webhook mappings, rate limits (`fetch_sleep_ms`, `post_sleep_ms`), and video filters (`include_premieres`, `include_live`, `include_shorts`).

**`src/config/webhooks.env`** — Discord webhook URLs, one per line as `KEY=VALUE`. Not in version control; see `webhooks.env.example`.

**`src/config/youtube.env`** — YouTube Data API key. Optional; without it, all channels use RSS only.

**CSV schemas**:
- `csv/channels.csv`: `channel_id,category,name,enabled,fetch_limit`
- `csv/notified.csv`: `video_id,channel_id,published_at,notified_at`

## Deployment

The job runs on GitHub Actions (`.github/workflows/notify.yml`) twice daily (8:00 AM and 8:00 PM JST). The workflow:
1. Checks out this repo and a separate "footprints" repo (holds the CSVs)
2. Writes secrets to `src/config/webhooks.env` and `src/config/youtube.env`
3. Syncs CSVs from footprints into `src/src/csv/`
4. Runs `cd src && go run ./cmd/job`
5. Commits and pushes updated CSVs back to footprints

Required GitHub secrets: `FOOTPRINTS_TOKEN`, `ENV_WEBHOOKS`, `ENV_YOUTUBE`, `FOOTPRINTS_AUTHOR_NAME`, `FOOTPRINTS_AUTHOR_EMAIL`.
