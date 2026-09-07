<template>
  <div class="dashboard-page">
    <!-- Header Title & Action Bar -->
    <div class="d-flex flex-wrap align-center justify-space-between mb-6 ga-3">
      <div>
        <h1 class="text-h4 font-weight-bold">Gateway Dashboard</h1>
        <p class="text-body-2 text-medium-emphasis">
          ZTechAI ZSMS Cellular Gateway Control Center
        </p>
      </div>

      <div class="d-flex ga-2">
        <v-btn
          color="primary"
          variant="flat"
          prepend-icon="mdi-cellphone-link"
          to="/phones"
        >
          Pair Phone
        </v-btn>
        <v-btn
          color="primary"
          variant="outlined"
          prepend-icon="mdi-message-plus-outline"
          to="/messages"
        >
          Send SMS
        </v-btn>
        <v-btn
          icon="mdi-refresh"
          variant="text"
          :loading="loading"
          @click="fetchDashboardData"
        />
      </div>
    </div>

    <!-- Metric Stat Cards -->
    <v-row class="mb-6">
      <v-col cols="12" sm="6" md="3">
        <v-card elevation="1" class="pa-4 rounded-lg">
          <div class="d-flex align-center justify-space-between mb-2">
            <span class="text-caption text-uppercase font-weight-bold text-medium-emphasis">Connected Gateways</span>
            <v-avatar color="primary" variant="tonal" size="36">
              <v-icon size="20">mdi-cellphone-wireless</v-icon>
            </v-avatar>
          </div>
          <div class="text-h4 font-weight-bold">
            {{ stats.online_phones }} / {{ stats.total_phones }}
          </div>
          <div class="text-caption text-success mt-1 d-flex align-center">
            <v-icon size="14" class="mr-1">mdi-circle</v-icon>
            <span>{{ stats.online_phones }} gateway(s) online</span>
          </div>
        </v-card>
      </v-col>

      <v-col cols="12" sm="6" md="3">
        <v-card elevation="1" class="pa-4 rounded-lg">
          <div class="d-flex align-center justify-space-between mb-2">
            <span class="text-caption text-uppercase font-weight-bold text-medium-emphasis">Messages Today</span>
            <v-avatar color="info" variant="tonal" size="36">
              <v-icon size="20">mdi-message-text-clock</v-icon>
            </v-avatar>
          </div>
          <div class="text-h4 font-weight-bold">
            {{ stats.messages_today }}
          </div>
          <div class="text-caption text-medium-emphasis mt-1">
            Total lifetime: {{ stats.total_messages }}
          </div>
        </v-card>
      </v-col>

      <v-col cols="12" sm="6" md="3">
        <v-card elevation="1" class="pa-4 rounded-lg">
          <div class="d-flex align-center justify-space-between mb-2">
            <span class="text-caption text-uppercase font-weight-bold text-medium-emphasis">Delivered / Sent</span>
            <v-avatar color="success" variant="tonal" size="36">
              <v-icon size="20">mdi-check-all</v-icon>
            </v-avatar>
          </div>
          <div class="text-h4 font-weight-bold">
            {{ stats.delivered_count }}
          </div>
          <div class="text-caption text-medium-emphasis mt-1">
            <span v-if="stats.failed_count > 0" class="text-error font-weight-bold">
              {{ stats.failed_count }} failed
            </span>
            <span v-else class="text-success font-weight-medium">
              100% cellular success
            </span>
          </div>
        </v-card>
      </v-col>

      <v-col cols="12" sm="6" md="3">
        <v-card elevation="1" class="pa-4 rounded-lg">
          <div class="d-flex align-center justify-space-between mb-2">
            <span class="text-caption text-uppercase font-weight-bold text-medium-emphasis">Received (Inbound)</span>
            <v-avatar color="warning" variant="tonal" size="36">
              <v-icon size="20">mdi-message-arrow-left-outline</v-icon>
            </v-avatar>
          </div>
          <div class="text-h4 font-weight-bold">
            {{ stats.received_count }}
          </div>
          <div class="text-caption text-medium-emphasis mt-1">
            Two-way SIM replies
          </div>
        </v-card>
      </v-col>
    </v-row>

    <!-- Paired Gateways & Recent Messages -->
    <v-row>
      <!-- Paired Hardware Phones -->
      <v-col cols="12" lg="5">
        <v-card elevation="1" rounded="lg" class="pa-4 h-100">
          <div class="d-flex align-center justify-space-between mb-4">
            <h2 class="text-h6 font-weight-bold">Cellular Gateways</h2>
            <v-btn to="/phones" size="small" variant="text" color="primary">Manage All</v-btn>
          </div>

          <div v-if="loading && phones.length === 0" class="text-center py-6">
            <v-progress-circular indeterminate color="primary" />
          </div>

          <div v-else-if="phones.length === 0" class="text-center py-8">
            <v-icon size="48" color="grey-lighten-1" class="mb-2">mdi-cellphone-link-off</v-icon>
            <p class="text-subtitle-2 text-medium-emphasis">No Android Gateways Paired</p>
            <p class="text-caption text-disabled mb-4">
              Install the ZSMS APK on your physical Android phone and pair it using a 6-digit code.
            </p>
            <v-btn to="/phones" color="primary" size="small" variant="flat">
              Pair Android Device
            </v-btn>
          </div>

          <v-list v-else lines="two" density="comfortable" class="pa-0">
            <v-list-item
              v-for="phone in phones"
              :key="phone.id"
              class="px-0 py-2 border-b"
            >
              <template #prepend>
                <v-avatar :color="phone.status === 'online' ? 'success' : 'grey'" variant="tonal" size="40">
                  <v-icon>{{ phone.status === 'online' ? 'mdi-cellphone' : 'mdi-cellphone-off' }}</v-icon>
                </v-avatar>
              </template>

              <v-list-item-title class="font-weight-bold">
                {{ phone.name }}
              </v-list-item-title>
              <v-list-item-subtitle class="text-caption">
                {{ phone.sim_carrier || 'Physical SIM' }} • {{ phone.phone_number || 'SIM Active' }}
              </v-list-item-subtitle>

              <template #append>
                <div class="text-end">
                  <v-chip
                    size="x-small"
                    :color="phone.status === 'online' ? 'success' : 'grey'"
                    variant="flat"
                    class="font-weight-bold"
                  >
                    {{ (phone.status || 'offline').toUpperCase() }}
                  </v-chip>
                  <div class="text-caption text-medium-emphasis mt-1">
                    {{ phone.battery_level ? `${phone.battery_level}% battery` : '' }}
                  </div>
                </div>
              </template>
            </v-list-item>
          </v-list>
        </v-card>
      </v-col>

      <!-- Recent Cellular SMS Messages -->
      <v-col cols="12" lg="7">
        <v-card elevation="1" rounded="lg" class="pa-4 h-100">
          <div class="d-flex align-center justify-space-between mb-4">
            <h2 class="text-h6 font-weight-bold">Recent Cellular Transmissions</h2>
            <v-btn to="/messages" size="small" variant="text" color="primary">View All Logs</v-btn>
          </div>

          <div v-if="loading && recentMessages.length === 0" class="text-center py-6">
            <v-progress-circular indeterminate color="primary" />
          </div>

          <div v-else-if="recentMessages.length === 0" class="text-center py-8">
            <v-icon size="48" color="grey-lighten-1" class="mb-2">mdi-message-outline</v-icon>
            <p class="text-subtitle-2 text-medium-emphasis">No SMS messages sent or received yet</p>
            <p class="text-caption text-disabled mb-4">
              Dispatch your first real SMS through your connected Android SIM.
            </p>
            <v-btn to="/messages" color="primary" size="small" variant="flat">
              Send First SMS
            </v-btn>
          </div>

          <v-table v-else density="compact">
            <thead>
              <tr>
                <th class="text-left">Type</th>
                <th class="text-left">Recipient / Sender</th>
                <th class="text-left">Message Snippet</th>
                <th class="text-left">Status</th>
                <th class="text-right">Time</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="msg in recentMessages" :key="msg.id">
                <td>
                  <v-icon
                    size="18"
                    :color="msg.direction === 'inbound' ? 'info' : 'primary'"
                  >
                    {{ msg.direction === 'inbound' ? 'mdi-arrow-bottom-left' : 'mdi-arrow-top-right' }}
                  </v-icon>
                </td>
                <td class="font-weight-medium">
                  {{ msg.direction === 'inbound' ? msg.sender_phone : msg.recipient_phone }}
                </td>
                <td class="text-truncate" style="max-width: 200px;">
                  {{ msg.body }}
                </td>
                <td>
                  <v-chip
                    size="x-small"
                    :color="getStatusColor(msg.status)"
                    variant="tonal"
                    class="font-weight-bold"
                  >
                    {{ msg.status }}
                  </v-chip>
                </td>
                <td class="text-right text-caption text-medium-emphasis">
                  {{ formatDate(msg.created_at) }}
                </td>
              </tr>
            </tbody>
          </v-table>
        </v-card>
      </v-col>
    </v-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useApi } from '~/composables/useApi'

const api = useApi()
const loading = ref(false)

const stats = ref({
  total_phones: 0,
  online_phones: 0,
  total_messages: 0,
  messages_today: 0,
  sent_count: 0,
  delivered_count: 0,
  failed_count: 0,
  received_count: 0,
})

const phones = ref<any[]>([])
const recentMessages = ref<any[]>([])

async function fetchDashboardData() {
  loading.value = true
  try {
    const [statsRes, phonesRes, msgsRes] = await Promise.all([
      api.get<any>('/api/v1/dashboard/stats'),
      api.get<any>('/api/v1/phones'),
      api.get<any>('/api/v1/messages?limit=5'),
    ])

    if (statsRes.success && statsRes.data) {
      stats.value = statsRes.data
    }
    if (phonesRes.success && phonesRes.data) {
      phones.value = phonesRes.data
    }
    if (msgsRes.success && msgsRes.data) {
      recentMessages.value = msgsRes.data
    }
  } catch (err) {
    console.error('Failed to load dashboard statistics', err)
  } finally {
    loading.value = false
  }
}

function getStatusColor(status: string): string {
  switch (status) {
    case 'delivered': return 'success'
    case 'sent': return 'info'
    case 'received': return 'purple'
    case 'failed': return 'error'
    case 'processing': return 'warning'
    default: return 'grey'
  }
}

function formatDate(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

onMounted(() => {
  fetchDashboardData()
})
</script>
