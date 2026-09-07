<template>
  <div class="messages-page">
    <!-- Header & Send Action -->
    <div class="d-flex flex-wrap align-center justify-space-between mb-6 ga-3">
      <div>
        <h1 class="text-h4 font-weight-bold">Cellular Messages</h1>
        <p class="text-body-2 text-medium-emphasis">
          Real-time cellular SMS dispatch log and SIM carrier delivery audit trail
        </p>
      </div>

      <div class="d-flex ga-2">
        <v-btn
          color="primary"
          prepend-icon="mdi-send"
          size="large"
          class="font-weight-bold"
          @click="openSendDialog"
        >
          Send Real SMS
        </v-btn>
        <v-btn
          icon="mdi-refresh"
          variant="outlined"
          :loading="loading"
          @click="fetchMessages"
        />
      </div>
    </div>

    <!-- Filter Toolbar -->
    <v-card elevation="1" class="pa-4 mb-6 rounded-lg">
      <v-row dense align="center">
        <v-col cols="12" sm="4" md="3">
          <v-select
            v-model="filters.direction"
            :items="[
              { title: 'All Directions', value: '' },
              { title: 'Outbound (Sent)', value: 'outbound' },
              { title: 'Inbound (Received)', value: 'inbound' },
            ]"
            label="Direction"
            density="compact"
            variant="outlined"
            hide-details
            @update:model-value="fetchMessages"
          />
        </v-col>

        <v-col cols="12" sm="4" md="3">
          <v-select
            v-model="filters.status"
            :items="[
              { title: 'All Statuses', value: '' },
              { title: 'Delivered', value: 'delivered' },
              { title: 'Sent', value: 'sent' },
              { title: 'Received', value: 'received' },
              { title: 'Failed', value: 'failed' },
              { title: 'Queued / Processing', value: 'processing' },
            ]"
            label="Status"
            density="compact"
            variant="outlined"
            hide-details
            @update:model-value="fetchMessages"
          />
        </v-col>

        <v-spacer />

        <v-col cols="auto">
          <span class="text-caption text-medium-emphasis">
            Total Messages: <strong>{{ totalMessages }}</strong>
          </span>
        </v-col>
      </v-row>
    </v-card>

    <!-- Message Table -->
    <v-card elevation="1" rounded-lg>
      <div v-if="loading && messages.length === 0" class="text-center py-12">
        <v-progress-circular indeterminate color="primary" size="48" />
        <p class="text-subtitle-2 text-medium-emphasis mt-4">Loading messages...</p>
      </div>

      <div v-else-if="messages.length === 0" class="text-center py-16 px-4">
        <v-avatar color="primary" variant="tonal" size="64" class="mb-4">
          <v-icon size="32">mdi-message-text-outline</v-icon>
        </v-avatar>
        <h2 class="text-h6 font-weight-bold mb-2">No Messages Found</h2>
        <p class="text-body-2 text-medium-emphasis mb-6">
          Transmit your first cellular SMS message through a paired Android phone gateway.
        </p>
        <v-btn color="primary" prepend-icon="mdi-send" @click="openSendDialog">
          Send SMS Now
        </v-btn>
      </div>

      <v-table v-else hover density="comfortable">
        <thead>
          <tr>
            <th class="text-left">Direction</th>
            <th class="text-left">Phone Gateway</th>
            <th class="text-left">Contact / Number</th>
            <th class="text-left">Message Content</th>
            <th class="text-left">Parts</th>
            <th class="text-left">Status</th>
            <th class="text-right">Timestamp</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="msg in messages"
            :key="msg.id"
            class="cursor-pointer"
            @click="viewMessageDetail(msg)"
          >
            <td>
              <v-chip
                size="x-small"
                :color="msg.direction === 'inbound' ? 'info' : 'primary'"
                variant="tonal"
                class="font-weight-bold"
              >
                <v-icon start size="12">
                  {{ msg.direction === 'inbound' ? 'mdi-arrow-bottom-left' : 'mdi-arrow-top-right' }}
                </v-icon>
                {{ msg.direction.toUpperCase() }}
              </v-chip>
            </td>
            <td class="font-weight-medium">
              {{ getPhoneName(msg.phone_id) }}
            </td>
            <td class="font-weight-bold">
              {{ msg.direction === 'inbound' ? msg.sender_phone : msg.recipient_phone }}
            </td>
            <td style="max-width: 320px;" class="text-truncate">
              {{ msg.body }}
            </td>
            <td class="text-caption">
              {{ msg.parts_count || 1 }} seg
            </td>
            <td>
              <v-chip
                size="x-small"
                :color="getStatusColor(msg.status)"
                variant="flat"
                class="font-weight-bold"
              >
                {{ msg.status.toUpperCase() }}
              </v-chip>
            </td>
            <td class="text-right text-caption text-medium-emphasis font-mono">
              {{ formatDateTime(msg.created_at) }}
            </td>
          </tr>
        </tbody>
      </v-table>
    </v-card>

    <!-- Send SMS Modal Dialog -->
    <v-dialog v-model="sendDialog" max-width="560" persistent>
      <v-card class="pa-6 rounded-xl">
        <div class="d-flex align-center justify-space-between mb-4">
          <div class="d-flex align-center">
            <v-icon color="primary" class="mr-2">mdi-send</v-icon>
            <h2 class="text-h6 font-weight-bold">Send Cellular SMS</h2>
          </div>
          <v-btn icon="mdi-close" variant="text" size="small" @click="sendDialog = false" />
        </div>

        <v-alert
          v-if="sendError"
          type="error"
          variant="tonal"
          density="compact"
          closable
          class="mb-4"
          @click:close="sendError = ''"
        >
          {{ sendError }}
        </v-alert>

        <v-form @submit.prevent="executeSendMessage">
          <!-- Gateway Device Selector -->
          <v-select
            v-model="sendForm.phone_id"
            :items="phoneOptions"
            item-title="title"
            item-value="value"
            label="Select Android Phone Gateway"
            variant="outlined"
            density="comfortable"
            class="mb-3"
            :rules="[v => !!v || 'Gateway phone selection is required']"
            required
          />

          <!-- Recipient Phone Number -->
          <v-text-field
            v-model="sendForm.recipient_phone"
            label="Recipient Phone Number (E.164 e.g. +1234567890)"
            placeholder="+1234567890"
            variant="outlined"
            density="comfortable"
            prepend-inner-icon="mdi-phone"
            class="mb-3"
            :rules="[v => !!v || 'Recipient phone number is required']"
            required
          />

          <!-- SIM Slot Selection -->
          <v-row dense class="mb-2">
            <v-col cols="6">
              <v-select
                v-model="sendForm.sim_slot"
                :items="[
                  { title: 'SIM 1 (Default)', value: 1 },
                  { title: 'SIM 2 (Secondary)', value: 2 },
                ]"
                label="SIM Slot"
                variant="outlined"
                density="comfortable"
              />
            </v-col>
            <v-col cols="6">
              <v-select
                v-model="sendForm.priority"
                :items="[
                  { title: 'Normal', value: 'normal' },
                  { title: 'High (Immediate)', value: 'high' },
                ]"
                label="Dispatch Priority"
                variant="outlined"
                density="comfortable"
              />
            </v-col>
          </v-row>

          <!-- Message Textarea -->
          <v-textarea
            v-model="sendForm.body"
            label="Message Content"
            placeholder="Type your SMS message here..."
            variant="outlined"
            rows="4"
            counter
            class="mb-2"
            :rules="[v => !!v || 'Message content cannot be empty']"
            required
          />

          <div class="d-flex justify-space-between text-caption text-medium-emphasis mb-4">
            <span>GSM-7 Segments: {{ calculateSegments(sendForm.body) }}</span>
            <span>Characters: {{ sendForm.body.length }}</span>
          </div>

          <v-btn
            type="submit"
            color="primary"
            block
            size="large"
            rounded="lg"
            :loading="sending"
            class="font-weight-bold"
          >
            Dispatch Real Cellular SMS
          </v-btn>
        </v-form>
      </v-card>
    </v-dialog>

    <!-- Message Detail Modal -->
    <v-dialog v-model="detailDialog" max-width="600">
      <v-card v-if="selectedMessage" class="pa-6 rounded-xl">
        <div class="d-flex align-center justify-space-between mb-4">
          <h2 class="text-h6 font-weight-bold">Message Details</h2>
          <v-btn icon="mdi-close" variant="text" size="small" @click="detailDialog = false" />
        </div>

        <div class="mb-4">
          <v-chip
            :color="getStatusColor(selectedMessage.status)"
            size="small"
            variant="flat"
            class="font-weight-bold mr-2"
          >
            {{ selectedMessage.status.toUpperCase() }}
          </v-chip>
          <v-chip
            :color="selectedMessage.direction === 'inbound' ? 'info' : 'primary'"
            size="small"
            variant="tonal"
            class="font-weight-bold"
          >
            {{ selectedMessage.direction.toUpperCase() }}
          </v-chip>
        </div>

        <v-card variant="tonal" class="pa-3 mb-4 rounded-lg">
          <p class="text-body-1 text-high-emphasis whitespace-pre-wrap">{{ selectedMessage.body }}</p>
        </v-card>

        <v-table density="compact" class="mb-4">
          <tbody>
            <tr>
              <td class="font-weight-bold text-medium-emphasis">Message ID:</td>
              <td class="font-mono text-caption">{{ selectedMessage.id }}</td>
            </tr>
            <tr>
              <td class="font-weight-bold text-medium-emphasis">Recipient:</td>
              <td>{{ selectedMessage.recipient_phone }}</td>
            </tr>
            <tr>
              <td class="font-weight-bold text-medium-emphasis">Sender:</td>
              <td>{{ selectedMessage.sender_phone || 'Cellular Gateway' }}</td>
            </tr>
            <tr>
              <td class="font-weight-bold text-medium-emphasis">SIM Slot:</td>
              <td>SIM {{ selectedMessage.sim_slot || 1 }}</td>
            </tr>
            <tr>
              <td class="font-weight-bold text-medium-emphasis">Dispatched At:</td>
              <td>{{ selectedMessage.dispatched_at ? formatDateTime(selectedMessage.dispatched_at) : 'Pending' }}</td>
            </tr>
            <tr>
              <td class="font-weight-bold text-medium-emphasis">Delivered At:</td>
              <td>{{ selectedMessage.delivered_at ? formatDateTime(selectedMessage.delivered_at) : '--' }}</td>
            </tr>
            <tr v-if="selectedMessage.error_code">
              <td class="font-weight-bold text-error">Error:</td>
              <td class="text-error font-weight-medium">
                {{ selectedMessage.error_code }}: {{ selectedMessage.error_message }}
              </td>
            </tr>
          </tbody>
        </v-table>

        <div class="d-flex justify-end">
          <v-btn color="primary" variant="flat" @click="detailDialog = false">Close</v-btn>
        </div>
      </v-card>
    </v-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useApi } from '~/composables/useApi'

const api = useApi()
const loading = ref(false)
const sending = ref(false)
const messages = ref<any[]>([])
const phones = ref<any[]>([])
const totalMessages = ref(0)

const filters = ref({
  direction: '',
  status: '',
})

const sendDialog = ref(false)
const sendError = ref('')
const sendForm = ref({
  phone_id: '',
  recipient_phone: '',
  sim_slot: 1,
  priority: 'normal',
  body: '',
})

const detailDialog = ref(false)
const selectedMessage = ref<any>(null)

const phoneOptions = computed(() => {
  return phones.value.map(p => ({
    title: `${p.name} (${p.sim_carrier || 'Physical SIM'}) - ${p.status.toUpperCase()}`,
    value: p.id,
  }))
})

async function fetchPhones() {
  try {
    const res = await api.get<any[]>('/api/v1/phones')
    if (res.success && res.data) {
      phones.value = res.data
      if (phones.value.length > 0 && !sendForm.value.phone_id) {
        sendForm.value.phone_id = phones.value[0].id
      }
    }
  } catch (err) {
    console.error('Failed to load phones', err)
  }
}

async function fetchMessages() {
  loading.value = true
  try {
    let url = '/api/v1/messages?'
    if (filters.value.direction) url += `direction=${filters.value.direction}&`
    if (filters.value.status) url += `status=${filters.value.status}&`

    const res = await api.get<any[]>(url)
    if (res.success && res.data) {
      messages.value = res.data
      totalMessages.value = res.meta?.total || res.data.length
    }
  } catch (err) {
    console.error('Failed to load messages', err)
  } finally {
    loading.value = false
  }
}

function openSendDialog() {
  sendError.value = ''
  if (phones.value.length === 0) {
    fetchPhones()
  }
  sendDialog.value = true
}

async function executeSendMessage() {
  sendError.value = ''
  sending.value = true

  try {
    const res = await api.post<any>('/api/v1/messages', {
      phone_id: sendForm.value.phone_id,
      recipient_phone: sendForm.value.recipient_phone.trim(),
      body: sendForm.value.body.trim(),
      sim_slot: sendForm.value.sim_slot,
      priority: sendForm.value.priority,
    })

    if (res.success && res.data) {
      sendDialog.value = false
      sendForm.value.body = ''
      await fetchMessages()
    } else {
      sendError.value = res.error?.message || 'Failed to dispatch SMS'
    }
  } catch (err: any) {
    sendError.value = err.message || 'Network error dispatching message'
  } finally {
    sending.value = false
  }
}

function viewMessageDetail(msg: any) {
  selectedMessage.value = msg
  detailDialog.value = true
}

function getPhoneName(phoneId: string): string {
  const p = phones.value.find(item => item.id === phoneId)
  return p ? p.name : (phoneId ? phoneId.substring(0, 8) + '...' : 'Unknown')
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

function calculateSegments(body: string): number {
  if (!body) return 0
  const len = body.length
  if (len <= 160) return 1
  return Math.ceil(len / 153)
}

function formatDateTime(dateStr: string): string {
  if (!dateStr) return '--'
  const d = new Date(dateStr)
  return d.toLocaleString([], {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

onMounted(async () => {
  await Promise.all([fetchPhones(), fetchMessages()])
})
</script>

<style scoped>
.whitespace-pre-wrap {
  white-space: pre-wrap;
}
.font-mono {
  font-family: monospace;
}
</style>
