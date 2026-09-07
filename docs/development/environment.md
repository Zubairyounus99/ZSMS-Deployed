# Environment Variables Specification & Reference

This document provides the authoritative specification for all environment variables used by **ZSMS** across the Go API backend, background worker, and Nuxt 3 web application.

---

## 1. Environment Variable Audit & Matrix

| Variable | Required Local | Required Production | Purpose | Default / Local Example |
| :--- | :--- | :--- | :--- | :--- |
| `APP_NAME` | **Yes** | **Yes** | Branding and application name | `ZSMS` |
| `APP_ENV` | **Yes** | **Yes** | Runtime environment (`development`, `staging`, `production`) | `development` |
| `APP_DEBUG` | Optional | **Yes** | Enables verbose logging and developer diagnostics | `true` |
| `LOG_LEVEL` | Optional | **Yes** | Minimum log severity level (`debug`, `info`, `warn`, `error`) | `info` |
| `TZ` | Optional | Optional | Server timezone identifier | `UTC` |
| `APP_URL` | **Yes** | **Yes** | Public frontend URL of the Nuxt 3 application | `http://localhost:3000` |
| `API_URL` | **Yes** | **Yes** | Base URL for REST API endpoints | `http://localhost:8080` |
| `WEB_URL` | Optional | Optional | Alias for `APP_URL` | `http://localhost:3000` |
| `DOCS_URL` | Optional | Optional | API documentation / Swagger endpoint | `http://localhost:8080/docs` |
| `PORT` | **Yes** | **Yes** | TCP port the Go HTTP backend binds to | `8080` |
| `HOST` | **Yes** | **Yes** | Network interface address (`0.0.0.0` required for physical phone LAN) | `0.0.0.0` |
| `ALLOW_HTTP_LOCAL` | **Yes** | **No** | Permits cleartext HTTP communication exclusively on local LANs | `true` |
| `CORS_ALLOWED_ORIGINS`| **Yes** | **Yes** | Comma-separated list of allowed web origins | `http://localhost:3000,http://127.0.0.1:3000` |
| `POSTGRES_DB` | **Yes** | **Yes** | Database name | `zsms_db` |
| `POSTGRES_USER` | **Yes** | **Yes** | Database user | `zsms_user` |
| `POSTGRES_PASSWORD` | **Yes** | **Yes** | Database user password | `zsms_dev_password` |
| `POSTGRES_PORT` | **Yes** | **Yes** | Database port | `5432` |
| `POSTGRES_HOST` | **Yes** | **Yes** | Database host address | `localhost` |
| `DATABASE_URL` | **Yes** | **Yes** | Canonical PostgreSQL connection URL | `postgres://zsms_user:zsms_dev_password@localhost:5432/zsms_db?sslmode=disable` |
| `DB_MAX_OPEN_CONNS` | Optional | **Yes** | Maximum open connections in database pool | `25` |
| `DB_MAX_IDLE_CONNS` | Optional | **Yes** | Maximum idle connections in database pool | `10` |
| `REDIS_HOST` | **Yes** | **Yes** | Redis host address | `localhost` |
| `REDIS_PORT` | **Yes** | **Yes** | Redis port | `6379` |
| `REDIS_PASSWORD` | Optional | **Yes** | Redis authentication password | `""` (none for local dev) |
| `REDIS_URL` | **Yes** | **Yes** | Canonical Redis connection URL | `redis://localhost:6379/0` |
| `JWT_SECRET` | **Yes** | **Yes** | Secret key for signing user authentication JWT tokens | Minimum 64-byte random string |
| `JWT_EXPIRATION` | **Yes** | **Yes** | Lifetime format string for JWT access tokens | `72h` |
| `PAIRING_CODE_EXPIRATION` | Optional | **Yes** | Validity window for 6-digit device pairing codes (in minutes) | `10` |
| `DEVICE_TOKEN_PREFIX` | Optional | **Yes** | Prefix for issued Android hardware device tokens | `zsms_dev_` |
| `STORAGE_DRIVER` | **Yes** | **Yes** | Object storage driver (`minio`, `local`, `r2`) | `minio` |
| `STORAGE_PATH` | Optional | Optional | Local file path for fallback local driver | `./storage` |
| `S3_ENDPOINT` | **Yes** | **No** | MinIO local S3 API endpoint | `localhost:9000` |
| `S3_ACCESS_KEY` | **Yes** | **No** | MinIO root access key | `zsms_minio_admin` |
| `S3_SECRET_KEY` | **Yes** | **No** | MinIO root secret key | `zsms_minio_secret_key` |
| `S3_BUCKET` | **Yes** | **Yes** | S3 bucket name for attachments | `zsms-media` |
| `S3_USE_SSL` | Optional | **Yes** | Enable TLS for S3 connection | `false` |
| `PHONE_OFFLINE_THRESHOLD_SECONDS` | Optional | **Yes** | Heartbeat timeout before device is considered offline | `120` |
| `DEFAULT_MESSAGE_EXPIRY_HOURS` | Optional | **Yes** | Expiration period for undelivered outbound messages | `24` |
| `FIREBASE_PROJECT_ID` | **No** (Optional)| **Yes** (Stage 6)| FCM push notification project ID | _Unset for Stage 3/3.5_ |
| `FIREBASE_CLIENT_EMAIL`| **No** (Optional)| **Yes** (Stage 6)| FCM service account email | _Unset for Stage 3/3.5_ |
| `FIREBASE_PRIVATE_KEY` | **No** (Optional)| **Yes** (Stage 6)| FCM service account private key | _Unset for Stage 3/3.5_ |
| `R2_ENDPOINT` | **No** (Optional)| **Yes** (Stage 9)| Cloudflare R2 S3 API endpoint for production MMS | _Unset for Stage 3/3.5_ |
| `R2_ACCESS_KEY` | **No** (Optional)| **Yes** (Stage 9)| Cloudflare R2 access key ID | _Unset for Stage 3/3.5_ |
| `R2_SECRET_KEY` | **No** (Optional)| **Yes** (Stage 9)| Cloudflare R2 secret access key | _Unset for Stage 3/3.5_ |
| `R2_BUCKET` | **No** (Optional)| **Yes** (Stage 9)| Cloudflare R2 bucket name | _Unset for Stage 3/3.5_ |
| `SMTP_HOST` | **No** (Optional)| **Yes** (Stage 10)| Email alerts relay server | _Unset for Stage 3/3.5_ |
| `SMTP_PORT` | **No** (Optional)| **Yes** (Stage 10)| Email alerts port | `587` |
| `SMTP_USERNAME` | **No** (Optional)| **Yes** (Stage 10)| Email alerts authentication username | _Unset for Stage 3/3.5_ |
| `SMTP_PASSWORD` | **No** (Optional)| **Yes** (Stage 10)| Email alerts authentication password | _Unset for Stage 3/3.5_ |
| `TURNSTILE_SITE_KEY` | **No** (Optional)| **Yes** (Stage 11)| Cloudflare Turnstile public site key | _Unset for Stage 3/3.5_ |
| `TURNSTILE_SECRET_KEY`| **No** (Optional)| **Yes** (Stage 11)| Cloudflare Turnstile secret key | _Unset for Stage 3/3.5_ |
| `ENCRYPTION_KEY` | **No** (Optional)| **Yes** (Stage 5+)| AES-256 application encryption key | _Unset for Stage 3/3.5_ |

---

## 2. Security Principles
1. **Never Commit Secrets**: The `.env` file is excluded in `.gitignore` and must never be committed to Git.
2. **Device Tokens**: Android device API keys (`zsms_dev_...`) are hashed using SHA-256 before storage in PostgreSQL.
3. **Local Simplicity**: Future credentials (Firebase, SMTP, Cloudflare, R2, Turnstile) are completely optional and not required for local physical SMS testing.
