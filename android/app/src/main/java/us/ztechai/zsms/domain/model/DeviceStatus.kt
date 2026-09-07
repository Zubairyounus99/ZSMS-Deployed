package us.ztechai.zsms.domain.model

/**
 * Domain model representing gateway device telemetry and operational status.
 */
data class DeviceStatus(
    val phoneId: String?,
    val isOnline: Boolean,
    val batteryLevel: Int,
    val isCharging: Boolean,
    val networkType: String,
    val signalStrengthDbm: Int?,
    val simSlotCount: Int,
    val defaultSimSlot: Int
)
