package us.ztechai.zsms

import android.app.Application
import android.util.Log
import us.ztechai.zsms.data.network.ApiClient
import us.ztechai.zsms.data.storage.SecureStorage

/**
 * Main application entrypoint for ZSMS Android Gateway.
 */
class ZsmsApplication : Application() {

    lateinit var secureStorage: SecureStorage
        private set

    lateinit var apiClient: ApiClient
        private set

    override fun onCreate() {
        super.onCreate()
        Log.i(TAG, "Initializing ZSMS Android Gateway Application")

        // Initialize hardware-backed encrypted storage
        secureStorage = SecureStorage(this)

        // Initialize networking layer
        apiClient = ApiClient(secureStorage)
    }

    companion object {
        private const val TAG = "ZsmsApplication"
    }
}
