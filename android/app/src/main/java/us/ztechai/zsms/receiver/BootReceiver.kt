package us.ztechai.zsms.receiver

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.os.Build
import android.util.Log
import us.ztechai.zsms.ZsmsApplication
import us.ztechai.zsms.service.ZsmsGatewayService

/**
 * Boot receiver to resume gateway operations automatically after device reboot.
 */
class BootReceiver : BroadcastReceiver() {

    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Intent.ACTION_BOOT_COMPLETED) {
            return
        }

        val app = context.applicationContext as? ZsmsApplication ?: return
        val deviceToken = app.secureStorage.getDeviceApiKey()

        if (deviceToken.isNullOrBlank()) {
            Log.i(TAG, "Device booted but is not paired with ZSMS server. Gateway service not started.")
            return
        }

        Log.i(TAG, "Device booted. Starting ZsmsGatewayService in foreground.")
        val serviceIntent = Intent(context, ZsmsGatewayService::class.java)

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            context.startForegroundService(serviceIntent)
        } else {
            context.startService(serviceIntent)
        }
    }

    companion object {
        private const val TAG = "BootReceiver"
    }
}
