# ZSMS Verification & Physical Cellular Test Plan

This document serves as the authoritative verification checklist for validating **ZSMS** on the Dokploy infrastructure (`https://sms.ztechai.us` & `https://sms-api.ztechai.us`) using a real physical Android smartphone with an active SIM card.

> [!WARNING]
> **Strict Verification Standard**:
> Do NOT mark any test item as `[x]` (PASS) based solely on code review or compilation. A test item may only be marked complete once the exact physical operation has been executed and observed.

---

## 1. Infrastructure Checklist (Dokploy VPS)

| Verification Item | Command / Method | Target Condition | Status |
| :--- | :--- | :--- | :--- |
| Dokploy deployment successful | Dokploy UI / Webhooks | Stack status: Running (All containers green) | `[ ] NOT TESTED` |
| PostgreSQL running & healthy | `docker exec zsms-staging-postgres pg_isready` | `accepting connections` | `[ ] NOT TESTED` |
| Redis running & healthy | `docker exec zsms-staging-redis redis-cli ping` | `PONG` | `[ ] NOT TESTED` |
| MinIO running if required | `curl -f http://localhost:9000/minio/health/live` | HTTP 200 | `[ ] NOT TESTED` |
| Schema migrations successful | Database query | Tables `users`, `phones`, `messages`, etc. present | `[ ] NOT TESTED` |
| Traefik SSL/TLS termination | `curl -I https://sms-api.ztechai.us/livez` | HTTP 200 with valid Let's Encrypt TLS cert | `[ ] NOT TESTED` |
| Backend API readiness | `curl https://sms-api.ztechai.us/health/ready` | `{"status":"ok","checks":{"database":"ok","redis":"ok"}}` | `[ ] NOT TESTED` |
| Web frontend readiness | `curl -I https://sms.ztechai.us/` | HTTP 200 Nuxt Nitro HTML response | `[ ] NOT TESTED` |

---

## 2. Authentication Checklist (Web Dashboard)

| Verification Item | Target Action | Verification Criteria | Status |
| :--- | :--- | :--- | :--- |
| User Registration | Sign up with new email & password at `/auth/login` | Account created in DB; JWT token saved in Pinia store | `[ ] NOT TESTED` |
| User Login | Sign in with registered credentials | Successfully redirected to `/dashboard` | `[ ] NOT TESTED` |
| User Logout | Click profile avatar -> Sign Out | Token cleared from localStorage; redirected to `/auth/login` | `[ ] NOT TESTED` |
| Session Validation | Refresh protected page (e.g. `/messages`) | Session restored without re-prompting login | `[ ] NOT TESTED` |
| Invalid Credentials | Submit incorrect password | Red alert displayed: `Invalid email or password` | `[ ] NOT TESTED` |

---

## 3. Android Application & Pairing Checklist

| Verification Item | Target Action | Verification Criteria | Status |
| :--- | :--- | :--- | :--- |
| Cloud APK Generation | GitHub Actions workflow execution | `build-apk` job passes; artifact generated | `[ ] NOT TESTED` |
| APK Download & Transfer | Download `ZSMS-Android-Debug` from Actions | `app-debug.apk` unpacks cleanly (~15-20MB) | `[ ] NOT TESTED` |
| Manual Installation | Install APK on physical Android device | App installs cleanly without package parse errors | `[ ] NOT TESTED` |
| Application Launch | Launch ZSMS from home screen / app drawer | Main dashboard opens; displays device hardware info | `[ ] NOT TESTED` |
| Runtime Permissions | Tap "Grant SMS Permissions" | System dialogs appear for SMS, Phone State, Notifications | `[ ] NOT TESTED` |
| Server API Configuration | Tap "Server Settings" -> "Test Server Connection" | Connects to `https://sms-api.ztechai.us`; green HTTP 200 | `[ ] NOT TESTED` |
| Device Pairing Session | Web: `/phones` -> "Pair New Phone" | Ephemeral 6-digit PIN generated with 10-minute timer | `[ ] NOT TESTED` |
| Device PIN Claim | Android: "Pair Gateway Device" -> Enter PIN | Phone claims token; stores in Keystore AES-256 | `[ ] NOT TESTED` |
| Foreground Service Start | Automatic after pairing | Ongoing notification: `Gateway ONLINE (Active cellular bridge)` | `[ ] NOT TESTED` |
| Secure WebSocket Connection | Phone connects to `/api/v1/gateway/ws` | Authenticates via `Authorization: Bearer <token>` header | `[ ] NOT TESTED` |
| Gateway Online Status | Web: `/phones` & `/dashboard` | Phone card displays green `ONLINE` status & carrier | `[ ] NOT TESTED` |

---

## 4. Real Cellular SMS Telephony Checklist

| Verification Item | Target Action | Verification Criteria | Status |
| :--- | :--- | :--- | :--- |
| **Outbound Real SMS** | Web: `/messages` -> Dispatch SMS to external phone | External handset physically receives SMS via cellular radio | `[ ] NOT TESTED` |
| **SMS Sent Status** | Observe Web Dashboard after outbound dispatch | Status transitions from `processing` to `sent` | `[ ] NOT TESTED` |
| **SMS Delivery Receipt** | Carrier returns SMSC delivery report | Status transitions from `sent` to `delivered` (if carrier supports) | `[ ] NOT TESTED` |
| **Inbound Real SMS** | Reply from external handset to phone's SIM number | Physical SIM receives SMS; `SmsReceiver` intercepts PDU | `[ ] NOT TESTED` |
| **Two-Way Chat Stream** | Open Web: `/conversations` | Inbound reply appears in chat bubble with correct timestamp | `[ ] NOT TESTED` |
| **Standard GSM-7 Message** | Send SMS under 160 characters | Transmitted as 1 segment (`parts_count = 1`) | `[ ] NOT TESTED` |
| **Multipart GSM-7 Message** | Send SMS over 160 characters (e.g. 200 chars) | Correctly split and reassembled (`parts_count = 2`) | `[ ] NOT TESTED` |
| **Unicode / Emoji Message** | Send SMS containing emojis or Arabic/Chinese chars | Character encoding intact; UTF-8 displayed correctly on handset | `[ ] NOT TESTED` |
| **Multipart Unicode SMS** | Send Unicode SMS exceeding 70 characters | Split into 67-char segments and delivered without garbled text | `[ ] NOT TESTED` |
| **Dual-SIM Slot Selection** | If phone has Dual SIM, dispatch via SIM 2 | Radio transmits from secondary SIM's phone number | `[ ] NOT TESTED` |
| **Invalid Recipient Number** | Dispatch to malformed or non-existent number | Radio reports failure; status transitions to `failed` | `[ ] NOT TESTED` |

---

## 5. Resilience & Failure Recovery Checklist

| Verification Item | Target Action | Verification Criteria | Status |
| :--- | :--- | :--- | :--- |
| **Wi-Fi / Internet Drop** | Turn off Wi-Fi on Android phone for 60 seconds | Reconnects automatically with exponential backoff | `[ ] NOT TESTED` |
| **HTTP Polling Fallback** | Queue SMS while WebSocket is disconnected | HTTP polling loop (`/api/v1/android/messages/poll`) picks up message | `[ ] NOT TESTED` |
| **Idempotency Protection** | Re-submit same `Idempotency-Key` / request | Database rejects duplicate; radio transmits only ONCE | `[ ] NOT TESTED` |
| **Backend API Restart** | Restart `zsms-staging-api` in Dokploy | Android phone waits and reconnects once API is back | `[ ] NOT TESTED` |
| **Android App Force-Stop** | Force-stop ZSMS in phone Application Settings | On relaunch, secure token is preserved; reconnects immediately | `[ ] NOT TESTED` |
| **Android Device Reboot** | Completely reboot the physical Android phone | `BootReceiver` restarts `ZsmsGatewayService`; status returns to `ONLINE` | `[ ] NOT TESTED` |

---

## 6. Physical Verification Execution Sign-Off

* **Tester Name**: __________________________
* **Physical Device Model**: __________________________
* **Android OS Version**: __________________________
* **Cellular Carrier**: __________________________
* **Test Date**: __________________________
* **Overall Result**: `[ ] PASS` | `[ ] FAIL`
