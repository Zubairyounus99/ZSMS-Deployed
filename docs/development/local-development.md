# Local Development & Physical Cellular Gateway Testing Guide

This document provides an exact, reproducible, step-by-step procedure to set up, build, deploy, and physically test the **ZSMS** platform locally on Windows with a physical Android phone.

---

## 1. Prerequisites & Host Verification

Before running ZSMS, ensure the required developer tools are installed. Verify your installation in PowerShell:

```powershell
# 1. Version Control
git --version

# 2. Containerization (For local PostgreSQL & Redis)
docker --version
docker compose version

# 3. Web Frontend Tooling
node --version
npm.cmd --version

# 4. Backend Go Tooling
go version

# 5. Android Development Tooling
java -version
adb version
```

### Required Tool Specifications:
* **Node.js**: `v20.x` or higher (includes `npm`)
* **Go**: `1.22.x` or higher
* **Docker Desktop**: With WSL2 backend enabled
* **JDK**: Version 17 (e.g. Eclipse Adoptium Temurin 17 or Microsoft OpenJDK 17)
* **Android Studio**: Jellyfish / Iguana with Android SDK Platform 34 and Platform Tools (`adb`)

---

## 2. Step-by-Step Copy-Paste Deployment Procedure

### Step 1: Clone or Open Repository
```powershell
cd d:\Zubair\Antigravity\ZSMS
```

### Step 2: Create Local Environment Configuration
```powershell
Copy-Item .env.example .env
```
*(No edits are required for standard local development defaults).*

### Step 3: Start PostgreSQL & Redis Infrastructure
```powershell
docker compose -f docker-compose.dev.yml up -d postgres redis
```
Verify container health:
```powershell
docker compose -f docker-compose.dev.yml ps
```
Inspect database logs if needed:
```powershell
docker compose -f docker-compose.dev.yml logs -f postgres
```

### Step 4: Run Database Migrations
The initial schema is automatically executed from `docker/postgres/init.sql` on the first volume boot. If using the Golang migrate CLI:
```powershell
# Optional: migrate -path migrations -database "postgres://zsms_user:zsms_dev_password@localhost:5432/zsms_db?sslmode=disable" up
```

### Step 5: Start the Go Backend API
Open Terminal 1:
```powershell
cd d:\Zubair\Antigravity\ZSMS\backend
go run ./api/main.go
```
Verify API health in your browser or curl:
```powershell
Invoke-RestMethod -Uri "http://localhost:8080/livez"
```
Output:
```json
{"status":"ok","timestamp":"...","version":"1.0.0"}
```

### Step 6: Start the Nuxt 3 Web Dashboard
Open Terminal 2:
```powershell
cd d:\Zubair\Antigravity\ZSMS\web
npm.cmd run dev
```
Access the web dashboard at: [http://localhost:3000](http://localhost:3000).

---

## 3. Physical Android Device LAN Networking

Because a physical smartphone runs on its own isolated baseband OS, `http://localhost:8080` points to the phone itself. You must connect your phone to the same Wi-Fi router as your Windows PC.

### Step 7: Find Your Windows PC LAN IP
```powershell
ipconfig
```
Identify your active IPv4 Address (e.g., `192.168.1.100`).

### Step 8: Open Windows Defender Firewall for Port 8080
Run PowerShell as **Administrator**:
```powershell
netsh advfirewall firewall add rule name="ZSMS_API_Gateway" dir=in action=allow protocol=TCP localport=8080
```

---

## 4. Android Gateway Build & Installation

### Step 9: Build the Debug APK

#### Method A: Using Android Studio (Recommended)
1. Launch **Android Studio**.
2. Select **Open** and select the folder `d:\Zubair\Antigravity\ZSMS\android`.
3. Allow Gradle sync to complete.
4. From the top menu, select: **Build > Build Bundle(s) / APK(s) > Build APK(s)**.

#### Method B: Using Gradle CLI
```powershell
cd d:\Zubair\Antigravity\ZSMS\android
.\gradlew.bat assembleDebug
```
The output APK will be located at:
`d:\Zubair\Antigravity\ZSMS\android\app\build\outputs\apk\debug\app-debug.apk`

### Step 10: Install APK on Physical Phone
Connect your physical Android phone via USB and enable **USB Debugging** in Developer Settings:
```powershell
adb devices
adb install -r android/app/build/outputs/apk/debug/app-debug.apk
```

---

## 5. Physical Verification & Real Cellular Testing

### Step 11: Configure Android App Server URL
1. Open the **ZSMS** app on your physical Android phone.
2. Grant all requested SMS, Phone State, and Notification permissions.
3. Tap **Server Settings**.
4. Set **Server URL** to your Windows LAN IP:
   ```text
   http://192.168.1.100:8080
   ```
5. Tap **Test Server Connection**. You must see:
   `CONNECTED! Server responded with HTTP 200`
6. Tap **Save Configuration**.

### Step 12: Pair Gateway Device
1. On your PC, go to [http://localhost:3000/phones](http://localhost:3000/phones) and click **Pair New Phone**.
2. Note the single-use 6-digit PIN (e.g., `582 914`).
3. In the mobile app, tap **Pair Gateway Device**, enter the PIN, and tap **Connect & Authenticate**.
4. The phone receives its device token, starts `ZsmsGatewayService`, and displays **ONLINE (CONNECTED)** in green.

### Step 13: Outbound Real Cellular SMS Test
1. On the Web Dashboard, navigate to [http://localhost:3000/messages](http://localhost:3000/messages).
2. Click **Send Real SMS**.
3. Select your paired phone from the dropdown.
4. Enter an external recipient mobile phone number (E.164 format, e.g. `+1234567890`).
5. Enter: `ZSMS physical cellular test 001`.
6. Click **Dispatch Real Cellular SMS**.
7. **Verify**:
   - The recipient handset buzzes and receives the physical SMS!
   - On the web dashboard, the message transitions: `processing` -> `sent` -> `delivered`.

### Step 14: Inbound Real Cellular SMS Test
1. From the external mobile handset, reply: `ZSMS inbound physical test 001`.
2. **Verify**:
   - The Android physical SIM receives the carrier SMS.
   - `SmsReceiver` intercepts the PDU and posts to `/api/v1/android/messages/inbound`.
   - On your PC, navigate to [http://localhost:3000/conversations](http://localhost:3000/conversations).
   - The reply appears instantly in the two-way conversation thread!

---

## 6. Troubleshooting Matrix

| Issue | Cause | Resolution |
| :--- | :--- | :--- |
| **Connection Refused on Phone** | Windows Firewall or wrong IP | Verify firewall rule with `netsh advfirewall firewall show rule name="ZSMS_API_Gateway"`. Confirm PC and phone are on the exact same Wi-Fi SSID. |
| **`JAVA_HOME is not set` during build** | JDK 17 missing from host PATH | Install JDK 17 (e.g. Eclipse Adoptium Temurin 17) and set `JAVA_HOME=C:\Program Files\Eclipse Adoptium\jdk-17...`. |
| **SMS stays in `processing`** | Gateway service killed by Android | Go to `Settings > Apps > ZSMS > Battery` on phone and select **Unrestricted**. |
| **`RESULT_ERROR_GENERIC_FAILURE`** | SIM credit / radio off | Ensure phone is not in Airplane Mode and the physical SIM card has active cellular credit / plan. |
