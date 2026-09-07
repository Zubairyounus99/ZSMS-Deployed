package us.ztechai.zsms

import android.Manifest
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.PackageManager
import android.graphics.Color
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.os.BatteryManager
import android.os.Build
import android.os.Bundle
import android.telephony.TelephonyManager
import android.view.View
import android.widget.Toast
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.core.content.ContextCompat
import us.ztechai.zsms.databinding.ActivityMainBinding
import us.ztechai.zsms.service.ZsmsGatewayService
import us.ztechai.zsms.ui.PairingActivity
import us.ztechai.zsms.ui.SettingsActivity

/**
 * Main dashboard activity providing real-time gateway status, telephony hardware
 * diagnostics, permission management, and navigation to pairing/settings.
 */
class MainActivity : AppCompatActivity() {

    private lateinit var binding: ActivityMainBinding

    private val requiredPermissions: Array<String>
        get() {
            val list = mutableListOf(
                Manifest.permission.SEND_SMS,
                Manifest.permission.RECEIVE_SMS,
                Manifest.permission.READ_SMS,
                Manifest.permission.READ_PHONE_STATE
            )
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
                list.add(Manifest.permission.POST_NOTIFICATIONS)
            }
            return list.toTypedArray()
        }

    private val permissionLauncher = registerForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions()
    ) { results ->
        val allGranted = results.values.all { it }
        if (allGranted) {
            binding.cardPermissions.visibility = View.GONE
            Toast.makeText(this, "All SMS & Telephony permissions granted.", Toast.LENGTH_SHORT).show()
            val app = application as ZsmsApplication
            if (!app.secureStorage.getPhoneId().isNullOrBlank()) {
                ZsmsGatewayService.start(this)
            }
        } else {
            binding.cardPermissions.visibility = View.VISIBLE
            Toast.makeText(this, "SMS permissions are required for the gateway to function.", Toast.LENGTH_LONG).show()
        }
        updateDiagnostics()
    }

    private val gatewayStatusReceiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context?, intent: Intent?) {
            if (intent?.action == ZsmsGatewayService.ACTION_GATEWAY_STATUS_CHANGED) {
                val isOnline = intent.getBooleanExtra(ZsmsGatewayService.EXTRA_IS_ONLINE, false)
                val statusMsg = intent.getStringExtra(ZsmsGatewayService.EXTRA_STATUS_MESSAGE) ?: "CONNECTED"
                updateConnectionBadge(isOnline, statusMsg)
            }
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        binding = ActivityMainBinding.inflate(layoutInflater)
        setContentView(binding.root)

        setupListeners()
    }

    override fun onResume() {
        super.onResume()
        checkPermissions()
        updateGatewayState()
        updateDiagnostics()

        val filter = IntentFilter(ZsmsGatewayService.ACTION_GATEWAY_STATUS_CHANGED)
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            registerReceiver(gatewayStatusReceiver, filter, Context.RECEIVER_NOT_EXPORTED)
        } else {
            registerReceiver(gatewayStatusReceiver, filter)
        }
    }

    override fun onPause() {
        super.onPause()
        try {
            unregisterReceiver(gatewayStatusReceiver)
        } catch (e: IllegalArgumentException) {
            // Receiver not registered
        }
    }

    private fun setupListeners() {
        binding.btnGrantPermissions.setOnClickListener {
            permissionLauncher.launch(requiredPermissions)
        }

        binding.btnPairPhone.setOnClickListener {
            startActivity(Intent(this, PairingActivity::class.java))
        }

        binding.btnSettings.setOnClickListener {
            startActivity(Intent(this, SettingsActivity::class.java))
        }
    }

    private fun checkPermissions() {
        val missingPermissions = requiredPermissions.filter {
            ContextCompat.checkSelfPermission(this, it) != PackageManager.PERMISSION_GRANTED
        }

        if (missingPermissions.isNotEmpty()) {
            binding.cardPermissions.visibility = View.VISIBLE
        } else {
            binding.cardPermissions.visibility = View.GONE
        }
    }

    private fun updateGatewayState() {
        val app = application as ZsmsApplication
        val storage = app.secureStorage
        val phoneId = storage.getPhoneId()
        val isPaired = !phoneId.isNullOrBlank()

        binding.tvServerUrl.text = "Server: ${storage.getServerUrl()}"

        if (isPaired) {
            binding.tvDeviceName.text = storage.getDeviceName()
            binding.tvPhoneId.text = "ID: $phoneId"
            binding.btnPairPhone.text = "Re-Pair Gateway"
            updateConnectionBadge(true, "ACTIVE")

            // Automatically ensure gateway foreground service is running
            val hasPermissions = requiredPermissions.all {
                ContextCompat.checkSelfPermission(this, it) == PackageManager.PERMISSION_GRANTED
            }
            if (hasPermissions) {
                ZsmsGatewayService.start(this)
            }
        } else {
            binding.tvDeviceName.text = "Not Paired"
            binding.tvPhoneId.text = "ID: --"
            binding.btnPairPhone.text = "Pair Gateway Device"
            updateConnectionBadge(false, "NOT PAIRED")
        }
    }

    private fun updateConnectionBadge(isOnline: Boolean, message: String) {
        if (isOnline) {
            binding.tvStatusBadge.text = "ONLINE ($message)"
            binding.tvStatusBadge.setTextColor(Color.parseColor("#00875A")) // Green
        } else {
            binding.tvStatusBadge.text = message.uppercase()
            binding.tvStatusBadge.setTextColor(Color.parseColor("#FF5630")) // Red / Orange
        }
    }

    private fun updateDiagnostics() {
        // SIM & Carrier Info
        val telephonyManager = getSystemService(Context.TELEPHONY_SERVICE) as? TelephonyManager
        val carrier = telephonyManager?.networkOperatorName?.takeIf { it.isNotBlank() }
            ?: telephonyManager?.simOperatorName?.takeIf { it.isNotBlank() }
            ?: "Detecting or No Physical SIM"
        binding.tvCarrierInfo.text = "SIM Carrier: $carrier"

        // Battery level
        val batteryManager = getSystemService(Context.BATTERY_SERVICE) as? BatteryManager
        val batteryLevel = batteryManager?.getIntProperty(BatteryManager.BATTERY_PROPERTY_CAPACITY) ?: -1
        binding.tvBatteryInfo.text = "Battery: ${if (batteryLevel >= 0) "$batteryLevel%" else "Unknown"}"

        // Network connection type
        val connectivityManager = getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager
        val activeNetwork = connectivityManager?.activeNetwork
        val capabilities = connectivityManager?.getNetworkCapabilities(activeNetwork)
        val networkType = when {
            capabilities?.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) == true -> "Wi-Fi LAN"
            capabilities?.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) == true -> "Cellular Radio"
            capabilities?.hasTransport(NetworkCapabilities.TRANSPORT_ETHERNET) == true -> "Ethernet"
            else -> "Disconnected"
        }
        binding.tvNetworkInfo.text = "Network: $networkType"
    }
}
