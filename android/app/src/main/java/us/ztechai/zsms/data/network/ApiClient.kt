package us.ztechai.zsms.data.network

import okhttp3.Interceptor
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import us.ztechai.zsms.data.storage.SecureStorage
import java.util.UUID
import java.util.concurrent.TimeUnit

/**
 * Foundation HTTP networking client for Android Gateway.
 */
class ApiClient(private val storage: SecureStorage) {

    val client: OkHttpClient = OkHttpClient.Builder()
        .connectTimeout(15, TimeUnit.SECONDS)
        .readTimeout(15, TimeUnit.SECONDS)
        .writeTimeout(15, TimeUnit.SECONDS)
        .addInterceptor(AuthAndTelemetryInterceptor())
        .build()

    private inner class AuthAndTelemetryInterceptor : Interceptor {
        override fun intercept(chain: Interceptor.Chain): Response {
            val original: Request = chain.request()
            val builder: Request.Builder = original.newBuilder()
                .header("Accept", "application/json")
                .header("Content-Type", "application/json")
                .header("X-Request-ID", UUID.randomUUID().toString())
                .header("User-Agent", "ZSMS-Android-Gateway/1.0.0")

            val apiKey = storage.getDeviceApiKey()
            if (!apiKey.isNullOrBlank()) {
                builder.header("Authorization", "Bearer $apiKey")
            }

            return chain.proceed(builder.build())
        }
    }
}
