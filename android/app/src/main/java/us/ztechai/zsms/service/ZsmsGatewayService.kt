package us.ztechai.zsms.service

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.content.pm.ServiceInfo
import android.os.BatteryManager
import android.os.Build
import android.os.Handler
import android.os.IBinder
import android.os.Looper
import android.telephony.SmsManager
import android.util.Log
import androidx.core.app.NotificationCompat
import androidx.core.app.ServiceCompat
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import org.json.JSONArray
import org.json.JSONObject
import us.ztechai.zsms.MainActivity
import us.ztechai.zsms.ZsmsApplication
import us.ztechai.zsms.receiver.SmsDeliveredReceiver
import us.ztechai.zsms.receiver.SmsSentReceiver
import java.util.Collections
import java.util.concurrent.atomic.AtomicBoolean

/**
 * Real persistent foreground service operating the cellular SMS gateway.
 * Maintains real-time WebSocket connection with ZSMS server, dispatches
 * outbound SMS commands via physical SIM SmsManager, and emits periodic heartbeats.
 */
class ZsmsGatewayService : Service() {

    private val serviceScope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    private var webSocket: WebSocket? = null
    private val isConnected = AtomicBoolean(false)
    private val isRunning = AtomicBoolean(false)
    private val processedMessages = Collections.synchronizedSet(mutableSetOf<String>())

    override fun onCreate() {
        super.onCreate()
        Log.i(TAG, "ZsmsGatewayService initializing")
        createNotificationChannel()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        if (isRunning.compareAndSet(false, true)) {
            val notification = buildForegroundNotification("Connecting to ZSMS server...")
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
                val serviceType = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.UPSIDE_DOWN_CAKE) {
                    ServiceInfo.FOREGROUND_SERVICE_TYPE_SPECIAL_USE
                } else {
                    0
                }
                ServiceCompat.startForeground(this, NOTIFICATION_ID, notification, serviceType)
            } else {
                startForeground(NOTIFICATION_ID, notification)
            }
            startGatewayLoop()
        }
        return START_STICKY
    }

    private fun startGatewayLoop() {
        val app = application as? ZsmsApplication ?: return
        val storage = app.secureStorage
        val deviceToken = storage.getDeviceApiKey()
        val phoneId = storage.getPhoneId()

        if (deviceToken.isNullOrBlank() || phoneId.isNullOrBlank()) {
            Log.w(TAG, "Cannot start gateway service: Device is not paired.")
            updateNotification("Gateway idle (Pairing required)")
            broadcastStatus(false, "Device not paired")
            return
        }

        // 1. Maintain WebSocket connection with auto-reconnect
        serviceScope.launch {
            var reconnectDelay = 3000L
            while (isActive) {
                try {
                    connectWebSocket(storage.getServerUrl(), deviceToken)
                } catch (e: Exception) {
                    Log.e(TAG, "WebSocket connection attempt error: ${e.message}")
                }

                // Wait until disconnected before attempting reconnect
                while (isActive && isConnected.get()) {
                    delay(2000L)
                }

                if (isActive) {
                    Log.i(TAG, "Reconnecting WebSocket in ${reconnectDelay / 1000}s...")
                    updateNotification("Reconnecting to server...")
                    broadcastStatus(false, "Reconnecting")
                    delay(reconnectDelay)
                    reconnectDelay = (reconnectDelay * 2).coerceAtMost(30000L)
                }
            }
        }

        // 2. Periodic HTTP Polling Fallback (ensures 100% command delivery even if WebSocket is interrupted)
        serviceScope.launch {
            while (isActive) {
                delay(15000L)
                if (!isConnected.get()) {
                    pollPendingCommands(storage.getServerUrl(), deviceToken)
                }
            }
        }

        // 3. Periodic Heartbeat Runner (every 30s)
        serviceScope.launch {
            while (isActive) {
                sendHeartbeat(storage.getServerUrl(), phoneId, deviceToken)
                delay(30000L)
            }
        }
    }

    private fun connectWebSocket(serverUrl: String, deviceToken: String) {
        val wsUrl = serverUrl
            .replace("http://", "ws://")
            .replace("https://", "wss://") + "/api/v1/gateway/ws"

        Log.i(TAG, "Initiating secure WebSocket connection to: $wsUrl")
        val app = application as ZsmsApplication
        val request = Request.Builder()
            .url(wsUrl)
            .header("Authorization", "Bearer $deviceToken")
            .build()

        webSocket = app.apiClient.client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                Log.i(TAG, "WebSocket connected successfully to ZSMS backend")
                isConnected.set(true)
                updateNotification("Gateway ONLINE (Active cellular bridge)")
                broadcastStatus(true, "ONLINE")
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                Log.i(TAG, "Received real-time command from server: $text")
                handleCommand(text)
            }

            override fun onClosing(webSocket: WebSocket, code: Int, reason: String) {
                Log.w(TAG, "WebSocket closing: $code / $reason")
                isConnected.set(false)
                broadcastStatus(false, "Disconnecting")
            }

            override fun onClosed(webSocket: WebSocket, code: Int, reason: String) {
                Log.w(TAG, "WebSocket closed: $code / $reason")
                isConnected.set(false)
                broadcastStatus(false, "OFFLINE")
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                Log.e(TAG, "WebSocket connection failure: ${t.message}")
                isConnected.set(false)
                broadcastStatus(false, "Connection error")
            }
        })
    }

    private fun handleCommand(payloadJson: String) {
        try {
            val json = JSONObject(payloadJson)
            val type = json.optString("type", "")

            if (type == "sms.send") {
                val messageId = json.getString("message_id")
                val recipient = json.getString("recipient")
                val content = json.getString("content")
                val simSlot = json.optInt("sim_slot", 1)

                sendPhysicalSms(messageId, recipient, content, simSlot)
            }
        } catch (e: Exception) {
            Log.e(TAG, "Failed to parse command JSON: ${e.message}", e)
        }
    }

    /**
     * Executes real SMS transmission through the physical SIM card using SmsManager.
     */
    private fun sendPhysicalSms(messageId: String, recipient: String, content: String, simSlot: Int) {
        // Idempotency: Prevent duplicate transmission
        if (processedMessages.contains(messageId)) {
            Log.w(TAG, "Duplicate SMS send command ignored: $messageId")
            return
        }
        processedMessages.add(messageId)
        if (processedMessages.size > 200) {
            processedMessages.clear()
        }

        Log.i(TAG, "Transmitting REAL SMS: messageId=$messageId, to=$recipient, length=${content.length}")

        try {
            val smsManager: SmsManager = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
                applicationContext.getSystemService(SmsManager::class.java)
            } else {
                @Suppress("DEPRECATION")
                SmsManager.getDefault()
            }

            // Divide message into GSM 7-bit or UCS-2 multipart segments
            val parts = smsManager.divideMessage(content)
            val numParts = parts.size

            val sentIntents = ArrayList<PendingIntent>()
            val deliveryIntents = ArrayList<PendingIntent>()

            for (i in 0 until numParts) {
                // Sent intent
                val sentIntent = Intent(applicationContext, SmsSentReceiver::class.java).apply {
                    action = SmsSentReceiver.ACTION_SMS_SENT
                    putExtra(SmsSentReceiver.EXTRA_MESSAGE_ID, messageId)
                }
                val flags = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
                    PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
                } else {
                    PendingIntent.FLAG_UPDATE_CURRENT
                }
                val sentPI = PendingIntent.getBroadcast(applicationContext, messageId.hashCode() + i, sentIntent, flags)
                sentIntents.add(sentPI)

                // Delivery intent
                val deliveryIntent = Intent(applicationContext, SmsDeliveredReceiver::class.java).apply {
                    action = SmsDeliveredReceiver.ACTION_SMS_DELIVERED
                    putExtra(SmsDeliveredReceiver.EXTRA_MESSAGE_ID, messageId)
                }
                val deliveryPI = PendingIntent.getBroadcast(applicationContext, messageId.hashCode() + 1000 + i, deliveryIntent, flags)
                deliveryIntents.add(deliveryPI)
            }

            // Inject into Android Telephony hardware stack
            smsManager.sendMultipartTextMessage(recipient, null, parts, sentIntents, deliveryIntents)
            Log.i(TAG, "SmsManager.sendMultipartTextMessage submitted ($numParts segments) for messageId: $messageId")

        } catch (e: Exception) {
            Log.e(TAG, "Hardware SmsManager error sending SMS: ${e.message}", e)
            // Report immediate failure back to backend
            reportFailure(messageId, "RADIO_EXCEPTION", e.message ?: "Failed to transmit via SmsManager")
        }
    }

    private fun reportFailure(messageId: String, errCode: String, errMsg: String) {
        val app = application as ZsmsApplication
        val storage = app.secureStorage
        val deviceToken = storage.getDeviceApiKey() ?: return
        val url = "${storage.getServerUrl()}/api/v1/android/messages/$messageId/result"

        serviceScope.launch {
            try {
                val payload = JSONObject().apply {
                    put("status", "failed")
                    put("error_code", errCode)
                    put("error_message", errMsg)
                }
                val mediaType = "application/json; charset=utf-8".toMediaType()
                val request = Request.Builder()
                    .url(url)
                    .post(payload.toString().toRequestBody(mediaType))
                    .header("Authorization", "Bearer $deviceToken")
                    .build()
                app.apiClient.client.newCall(request).execute().close()
            } catch (e: Exception) {
                Log.e(TAG, "Failed to report failure to server: ${e.message}")
            }
        }
    }

    private fun pollPendingCommands(serverUrl: String, deviceToken: String) {
        val app = application as ZsmsApplication
        val endpoint = "$serverUrl/api/v1/android/messages/poll"
        try {
            val request = Request.Builder()
                .url(endpoint)
                .get()
                .header("Authorization", "Bearer $deviceToken")
                .build()

            app.apiClient.client.newCall(request).execute().use { response ->
                if (response.isSuccessful) {
                    val body = response.body?.string() ?: return
                    val json = JSONObject(body)
                    val dataArray: JSONArray = json.optJSONArray("data") ?: return
                    for (i in 0 until dataArray.length()) {
                        val cmdObj = dataArray.getJSONObject(i)
                        handleCommand(cmdObj.toString())
                    }
                }
            }
        } catch (e: Exception) {
            Log.d(TAG, "Polling fallback check error: ${e.message}")
        }
    }

    private fun sendHeartbeat(serverUrl: String, phoneId: String, deviceToken: String) {
        val app = application as ZsmsApplication
        val endpoint = "$serverUrl/api/v1/phones/$phoneId/heartbeat"

        val batteryLevel = getBatteryLevel()
        val isCharging = isBatteryCharging()
        val networkType = getNetworkType()

        try {
            val payload = JSONObject().apply {
                put("battery_level", batteryLevel)
                put("battery_charging", isCharging)
                put("network_type", networkType)
                put("app_version", "1.0.0")
                put("os_version", "Android ${Build.VERSION.RELEASE} (API ${Build.VERSION.SDK_INT})")
            }

            val mediaType = "application/json; charset=utf-8".toMediaType()
            val request = Request.Builder()
                .url(endpoint)
                .post(payload.toString().toRequestBody(mediaType))
                .header("Authorization", "Bearer $deviceToken")
                .build()

            app.apiClient.client.newCall(request).execute().use { response ->
                if (response.isSuccessful) {
                    Log.d(TAG, "Heartbeat recorded: battery=$batteryLevel%, net=$networkType")
                }
            }
        } catch (e: Exception) {
            Log.d(TAG, "Heartbeat network error: ${e.message}")
        }
    }

    private fun getBatteryLevel(): Int {
        val batteryIntent = registerReceiver(null, IntentFilter(Intent.ACTION_BATTERY_CHANGED))
        val level = batteryIntent?.getIntExtra(BatteryManager.EXTRA_LEVEL, -1) ?: -1
        val scale = batteryIntent?.getIntExtra(BatteryManager.EXTRA_SCALE, -1) ?: -1
        return if (level >= 0 && scale > 0) (level * 100) / scale else 100
    }

    private fun isBatteryCharging(): Boolean {
        val batteryIntent = registerReceiver(null, IntentFilter(Intent.ACTION_BATTERY_CHANGED))
        val status = batteryIntent?.getIntExtra(BatteryManager.EXTRA_STATUS, -1) ?: -1
        return status == BatteryManager.BATTERY_STATUS_CHARGING || status == BatteryManager.BATTERY_STATUS_FULL
    }

    private fun getNetworkType(): String {
        val cm = getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager ?: return "unknown"
        val activeNetwork = cm.activeNetwork ?: return "offline"
        val capabilities = cm.getNetworkCapabilities(activeNetwork) ?: return "offline"
        return when {
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) -> "wifi"
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) -> "cellular"
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_ETHERNET) -> "ethernet"
            else -> "other"
        }
    }

    private fun broadcastStatus(online: Boolean, message: String) {
        val intent = Intent(ACTION_GATEWAY_STATUS_CHANGED).apply {
            putExtra(EXTRA_IS_ONLINE, online)
            putExtra(EXTRA_STATUS_MESSAGE, message)
        }
        sendBroadcast(intent)
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "ZSMS Cellular Gateway Service",
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = "Shows active cellular gateway connection status"
            }
            val manager = getSystemService(NotificationManager::class.java)
            manager.createNotificationChannel(channel)
        }
    }

    private fun buildForegroundNotification(statusText: String): Notification {
        val openAppIntent = Intent(this, MainActivity::class.java)
        val flags = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE
        } else {
            PendingIntent.FLAG_UPDATE_CURRENT
        }
        val pendingIntent = PendingIntent.getActivity(this, 0, openAppIntent, flags)

        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle("ZSMS Cellular Gateway")
            .setContentText(statusText)
            .setSmallIcon(android.R.drawable.stat_notify_chat)
            .setContentIntent(pendingIntent)
            .setOngoing(true)
            .build()
    }

    private fun updateNotification(statusText: String) {
        val manager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
        manager.notify(NOTIFICATION_ID, buildForegroundNotification(statusText))
    }

    override fun onDestroy() {
        super.onDestroy()
        Log.i(TAG, "ZsmsGatewayService stopping")
        isRunning.set(false)
        isConnected.set(false)
        webSocket?.close(1000, "Service stopped")
        serviceScope.cancel()
    }

    override fun onBind(intent: Intent?): IBinder? = null

    companion object {
        private const val TAG = "ZsmsGatewayService"
        private const val CHANNEL_ID = "zsms_gateway_channel"
        private const val NOTIFICATION_ID = 1001

        const val ACTION_GATEWAY_STATUS_CHANGED = "us.ztechai.zsms.GATEWAY_STATUS_CHANGED"
        const val EXTRA_IS_ONLINE = "extra_is_online"
        const val EXTRA_STATUS_MESSAGE = "extra_status_message"

        fun start(context: Context) {
            val intent = Intent(context, ZsmsGatewayService::class.java)
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                context.startForegroundService(intent)
            } else {
                context.startService(intent)
            }
        }

        fun stop(context: Context) {
            val intent = Intent(context, ZsmsGatewayService::class.java)
            context.stopService(intent)
        }
    }
}
