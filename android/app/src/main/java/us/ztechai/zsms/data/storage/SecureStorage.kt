package us.ztechai.zsms.data.storage

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey

/**
 * Hardware-backed encrypted key-value store using Android Keystore and AES-256-GCM.
 */
class SecureStorage(context: Context) {

    private val masterKey: MasterKey = MasterKey.Builder(context)
        .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
        .build()

    private val prefs: SharedPreferences = EncryptedSharedPreferences.create(
        context,
        PREFS_FILENAME,
        masterKey,
        EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
        EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
    )

    fun saveServerUrl(url: String) {
        prefs.edit().putString(KEY_SERVER_URL, url).apply()
    }

    fun getServerUrl(): String {
        return prefs.getString(KEY_SERVER_URL, DEFAULT_SERVER_URL) ?: DEFAULT_SERVER_URL
    }

    fun saveDeviceApiKey(apiKey: String) {
        prefs.edit().putString(KEY_DEVICE_API_KEY, apiKey).apply()
    }

    fun getDeviceApiKey(): String? {
        return prefs.getString(KEY_DEVICE_API_KEY, null)
    }

    fun savePhoneId(phoneId: String) {
        prefs.edit().putString(KEY_PHONE_ID, phoneId).apply()
    }

    fun getPhoneId(): String? {
        return prefs.getString(KEY_PHONE_ID, null)
    }

    fun saveDeviceName(name: String) {
        prefs.edit().putString(KEY_DEVICE_NAME, name).apply()
    }

    fun getDeviceName(): String {
        return prefs.getString(KEY_DEVICE_NAME, "Android Gateway") ?: "Android Gateway"
    }

    fun savePhoneNumber(phone: String) {
        prefs.edit().putString(KEY_PHONE_NUMBER, phone).apply()
    }

    fun getPhoneNumber(): String? {
        return prefs.getString(KEY_PHONE_NUMBER, null)
    }

    fun clearAll() {
        prefs.edit().clear().apply()
    }

    companion object {
        private const val PREFS_FILENAME = "zsms_secure_prefs"
        private const val KEY_SERVER_URL = "server_url"
        private const val KEY_DEVICE_API_KEY = "device_api_key"
        private const val KEY_PHONE_ID = "phone_id"
        private const val KEY_DEVICE_NAME = "device_name"
        private const val KEY_PHONE_NUMBER = "phone_number"
        private const val DEFAULT_SERVER_URL = "https://sms-api.ztechai.us" // Default to Production/Staging API
    }
}
