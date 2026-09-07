# Docker Architecture & Operations

## Overview

The ZSMS repository provides two Docker Compose configurations:

1. **`docker-compose.dev.yml`**:
   * Tailored for local engineering and testing.
   * Maps ports directly to `localhost` (`3000`, `8080`, `5432`, `6379`, `9000`, `9001`).
   * Includes automatic MinIO bucket provisioning (`minio-init`).
   * Uses development database credentials.

2. **`docker-compose.yml`**:
   * Production Dokploy blueprint for Contabo VPS.
   * Traefik routing labels for `sms.ztechai.us` and `sms-api.ztechai.us`.
   * Ports for databases and caches are isolated within the internal bridge network (`zsms_prod_net`) and not exposed publicly.
   * Non-root container security execution.

## Common Docker Commands

### Start development stack:
```bash
docker compose -f docker-compose.dev.yml up -d
```

### Stop development stack:
```bash
docker compose -f docker-compose.dev.yml down
```

### View container logs:
```bash
docker compose -f docker-compose.dev.yml logs -f api
docker compose -f docker-compose.dev.yml logs -f worker
```
