<template>
  <div class="phones-page">
    <!-- Header Title & Pair Button -->
    <div class="d-flex flex-wrap align-center justify-space-between mb-6 ga-3">
      <div>
        <h1 class="text-h4 font-weight-bold">Phone Gateways</h1>
        <p class="text-body-2 text-medium-emphasis">
          Manage physical Android smartphones connected to the ZSMS gateway network
        </p>
      </div>

      <div class="d-flex ga-2">
        <v-btn
          color="primary"
          prepend-icon="mdi-cellphone-link"
          size="large"
          class="font-weight-bold"
          @click="openPairingDialog"
        >
          Pair New Phone
        </v-btn>
        <v-btn
          icon="mdi-refresh"
          variant="outlined"
          :loading="loading"
          @click="fetchPhones"
        />
      </div>
    </div>

    <!-- Phone List or Empty State -->
    <div v-if="loading && phones.length === 0" class="text-center py-12">
      <v-progress-circular indeterminate color="primary" size="48" />
      <p class="text-subtitle-2 text-medium-emphasis mt-4">Loading paired gateways...</p>
    </div>

    <v-card v-else-if="phones.length === 0" elevation="1" class="text-center py-16 px-4 rounded-xl">
      <v-avatar color="primary" variant="tonal" size="80" class="mb-4">
        <v-icon size="40">mdi-cellphone-wireless</v-icon>
      </v-avatar>
      <h2 class="text-h5 font-weight-bold mb-2">No Cellular Gateways Connected</h2>
      <p class="text-body-1 text-medium-emphasis mb-6" style="max-width: 540px; margin: 0 auto;">
        Connect your physical Android smartphone to turn it into an automated cellular SMS gateway. Install the ZSMS APK and enter a 6-digit pairing code.
      </p>
      <v-btn
        color="primary"
        size="large"
        prepend-icon="mdi-plus"
        class="font-weight-bold"
        @click="openPairingDialog"
      >
        Pair First Android Phone
      </v-btn>
    </v-card>

    <v-row v-else>
      <v-col
        v-for="phone in phones"
        :key="phone.id"
        cols="12"
        md="6"
        lg="4"
      >
        <v-card elevation="1" class="pa-5 rounded-xl h-100 d-flex flex-column justify-space-between">
          <div>
            <!-- Top row: Name & Status -->
            <div class="d-flex align-center justify-space-between mb-3">
              <div class="d-flex align-center">
                <v-avatar :color="phone.status === 'online' ? 'success' : 'grey'" variant="tonal" size="42" class="mr-3">
                  <v-icon>{{ phone.status === 'online' ? 'mdi-cellphone' : 'mdi-cellphone-off' }}</v-icon>
                </v-avatar>
                <div>
                  <h3 class="text-subtitle-1 font-weight-bold leading-tight">{{ phone.name }}</h3>
                  <span class="text-caption text-medium-emphasis">ID: {{ phone.id.substring(0, 8) }}...</span>
                </div>
              </div>

              <v-chip
                size="small"
                :color="phone.status === 'online' ? 'success' : 'grey'"
                variant="flat"
                class="font-weight-bold"
              >
                {{ (phone.status || 'offline').toUpperCase() }}
              </v-chip>
            </div>

            <v-divider class="my-3" />

            <!-- Telemetry Details -->
            <div class="text-body-2 ga-2 d-flex flex-column">
              <div class="d-flex justify-space-between py-1">
                <span class="text-medium-emphasis">SIM Carrier:</span>
                <span class="font-weight-medium">{{ phone.sim_carrier || 'Physical SIM' }}</span>
              </div>
              <div class="d-flex justify-space-between py-1">
                <span class="text-medium-emphasis">Phone Number:</span>
                <span class="font-weight-medium">{{ phone.phone_number || 'Detected' }}</span>
              </div>
              <div class="d-flex justify-space-between py-1">
                <span class="text-medium-emphasis">Battery Level:</span>
                <span class="font-weight-medium">
                  {{ phone.battery_level ? `${phone.battery_level}%` : 'Unknown' }}
                </span>
              </div>
              <div class="d-flex justify-space-between py-1">
                <span class="text-medium-emphasis">Network:</span>
                <span class="font-weight-medium">{{ phone.network_type || 'Wi-Fi LAN' }}</span>
              </div>
              <div class="d-flex justify-space-between py-1">
                <span class="text-medium-emphasis">Last Heartbeat:</span>
                <span class="font-weight-medium text-caption">{{ formatTime(phone.last_seen_at) }}</span>
              </div>
            </div>
          </div>

          <div class="mt-4 pt-3 border-t d-flex justify-end">
            <v-btn
              to="/messages"
              variant="tonal"
              color="primary"
              size="small"
              prepend-icon="mdi-message-text-outline"
            >
              Send SMS via This SIM
            </v-btn>
          </div>
        </v-card>
      </v-col>
    </v-row>

    <!-- Pairing Modal Dialog -->
    <v-dialog v-model="pairingDialog" max-width="500" persistent>
      <v-card class="pa-6 rounded-xl">
        <div class="d-flex align-center justify-space-between mb-4">
          <div class="d-flex align-center">
            <v-icon color="primary" class="mr-2">mdi-cellphone-link</v-icon>
            <h2 class="text-h6 font-weight-bold">Pair Android Smartphone</h2>
          </div>
          <v-btn icon="mdi-close" variant="text" size="small" @click="closePairingDialog" />
        </div>

        <div v-if="pairingLoading" class="text-center py-10">
          <v-progress-circular indeterminate color="primary" size="48" />
          <p class="text-body-2 text-medium-emphasis mt-3">Generating secure pairing PIN...</p>
        </div>

        <div v-else-if="pairingError" class="text-center py-6">
          <v-alert type="error" variant="tonal" class="mb-4">{{ pairingError }}</v-alert>
          <v-btn color="primary" @click="generatePairingCode">Retry</v-btn>
        </div>

        <div v-else class="text-center">
          <p class="text-body-2 text-medium-emphasis mb-4">
            Open the ZSMS App on your Android device, tap <strong>"Pair Gateway Device"</strong>, and enter this single-use 6-digit code:
          </p>

          <!-- 6-digit PIN Box -->
          <v-card color="primary" variant="tonal" class="pa-6 rounded-xl my-4">
            <div class="text-h3 font-weight-black text-primary letter-spacing-widest">
              {{ formatPairingCode(pairingCode) }}
            </div>
            <div class="text-caption text-medium-emphasis mt-2">
              Valid for: <span class="font-weight-bold">{{ formatCountdown(timeRemaining) }}</span>
            </div>
          </v-card>

          <v-alert type="info" variant="tonal" density="compact" class="text-left text-caption mb-4">
            <strong>Gateway Connection:</strong> Ensure your phone has internet access (Wi-Fi or cellular) and the Server URL in app settings is set to <code>https://sms-api.ztechai.us</code>.
          </v-alert>

          <v-btn
            color="primary"
            block
            size="large"
            rounded="lg"
            class="font-weight-bold"
            @click="closePairingDialog"
          >
            I Have Entered The Code
          </v-btn>
        </div>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useApi } from '~/composables/useApi'

const api = useApi()
const loading = ref(false)
const phones = ref<any[]>([])

const pairingDialog = ref(false)
const pairingLoading = ref(false)
const pairingError = ref('')
const pairingCode = ref('')
const timeRemaining = ref(600)
let timerInterval: any = null

async function fetchPhones() {
  loading.value = true
  try {
    const res = await api.get<any[]>('/api/v1/phones')
    if (res.success && res.data) {
      phones.value = res.data
    }
  } catch (err) {
    console.error('Failed to load phones', err)
  } finally {
    loading.value = false
  }
}

async function openPairingDialog() {
  pairingDialog.value = true
  await generatePairingCode()
}

function closePairingDialog() {
  pairingDialog.value = false
  clearInterval(timerInterval)
  fetchPhones()
}

async function generatePairingCode() {
  pairingLoading.value = true
  pairingError.value = ''
  clearInterval(timerInterval)

  try {
    const res = await api.post<any>('/api/v1/pairing/sessions', {})
    if (res.success && res.data) {
      pairingCode.value = res.data.pairing_code
      timeRemaining.value = res.data.expires_in_s || 600
      startTimer()
    } else {
      pairingError.value = res.error?.message || 'Failed to generate pairing session.'
    }
  } catch (err: any) {
    pairingError.value = err.message || 'Network error generating pairing code.'
  } finally {
    pairingLoading.value = false
  }
}

function startTimer() {
  clearInterval(timerInterval)
  timerInterval = setInterval(() => {
    if (timeRemaining.value > 0) {
      timeRemaining.value--
    } else {
      clearInterval(timerInterval)
      pairingError.value = 'Pairing code expired. Please generate a new code.'
    }
  }, 1000)
}

function formatPairingCode(code: string): string {
  if (!code || code.length !== 6) return code
  return `${code.substring(0, 3)} ${code.substring(3, 6)}`
}

function formatCountdown(sec: number): string {
  const m = Math.floor(sec / 60)
  const s = sec % 60
  return `${m}:${s.toString().padStart(2, '0')}`
}

function formatTime(dateStr: string): string {
  if (!dateStr) return 'Never'
  const d = new Date(dateStr)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

onMounted(() => {
  fetchPhones()
})

onUnmounted(() => {
  clearInterval(timerInterval)
})
</script>

<style scoped>
.letter-spacing-widest {
  letter-spacing: 0.35em !important;
}
</style>
