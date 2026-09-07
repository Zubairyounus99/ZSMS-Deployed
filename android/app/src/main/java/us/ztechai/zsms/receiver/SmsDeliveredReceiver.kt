package us.ztechai.zsms.receiver

import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import us.ztechai.zsms.ZsmsApplication

/**
 * Broadcast receiver triggered when the carrier network confirms delivery to the recipient handset.
 */
class SmsDeliveredReceiver : BroadcastReceiver() {

    override fun onReceive(context: Context, intent: Intent) {
        val messageId = intent.getStringExtra(EXTRA_MESSAGE_ID) ?: return
        val app = context.applicationContext as? ZsmsApplication ?: return
        val storage = app.secureStorage
        val deviceToken = storage.getDeviceApiKey() ?: return

        val isDelivered = (resultCode == Activity.RESULT_OK)
        val status = if (isDelivered) "delivered" else "failed"

        Log.i(TAG, "Carrier SMS delivery confirmation received: messageId=$messageId, delivered=$isDelivered")

        val serverUrl = storage.getServerUrl()
        val endpoint = "$serverUrl/api/v1/android/messages/$messageId/result"

        val payload = JSONObject().apply {
            put("status", status)
            if (!isDelivered) {
                put("error_code", "DELIVERY_PDU_FAILED")
                put("error_message", "Carrier SMSC reported failure delivering to destination handset.")
            }
        }

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

                app.apiClient.client.newCall(request).execute().use { response ->
                    if (response.isSuccessful) {
                        Log.d(TAG, "SMS delivery status reported to server: $messageId -> $status")
                    }
                }
            } catch (e: Exception) {
                Log.e(TAG, "Failed to report delivery receipt: ${e.message}", e)
            } finally {
                pendingResult.finish()
            }
        }
    }

    companion object {
        private const val TAG = "SmsDeliveredReceiver"
        const val ACTION_SMS_DELIVERED = "us.ztechai.zsms.ACTION_SMS_DELIVERED"
        const val EXTRA_MESSAGE_ID = "extra_message_id"
    }
}
