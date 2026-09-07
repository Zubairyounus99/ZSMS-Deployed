package us.ztechai.zsms.receiver

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.provider.Telephony
import android.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import us.ztechai.zsms.ZsmsApplication
import java.text.SimpleDateFormat
import java.util.Collections
import java.util.Date
import java.util.Locale
import java.util.TimeZone

/**
 * Broadcast receiver listening for inbound SMS received on the physical SIM card.
 */
class SmsReceiver : BroadcastReceiver() {

    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Telephony.Sms.Intents.SMS_RECEIVED_ACTION) {
            return
        }

        val app = context.applicationContext as? ZsmsApplication ?: return
        val storage = app.secureStorage
        val deviceToken = storage.getDeviceApiKey()

        // If gateway device is not paired with a server, ignore
        if (deviceToken.isNullOrBlank()) {
            Log.w(TAG, "Inbound SMS received but device is not paired. Ignoring.")
            return
        }

        val messages = Telephony.Sms.Intents.getMessagesFromIntent(intent)
        if (messages.isNullOrEmpty()) {
            return
        }

        val sender = messages[0].displayOriginatingAddress ?: "Unknown"
        val bodyBuilder = StringBuilder()
        var timestampMs = messages[0].timestampMillis

        for (msg in messages) {
            bodyBuilder.append(msg.displayMessageBody)
            if (msg.timestampMillis > 0) {
                timestampMs = msg.timestampMillis
            }
        }

        val fullContent = bodyBuilder.toString()
        val dedupKey = "$sender:$timestampMs:$fullContent"

        // Deduplicate against multiple carrier/Android broadcast deliveries
        if (processedCache.contains(dedupKey)) {
            Log.d(TAG, "Duplicate inbound SMS broadcast suppressed: $dedupKey")
            return
        }
        processedCache.add(dedupKey)
        if (processedCache.size > 100) {
            processedCache.clear()
        }

        Log.i(TAG, "Inbound SMS received from physical SIM: sender=$sender, length=${fullContent.length}")

        // Format ISO timestamp
        val sdf = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss'Z'", Locale.US)
        sdf.timeZone = TimeZone.getTimeZone("UTC")
        val isoTimestamp = sdf.format(Date(timestampMs))

        val serverUrl = storage.getServerUrl()
        val endpoint = "$serverUrl/api/v1/android/messages/inbound"

        val payload = JSONObject().apply {
            put("sender", sender)
            put("recipient", storage.getPhoneNumber() ?: "")
            put("content", fullContent)
            put("sim_slot", 1)
            put("received_at", isoTimestamp)
        }

        // Post to server asynchronously
        val pendingResult = goAsync()
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val mediaType = "application/json; charset=utf-8".toMediaType()
                val requestBody = payload.toString().toRequestBody(mediaType)
                val request = Request.Builder()
                    .url(endpoint)
                    .post(requestBody)
                    .header("Authorization", "Bearer $deviceToken")
                    .header("Content-Type", "application/json")
                    .build()

                val client = app.apiClient.client
                client.newCall(request).execute().use { response ->
                    if (response.isSuccessful) {
                        Log.i(TAG, "Inbound SMS successfully ingested by ZSMS backend.")
                    } else {
                        Log.e(TAG, "Server rejected inbound SMS ingestion: HTTP ${response.code}")
                    }
                }
            } catch (e: Exception) {
                Log.e(TAG, "Failed to upload inbound SMS to server: ${e.message}", e)
            } finally {
                pendingResult.finish()
            }
        }
    }

    companion object {
        private const val TAG = "SmsReceiver"
        private val processedCache = Collections.synchronizedSet(mutableSetOf<String>())
    }
}
