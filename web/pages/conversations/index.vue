<template>
  <div class="conversations-page h-100">
    <div class="d-flex align-center justify-space-between mb-4">
      <div>
        <h1 class="text-h4 font-weight-bold">Two-Way Conversations</h1>
        <p class="text-body-2 text-medium-emphasis">
          Interactive chat threads with real-time cellular SMS dispatch and inbound replies
        </p>
      </div>

      <v-btn
        icon="mdi-refresh"
        variant="outlined"
        :loading="loadingThreads"
        @click="fetchThreads"
      />
    </div>

    <!-- Main Split-Pane Container -->
    <v-card elevation="1" rounded-xl class="overflow-hidden chat-container">
      <v-row no-gutters class="h-100">
        <!-- Left Pane: Thread List -->
        <v-col cols="12" md="4" class="border-e d-flex flex-column thread-list-pane">
          <div class="pa-3 border-b bg-surface">
            <v-text-field
              v-model="searchQuery"
              placeholder="Search conversations..."
              prepend-inner-icon="mdi-magnify"
              density="compact"
              variant="outlined"
              hide-details
              clearable
            />
          </div>

          <div v-if="loadingThreads && threads.length === 0" class="text-center py-10">
            <v-progress-circular indeterminate color="primary" />
          </div>

          <div v-else-if="filteredThreads.length === 0" class="text-center py-12 px-4">
            <v-icon size="40" color="grey-lighten-1" class="mb-2">mdi-forum-outline</v-icon>
            <p class="text-subtitle-2 text-medium-emphasis">No conversation threads</p>
            <p class="text-caption text-disabled">
              Threads are created automatically when SMS messages are sent or received.
            </p>
          </div>

          <v-list v-else lines="two" density="comfortable" class="overflow-y-auto flex-grow-1 pa-0">
            <v-list-item
              v-for="thread in filteredThreads"
              :key="thread.id"
              :active="selectedThread?.id === thread.id"
              color="primary"
              class="border-b"
              @click="selectThread(thread)"
            >
              <template #prepend>
                <v-avatar color="primary" variant="tonal" size="42" class="mr-3">
                  <v-icon>mdi-account</v-icon>
                </v-avatar>
              </template>

              <v-list-item-title class="font-weight-bold">
                {{ thread.contact_phone }}
              </v-list-item-title>
              <v-list-item-subtitle class="text-truncate">
                {{ thread.last_message_snippet || 'No messages' }}
              </v-list-item-subtitle>

              <template #append>
                <div class="text-end">
                  <div class="text-caption text-medium-emphasis">
                    {{ formatThreadTime(thread.last_message_at) }}
                  </div>
                  <v-badge
                    v-if="thread.unread_count > 0"
                    :content="thread.unread_count"
                    color="primary"
                    inline
                    class="mt-1"
                  />
                </div>
              </template>
            </v-list-item>
          </v-list>
        </v-col>

        <!-- Right Pane: Active Conversation -->
        <v-col cols="12" md="8" class="d-flex flex-column chat-pane">
          <!-- Active Conversation Header -->
          <div v-if="selectedThread" class="pa-4 border-b bg-surface d-flex align-center justify-space-between">
            <div class="d-flex align-center">
              <v-avatar color="primary" variant="tonal" size="40" class="mr-3">
                <v-icon>mdi-cellphone</v-icon>
              </v-avatar>
              <div>
                <h3 class="text-subtitle-1 font-weight-bold leading-tight">{{ selectedThread.contact_phone }}</h3>
                <span class="text-caption text-medium-emphasis">
                  Cellular SIM Gateway: {{ selectedThread.phone_id.substring(0, 8) }}...
                </span>
              </div>
            </div>

            <v-btn
              icon="mdi-refresh"
              variant="text"
              size="small"
              :loading="loadingMessages"
              @click="fetchMessagesForThread(selectedThread.id)"
            />
          </div>

          <!-- Empty State when no thread selected -->
          <div
            v-if="!selectedThread"
            class="flex-grow-1 d-flex flex-column align-center justify-center text-center pa-8"
          >
            <v-avatar color="grey-lighten-3" size="80" class="mb-4">
              <v-icon size="40" color="grey-darken-1">mdi-message-text-outline</v-icon>
            </v-avatar>
            <h3 class="text-h6 font-weight-bold mb-1">Select a Conversation</h3>
            <p class="text-body-2 text-medium-emphasis" style="max-width: 360px;">
              Choose an active thread from the left pane or send a new SMS to begin chatting.
            </p>
          </div>

          <!-- Message Bubbles Stream -->
          <div
            v-else
            ref="messageContainer"
            class="flex-grow-1 overflow-y-auto pa-4 bg-grey-lighten-4 message-stream"
          >
            <div v-if="loadingMessages && currentMessages.length === 0" class="text-center py-8">
              <v-progress-circular indeterminate color="primary" />
            </div>

            <div v-else-if="currentMessages.length === 0" class="text-center py-10 text-medium-emphasis">
              No messages in this thread yet.
            </div>

            <div
              v-for="msg in currentMessages"
              :key="msg.id"
              class="d-flex flex-column mb-3"
              :class="msg.direction === 'outbound' ? 'align-end' : 'align-start'"
            >
              <div
                class="pa-3 rounded-xl message-bubble"
                :class="msg.direction === 'outbound' ? 'bg-primary text-white' : 'bg-white text-high-emphasis elevation-1'"
              >
                <div class="text-body-1 whitespace-pre-wrap">{{ msg.body }}</div>
              </div>

              <div class="d-flex align-center ga-1 mt-1 text-caption text-medium-emphasis">
                <span>{{ formatMsgTime(msg.created_at) }}</span>
                <template v-if="msg.direction === 'outbound'">
                  <span>•</span>
                  <v-icon
                    size="14"
                    :color="msg.status === 'delivered' ? 'success' : (msg.status === 'failed' ? 'error' : 'grey')"
                  >
                    {{ msg.status === 'delivered' ? 'mdi-check-all' : (msg.status === 'sent' ? 'mdi-check' : 'mdi-clock-outline') }}
                  </v-icon>
                  <span class="text-uppercase" style="font-size: 10px;">{{ msg.status }}</span>
                </template>
              </div>
            </div>
          </div>

          <!-- Reply Input Bar -->
          <div v-if="selectedThread" class="pa-3 border-t bg-surface">
            <v-form @submit.prevent="sendReply">
              <div class="d-flex align-end ga-2">
                <v-textarea
                  v-model="replyText"
                  placeholder="Type an SMS reply..."
                  variant="outlined"
                  density="comfortable"
                  rows="2"
                  max-rows="4"
                  auto-grow
                  hide-details
                  class="flex-grow-1"
                  @keydown.enter.prevent="handleEnterPress"
                />
                <v-btn
                  type="submit"
                  color="primary"
                  icon="mdi-send"
                  size="large"
                  :loading="sendingReply"
                  :disabled="!replyText.trim()"
                />
              </div>
            </v-form>
          </div>
        </v-col>
      </v-row>
    </v-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useApi } from '~/composables/useApi'

const api = useApi()
const loadingThreads = ref(false)
const loadingMessages = ref(false)
const sendingReply = ref(false)

const threads = ref<any[]>([])
const selectedThread = ref<any>(null)
const currentMessages = ref<any[]>([])
const replyText = ref('')
const searchQuery = ref('')
const messageContainer = ref<HTMLElement | null>(null)

const filteredThreads = computed(() => {
  if (!searchQuery.value.trim()) return threads.value
  const q = searchQuery.value.toLowerCase()
  return threads.value.filter(t =>
    t.contact_phone.toLowerCase().includes(q) ||
    (t.last_message_snippet && t.last_message_snippet.toLowerCase().includes(q))
  )
})

async function fetchThreads() {
  loadingThreads.value = true
  try {
    const res = await api.get<any[]>('/api/v1/message-threads')
    if (res.success && res.data) {
      threads.value = res.data
      if (threads.value.length > 0 && !selectedThread.value) {
        selectThread(threads.value[0])
      }
    }
  } catch (err) {
    console.error('Failed to load threads', err)
  } finally {
    loadingThreads.value = false
  }
}

async function selectThread(thread: any) {
  selectedThread.value = thread
  await fetchMessagesForThread(thread.id)
}

async function fetchMessagesForThread(threadId: string) {
  loadingMessages.value = true
  try {
    const res = await api.get<any[]>(`/api/v1/message-threads/${threadId}/messages`)
    if (res.success && res.data) {
      currentMessages.value = res.data
      scrollToBottom()
    }
  } catch (err) {
    console.error('Failed to load thread messages', err)
  } finally {
    loadingMessages.value = false
  }
}

function handleEnterPress(e: KeyboardEvent) {
  if (!e.shiftKey) {
    sendReply()
  }
}

async function sendReply() {
  const text = replyText.value.trim()
  if (!text || !selectedThread.value) return

  sendingReply.value = true
  try {
    const res = await api.post<any>('/api/v1/messages', {
      phone_id: selectedThread.value.phone_id,
      recipient_phone: selectedThread.value.contact_phone,
      body: text,
      sim_slot: 1,
      priority: 'high',
    })

    if (res.success && res.data) {
      replyText.value = ''
      currentMessages.value.push(res.data)
      scrollToBottom()
      fetchThreads()
    }
  } catch (err) {
    console.error('Failed to dispatch reply', err)
  } finally {
    sendingReply.value = false
  }
}

function scrollToBottom() {
  nextTick(() => {
    if (messageContainer.value) {
      messageContainer.value.scrollTop = messageContainer.value.scrollHeight
    }
  })
}

function formatThreadTime(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function formatMsgTime(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

onMounted(() => {
  fetchThreads()
})
</script>

<style scoped>
.chat-container {
  height: calc(100vh - 180px);
  min-height: 520px;
}
.thread-list-pane {
  height: 100%;
}
.chat-pane {
  height: 100%;
}
.message-stream {
  overflow-y: auto;
}
.message-bubble {
  max-width: 75%;
  word-break: break-word;
}
.whitespace-pre-wrap {
  white-space: pre-wrap;
}
</style>
