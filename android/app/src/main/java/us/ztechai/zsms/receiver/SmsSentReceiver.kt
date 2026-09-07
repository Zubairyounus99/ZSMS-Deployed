package us.ztechai.zsms.receiver

import android.app.Activity
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.telephony.SmsManager
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
 * Broadcast receiver triggered by SmsManager PendingIntent when the cellular radio
 * transmits the SMS or fails at the base station level.
 */
class SmsSentReceiver : BroadcastReceiver() {

    override fun onReceive(context: Context, intent: Intent) {
        val messageId = intent.getStringExtra(EXTRA_MESSAGE_ID) ?: return
        val app = context.applicationContext as? ZsmsApplication ?: return
        val storage = app.secureStorage
        val deviceToken = storage.getDeviceApiKey() ?: return

        val status: String
        var errorCode: String? = null
        var errorMessage: String? = null

        when (resultCode) {
            Activity.RESULT_OK -> {
                status = "sent"
                Log.i(TAG, "SMS successfully transmitted over cellular radio: $messageId")
            }
            SmsManager.RESULT_ERROR_GENERIC_FAILURE -> {
                status = "failed"
                errorCode = "GENERIC_FAILURE"
                errorMessage = "Radio generic failure. Check carrier airtime balance."
                Log.e(TAG, "SMS transmission failed (generic failure): $messageId")
            }
            SmsManager.RESULT_ERROR_NO_SERVICE -> {
                status = "failed"
                errorCode = "NO_SERVICE"
                errorMessage = "No cellular network service available."
                Log.e(TAG, "SMS transmission failed (no service): $messageId")
            }
            SmsManager.RESULT_ERROR_RADIO_OFF -> {
                status = "failed"
                errorCode = "RADIO_OFF"
                errorMessage = "Cellular radio is disabled or device is in Airplane Mode."
                Log.e(TAG, "SMS transmission failed (radio off): $messageId")
            }
            SmsManager.RESULT_ERROR_NULL_PDU -> {
                status = "failed"
                errorCode = "NULL_PDU"
                errorMessage = "Null PDU generated."
                Log.e(TAG, "SMS transmission failed (null PDU): $messageId")
            }
            else -> {
                status = "failed"
                errorCode = "RESULT_CODE_$resultCode"
                errorMessage = "Unknown carrier error code: $resultCode"
                Log.e(TAG, "SMS transmission failed (code $resultCode): $messageId")
            }
        }

        // Post result back to ZSMS backend
        val serverUrl = storage.getServerUrl()
        val endpoint = "$serverUrl/api/v1/android/messages/$messageId/result"

        val payload = JSONObject().apply {
            put("status", status)
            if (errorCode != null) put("error_code", errorCode)
            if (errorMessage != null) put("error_message", errorMessage)
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
                        Log.d(TAG, "SMS sent status reported to server: $messageId -> $status")
                    } else {
                        Log.e(TAG, "Server error reporting SMS status: HTTP ${response.code}")
                    }
                }
            } catch (e: Exception) {
                Log.e(TAG, "Failed to report SMS result to server: ${e.message}", e)
            } finally {
                pendingResult.finish()
            }
        }
    }

    companion object {
        private const val TAG = "SmsSentReceiver"
        const val ACTION_SMS_SENT = "us.ztechai.zsms.ACTION_SMS_SENT"
        const val EXTRA_MESSAGE_ID = "extra_message_id"
    }
}
