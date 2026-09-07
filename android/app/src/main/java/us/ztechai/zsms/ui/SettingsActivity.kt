package us.ztechai.zsms.ui

import android.graphics.Color
import android.os.Bundle
import android.view.View
import android.widget.Toast
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.lifecycle.lifecycleScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import okhttp3.OkHttpClient
import okhttp3.Request
import us.ztechai.zsms.ZsmsApplication
import us.ztechai.zsms.databinding.ActivitySettingsBinding
import us.ztechai.zsms.service.ZsmsGatewayService
import java.util.concurrent.TimeUnit

/**
 * Activity for configuring LAN server endpoints and managing gateway pairing state.
 */
class SettingsActivity : AppCompatActivity() {

    private lateinit var binding: ActivitySettingsBinding

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivitySettingsBinding.inflate(layoutInflater)
        setContentView(binding.root)

        val app = application as ZsmsApplication
        val storage = app.secureStorage

        binding.etServerUrl.setText(storage.getServerUrl())

        val isPaired = !storage.getPhoneId().isNullOrBlank()
        binding.btnUnpair.visibility = if (isPaired) View.VISIBLE else View.GONE

        binding.btnTestPing.setOnClickListener {
            val url = binding.etServerUrl.text?.toString()?.trim() ?: ""
            if (url.isBlank()) {
                binding.tvPingResult.text = "Please enter a valid Server URL."
                binding.tvPingResult.setTextColor(Color.RED)
                return@setOnClickListener
            }
            testConnection(url)
        }

        binding.btnSaveSettings.setOnClickListener {
            var url = binding.etServerUrl.text?.toString()?.trim() ?: ""
            if (!url.startsWith("http://") && !url.startsWith("https://")) {
                url = "https://$url"
            }
            storage.saveServerUrl(url)
            Toast.makeText(this, "Settings saved.", Toast.LENGTH_SHORT).show()

            if (isPaired) {
                // Restart service to pick up new URL
                ZsmsGatewayService.stop(this)
                ZsmsGatewayService.start(this)
            }

            finish()
        }

        binding.btnUnpair.setOnClickListener {
            AlertDialog.Builder(this)
                .setTitle("Unpair Gateway")
                .setMessage("Are you sure you want to unpair this phone? It will stop listening for and dispatching cellular SMS.")
                .setPositiveButton("Unpair") { _, _ ->
                    ZsmsGatewayService.stop(this)
                    storage.clearAll()
                    Toast.makeText(this, "Device unpaired.", Toast.LENGTH_SHORT).show()
                    finish()
                }
                .setNegativeButton("Cancel", null)
                .show()
        }
    }

    private fun testConnection(serverUrl: String) {
        binding.tvPingResult.text = "Testing connection..."
        binding.tvPingResult.setTextColor(Color.DKGRAY)
        binding.btnTestPing.isEnabled = false

        val testClient = OkHttpClient.Builder()
            .connectTimeout(5, TimeUnit.SECONDS)
            .readTimeout(5, TimeUnit.SECONDS)
            .build()

        lifecycleScope.launch(Dispatchers.IO) {
            try {
                val pingUrl = "${serverUrl.trimEnd('/')}/livez"
                val request = Request.Builder().url(pingUrl).build()
                val response = testClient.newCall(request).execute()

                withContext(Dispatchers.Main) {
                    binding.btnTestPing.isEnabled = true
                    if (response.isSuccessful) {
                        binding.tvPingResult.text = "CONNECTED! Server responded with HTTP ${response.code}"
                        binding.tvPingResult.setTextColor(Color.parseColor("#00875A"))
                    } else {
                        binding.tvPingResult.text = "UNREACHABLE: Server returned HTTP ${response.code}"
                        binding.tvPingResult.setTextColor(Color.RED)
                    }
                }
            } catch (e: Exception) {
                withContext(Dispatchers.Main) {
                    binding.btnTestPing.isEnabled = true
                    binding.tvPingResult.text = "ERROR: ${e.localizedMessage ?: "Connection refused. Check LAN IP and Windows Firewall."}"
                    binding.tvPingResult.setTextColor(Color.RED)
                }
            }
        }
    }
}
