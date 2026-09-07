# ZSMS System Architecture

**Product**: ZSMS (Carrier-Independent Cellular SMS/MMS Gateway)  
**Client**: ZTechAI  
**Version**: 1.0.0 (Monorepo Foundation)

---

## 1. System Overview

ZSMS turns physical Android mobile devices into programmable, carrier-independent cellular SMS/MMS gateways. It consists of:

1. **Backend Core (`backend/`)**:
   * **API Service**: High-performance Go Fiber HTTP server providing versioned REST endpoints (`/v1`), health checks, server-side RBAC, and request routing.
   * **Worker Service**: Autonomous background service orchestrating queues, device selection, FCM pushes, carrier delivery reports, and webhook delivery.
2. **Frontend Dashboard (`web/`)**:
   * Modern, responsive SaaS dashboard built with Nuxt 3, Vue 3, Vuetify 3, and Pinia.
   * Provides real-time gateway metrics, message logs, two-way conversations, contact management, and broadcast campaign tools.
3. **Android Client (`android/`)**:
   * Native Kotlin application (API 26–34) executing persistent gateway operations.
   * Transmits SMS/MMS via hardware cellular SIM cards and reports delivery receipts back to the central server.
4. **Data & State Layer**:
   * **PostgreSQL 16**: Authoritative persistent database.
   * **Redis 7**: Distributed queues, mutual exclusion locks, presence indicators, and rate limiters.
   * **MinIO / Cloudflare R2**: S3-compliant object storage for multimedia attachments.
