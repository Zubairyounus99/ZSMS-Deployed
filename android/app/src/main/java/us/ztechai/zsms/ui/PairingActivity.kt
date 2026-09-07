package us.ztechai.zsms.ui

import android.content.Context
import android.os.Build
import android.os.Bundle
import android.provider.Settings
import android.telephony.TelephonyManager
import android.view.View
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.lifecycleScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.json.JSONObject
import us.ztechai.zsms.ZsmsApplication
import us.ztechai.zsms.databinding.ActivityPairingBinding
import us.ztechai.zsms.service.ZsmsGatewayService
import java.util.UUID

/**
 * Activity for pairing the physical Android smartphone with the ZSMS server
 * using a single-use 6-digit pairing code generated on the Web Dashboard.
 */
class PairingActivity : AppCompatActivity() {

    private lateinit var binding: ActivityPairingBinding

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityPairingBinding.inflate(layoutInflater)
        setContentView(binding.root)

        val defaultName = "${Build.MANUFACTURER.replaceFirstChar { it.uppercase() }} ${Build.MODEL}"
        binding.etDeviceName.setText(defaultName)

        binding.btnSubmitPairing.setOnClickListener {
            val code = binding.etPairingCode.text?.toString()?.trim() ?: ""
            val name = binding.etDeviceName.text?.toString()?.trim().takeIf { !it.isNullOrBlank() } ?: defaultName

            if (code.length != 6) {
                showError("Please enter a valid 6-digit pairing code.")
                return@setOnClickListener
            }

            executePairing(code, name)
        }
    }

    private fun executePairing(pairingCode: String, deviceName: String) {
        setLoading(true)

        val app = application as ZsmsApplication
        val storage = app.secureStorage
        val client = app.apiClient.client
        val serverUrl = storage.getServerUrl().trimEnd('/')

        val telephonyManager = getSystemService(Context.TELEPHONY_SERVICE) as? TelephonyManager
        val carrierName = telephonyManager?.networkOperatorName?.takeIf { it.isNotBlank() }
            ?: telephonyManager?.simOperatorName?.takeIf { it.isNotBlank() }
            ?: "Cellular Provider"

        val androidId = Settings.Secure.getString(contentResolver, Settings.Secure.ANDROID_ID)
            ?: UUID.randomUUID().toString()

        lifecycleScope.launch(Dispatchers.IO) {
            try {
                val payload = JSONObject().apply {
                    put("pairing_code", pairingCode)
                    put("device_name", deviceName)
                    put("device_identifier", androidId)
                    put("phone_number", storage.getPhoneNumber() ?: "")
                    put("sim_carrier", carrierName)
                }

                val request = Request.Builder()
                    .url("$serverUrl/api/v1/pairing/complete")
                    .post(payload.toString().toRequestBody("application/json; charset=utf-8".toMediaType()))
                    .build()

                val response = client.newCall(request).execute()
                val responseBody = response.body?.string() ?: ""

                if (!response.isSuccessful) {
                    val errorMsg = try {
                        val json = JSONObject(responseBody)
                        json.optJSONObject("error")?.optString("message") ?: "Pairing request failed (${response.code})"
                    } catch (e: Exception) {
                        "Server returned HTTP ${response.code}"
                    }

                    withContext(Dispatchers.Main) {
                        setLoading(false)
                        showError(errorMsg)
                    }
                    return@launch
                }

                val responseJson = JSONObject(responseBody)
                val success = responseJson.optBoolean("success", false)
                if (!success) {
                    val msg = responseJson.optJSONObject("error")?.optString("message") ?: "Pairing unsuccessful."
                    withContext(Dispatchers.Main) {
                        setLoading(false)
                        showError(msg)
                    }
                    return@launch
                }

                val data = responseJson.getJSONObject("data")
                val phoneId = data.getString("phone_id")
                val deviceToken = data.getString("device_token")

                // Persist device identity securely
                storage.savePhoneId(phoneId)
                storage.saveDeviceApiKey(deviceToken)
                storage.saveDeviceName(deviceName)

                withContext(Dispatchers.Main) {
                    setLoading(false)
                    Toast.makeText(this@PairingActivity, "Gateway Paired Successfully!", Toast.LENGTH_LONG).show()

                    // Start persistent foreground service immediately
                    ZsmsGatewayService.start(this@PairingActivity)

                    finish()
                }

            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    setLoading(false)
                    showError("Connection failed: ${e.localizedMessage ?: "Unable to reach server. Check LAN IP settings."}")
                }
            }
        }
    }

    private fun setLoading(loading: Boolean) {
        binding.progressBar.visibility = if (loading) View.VISIBLE else View.GONE
        binding.btnSubmitPairing.isEnabled = !loading
        binding.tilPairingCode.isEnabled = !loading
        binding.tilDeviceName.isEnabled = !loading
        if (loading) {
            binding.tvError.visibility = View.GONE
        }
    }

    private fun showError(message: String) {
        binding.tvError.text = message
        binding.tvError.visibility = View.VISIBLE
    }
}
