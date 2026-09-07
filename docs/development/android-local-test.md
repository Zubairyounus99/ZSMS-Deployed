# Physical Android Gateway Local Testing Guide

This guide details the exact procedure for configuring, installing, and validating the **ZSMS Android Gateway** on a **physical Android smartphone** connected to your local Windows development machine.

> [!IMPORTANT]
> **Stage 3.5 Status Classification**:
> * **Code Implementation**: **PASS** (Backend Go API, Nuxt 3 Web UI, Kotlin Android code complete)
> * **Frontend Build**: **PASS** (`vue-tsc` typecheck 0 errors, Nitro production bundle built)
> * **Backend Services**: **PASS** (Go REST & WebSocket routes, DB repos, JWT/Token auth implemented)
> * **Android Source Code**: **IMPLEMENTED** (`SmsManager`, receivers, foreground service, UI)
> * **Gradle Wrapper**: **CONFIGURED** (`gradlew`, `gradlew.bat`, and `gradle-wrapper.jar` present)
> * **APK Binary**: **BLOCKED on host environment** (requires JDK 17 / Android Studio to execute build)
> * **Physical Android Handset**: **NOT TESTED** (pending user APK installation on device)
> * **Real Cellular SMS Transmission**: **NOT VERIFIED** (pending physical SIM transmission test)
>
> *Do NOT convert source-code implementation into physical verification.* The system will only be declared fully verified once a real SMS is confirmed transmitted across the physical carrier cellular radio.

---

## 1. Network Topology & Architecture

Because a physical Android phone runs on its own isolated operating system, `http://localhost:8080` points to the phone itself rather than your development workstation. All communication occurs over your local Wi-Fi router LAN.

```text
┌────────────────────────────────────────────────────────┐
│               Local Wi-Fi Router (LAN)                 │
│                 192.168.1.0 / 24                       │
└───────────────▲──────────────────────▲─────────────────┘
                │                      │
                │ Wi-Fi                │ Wi-Fi / Ethernet
                ▼                      ▼
┌────────────────────────────┐ ┌─────────────────────────────────────────┐
│  Physical Android Device   │ │         Windows Development PC          │
│   (e.g., 192.168.1.145)    │ │          (e.g., 192.168.1.100)          │
│                            │ │                                         │
│ • ZSMS Gateway App         │ │ • Go Fiber API Server (:8080)           │
│ • SmsManager (Radio SIM)   │ │ • Nuxt 3 Web Dashboard (:3000)          │
│ • Persistent Foreground Svc│ │ • PostgreSQL Database (:5432)           │
│ • Real-time WebSocket      │ │ • Redis Message Broker (:6379)          │
└────────────────────────────┘ └─────────────────────────────────────────┘
```

---

## 2. Windows Development Host Preparation

### 2.1 Determine Your Windows LAN IP Address
Open PowerShell and run:
```powershell
ipconfig
```
Look for the **IPv4 Address** under your active Wi-Fi or Ethernet adapter (e.g. `192.168.1.100`).

### 2.2 Configure Windows Defender Firewall (Crucial)
By default, Windows Firewall blocks incoming connections from other devices on the LAN to port `8080`.

Open PowerShell as **Administrator** and run:
```powershell
netsh advfirewall firewall add rule name="ZSMS_API_Gateway" dir=in action=allow protocol=TCP localport=8080
```
To verify the rule:
```powershell
netsh advfirewall firewall show rule name="ZSMS_API_Gateway"
```

### 2.3 Start Backend and Web Services
Ensure the PostgreSQL, Redis, Go API, and Nuxt Dashboard are running:
```powershell
# In repo root:
docker compose up -d postgres redis

# Run backend:
cd backend
go run ./api/main.go

# Run web frontend (in a separate terminal):
cd web
npm run dev
```

Test that the API responds on the LAN interface from your browser or mobile browser:
```text
http://192.168.1.100:8080/livez
```
You should receive:
```json
{"status":"ok","timestamp":"...","version":"1.0.0"}
```

---

## 3. Android APK Build & Installation

### Option A: Open with Android Studio (Recommended)
1. Launch **Android Studio**.
2. Select **Open** and choose the `d:\Zubair\Antigravity\ZSMS\android` directory.
3. Allow Gradle to sync dependencies and generate wrapper binaries.
4. Connect your physical Android smartphone via USB and enable **USB Debugging** (in Developer Options).
5. Select your physical phone in the device dropdown and click **Run** (or `Shift + F10`).
6. Alternatively, build the APK from the menu: **Build > Build Bundle(s) / APK(s) > Build APK(s)**.

### Option B: Build via CLI with Gradle & JDK 17
If you have JDK 17 and Android SDK installed:
```powershell
cd android
./gradlew assembleDebug
# Output APK location:
# android/app/build/outputs/apk/debug/app-debug.apk
```
Install on device:
```powershell
adb install -r app/build/outputs/apk/debug/app-debug.apk
```

---

## 4. Device Setup & Pairing Procedure

1. **Launch App on Phone**:
   - Open the **ZSMS** application on your physical Android smartphone.

2. **Grant Telephony & SMS Permissions**:
   - When prompted, tap **"Grant SMS Permissions"**.
   - Accept the runtime permission dialogs:
     - `SEND_SMS`: Allows sending real SMS messages via SIM card.
     - `RECEIVE_SMS`: Enables receiving and ingesting inbound carrier SMS.
     - `READ_PHONE_STATE`: Allows detecting cellular carrier and SIM status.
     - `POST_NOTIFICATIONS` (Android 13+): Ensures persistent foreground notification.

3. **Configure LAN Server Endpoint**:
   - In the ZSMS app, tap **Server Settings**.
   - Set **Server URL** to your Windows PC LAN IP:
     ```text
     http://192.168.1.100:8080
     ```
   - Tap **Test Server Connection**. You should see:
     `CONNECTED! Server responded with HTTP 200`
   - Tap **Save Configuration**.

4. **Generate 6-Digit Pairing PIN on Web Dashboard**:
   - On your PC, navigate to `http://localhost:3000/phones` (or click **Pair New Phone** on the Dashboard).
   - A modal displays a secure, single-use 6-digit PIN (e.g. `582 914`) with a 10-minute expiry timer.

5. **Enter Code in Android App**:
   - In the mobile app, tap **Pair Gateway Device**.
   - Enter the 6-digit code.
   - Tap **Connect & Authenticate**.
   - The device will securely claim the session, receive a device API key (`zsms_dev_...`), store it in Android Keystore `EncryptedSharedPreferences`, and start `ZsmsGatewayService`.

6. **Verify Online Gateway Status**:
   - The app status badge will change to **ONLINE (CONNECTED)** in green.
   - On the Web Dashboard (`/phones` and `/dashboard`), the device will appear with its SIM carrier, battery level, and live **ONLINE** status.

---

## 5. End-to-End Functional Verification

### 5.1 Test 1: Outbound SMS Dispatch (Web Dashboard -> SIM -> External Phone)
1. Open the Web Dashboard at `http://localhost:3000/messages`.
2. Click **Send Real SMS**.
3. Select your paired Android phone from the gateway dropdown.
4. Enter your personal mobile phone number (E.164 format e.g. `+1234567890`).
5. Enter a test message: `ZSMS Stage 3 cellular gateway test from physical Android SIM.`
6. Click **Dispatch Real Cellular SMS**.
7. **Observe Execution**:
   - Server registers message with status `processing`.
   - The WebSocket instantly pushes the message to the Android service.
   - Android `SmsManager.sendMultipartTextMessage` transmits the message over the cellular radio.
   - `SmsSentReceiver` catches `Activity.RESULT_OK` and updates the server status to `sent`.
   - Your personal phone receives the real SMS!
   - `SmsDeliveredReceiver` catches the carrier SMSC delivery receipt and updates the server status to `delivered`.

### 5.2 Test 2: Inbound SMS Ingestion (External Phone -> SIM -> Web Dashboard)
1. From your personal mobile phone, reply to the message received from the Android gateway.
2. **Observe Execution**:
   - The Android cellular radio receives the SMS.
   - Android `SmsReceiver` intercepts `Telephony.SMS_RECEIVED`.
   - PDU is parsed, deduplicated, and POSTed to `/api/v1/android/messages/inbound`.
   - Server links message to the conversation thread, updates the snippet and increments unread count.
3. Open `http://localhost:3000/conversations`.
4. The incoming reply appears in real time in the conversation chat stream!

---

## 6. Physical Device Verification Matrix

| Verification Check | Stage Status | Expected Result | Pass Criteria |
| :--- | :--- | :--- | :--- |
| **Wi-Fi Reachability** | `IMPLEMENTED` / Ready for test | Phone reaches `http://<PC_LAN_IP>:8080/livez` | HTTP 200 with status ok |
| **Runtime Permissions** | `IMPLEMENTED` / Ready for test | SMS, Phone State, Notifications granted | No permission prompt cards visible |
| **SIM Detection** | `IMPLEMENTED` / Ready for test | App detects carrier (e.g. T-Mobile, Verizon) | Displayed on dashboard card |
| **Device Pairing** | `IMPLEMENTED` / Ready for test | 6-digit PIN claims single-use token | `zsms_dev_` token saved to Keystore |
| **WebSocket Connection** | `IMPLEMENTED` / Ready for test | Phone connects to `/api/v1/gateway/ws` | Notification shows active connection |
| **Outbound SMS** | `NOT TESTED` (Pending APK) | Radio broadcasts real SMS | Recipient phone buzzes with SMS |
| **Sent Callback** | `NOT TESTED` (Pending APK) | `SmsSentReceiver` fires on radio commit | Status transitions to `sent` |
| **Delivery Receipt** | `NOT TESTED` (Pending APK) | `SmsDeliveredReceiver` fires on SMSC confirmation | Status transitions to `delivered` |
| **Inbound SMS** | `NOT TESTED` (Pending APK) | `SmsReceiver` intercepts inbound PDU | Reply appears in `/conversations` |
| **Polling Fallback** | `IMPLEMENTED` / Ready for test | Disconnect WebSocket; queue SMS | Polling picks up and transmits SMS |
| **Heartbeat Loop** | `IMPLEMENTED` / Ready for test | Background loop sends telemetry every 30s | Phone stays online on Web dashboard |
| **Dual-SIM Dispatch** | `NOT TESTED` (Device Dependent) | Target SIM slot specified in request | SMS transmitted through selected SIM |

---

## 7. Troubleshooting & Diagnostics

- **Problem**: Android app says `Connection refused` when testing server ping.
  - **Solution**: Confirm both devices are on the exact same Wi-Fi SSID. Verify Windows Firewall rule (`netsh advfirewall firewall add rule name="ZSMS_API_Gateway" dir=in action=allow protocol=TCP localport=8080`). Ensure the server is listening on `0.0.0.0:8080`, not `127.0.0.1:8080`.
- **Problem**: SMS remains in `processing` state.
  - **Solution**: Check that the Android gateway service is running (persistent notification visible in status bar). If battery optimization is killing the service, go to **Settings > Apps > ZSMS > Battery** and choose **Unrestricted**.
- **Problem**: `RESULT_ERROR_GENERIC_FAILURE` or `RESULT_ERROR_RADIO_OFF`.
  - **Solution**: Ensure the physical SIM card is active, has cellular credit/SMS allowance, and Airplane Mode is OFF.
