# Dokploy & Contabo Deployment Guide

## Production Architecture

In production, ZSMS runs on a **Contabo VPS** managed by **Dokploy** behind **Cloudflare**:

```text
Cloudflare (DNS, SSL, WAF, Turnstile)
           │
           ▼
Dokploy Traefik Reverse Proxy
     ├── sms.ztechai.us      -> web:3000 (Nuxt 3)
     └── sms-api.ztechai.us  -> api:8080 (Go Fiber)
```

## Deployment Steps in Dokploy

1. **Create New Project**:
   * Project Name: `ZSMS`
2. **Add Compose Service**:
   * Source: Git Repository (`github.com/Zubairyounus99/httpsmsModified2`)
   * Branch: `main`
   * Compose Path: `docker-compose.yml`
3. **Configure Environment Variables**:
   * Copy the required variables from `.env.example` into the Dokploy Environment tab.
   * Set `APP_ENV=production`.
   * Set strong passwords for `DB_PASSWORD`, `REDIS_PASSWORD`, and `JWT_SECRET`.
4. **Deploy**:
   * Click **Deploy**. Dokploy executes `docker compose pull && docker compose build && docker compose up -d`.
   * Traefik will automatically issue Let's Encrypt certificates for the configured domains.
