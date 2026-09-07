<template>
  <div class="zsms-app-layout">
    <!-- Top Navigation App Bar -->
    <v-app-bar elevation="1" density="comfortable">
      <v-app-bar-nav-icon @click="drawer = !drawer" />

      <v-toolbar-title class="font-weight-bold d-flex align-center">
        <v-icon color="primary" class="mr-2">mdi-cellphone-wireless</v-icon>
        <span>ZSMS</span>
        <v-chip size="x-small" color="primary" variant="flat" class="ml-2 font-weight-bold">
          GATEWAY
        </v-chip>
      </v-toolbar-title>

      <v-spacer />

      <!-- API Connectivity Status Badge -->
      <v-chip
        size="small"
        :color="apiStatusColor"
        variant="tonal"
        class="mr-3 font-weight-medium"
      >
        <v-icon start size="small">{{ apiStatusIcon }}</v-icon>
        API: {{ apiStatusText }}
      </v-chip>

      <!-- Theme Toggle -->
      <v-btn icon size="small" class="mr-2" @click="toggleTheme">
        <v-icon>{{ isDark ? 'mdi-weather-sunny' : 'mdi-weather-night' }}</v-icon>
      </v-btn>

      <!-- User Menu -->
      <v-menu v-if="authStore.isAuthenticated" location="bottom end">
        <template #activator="{ props }">
          <v-avatar color="primary" size="32" class="cursor-pointer" v-bind="props">
            <span class="text-white text-caption font-weight-bold">
              {{ (authStore.user?.email?.[0] || 'Z').toUpperCase() }}
            </span>
          </v-avatar>
        </template>
        <v-list density="compact" min-width="180">
          <v-list-item
            :title="authStore.user?.email || 'Logged In'"
            :subtitle="authStore.userRole"
            prepend-icon="mdi-account-circle"
          />
          <v-divider />
          <v-list-item
            prepend-icon="mdi-logout"
            title="Sign Out"
            @click="authStore.logout()"
          />
        </v-list>
      </v-menu>
      <v-btn v-else to="/auth/login" size="small" color="primary" variant="flat">
        Sign In
      </v-btn>
    </v-app-bar>

    <!-- Side Navigation Drawer -->
    <v-navigation-drawer v-model="drawer" elevation="1">
      <v-list-item
        prepend-icon="mdi-domain"
        title="ZTechAI"
        subtitle="ZSMS Enterprise"
        class="py-3"
      />

      <v-divider />

      <v-list density="compact" nav>
        <v-list-subheader class="font-weight-bold text-uppercase">Messaging</v-list-subheader>
        <v-list-item
          v-for="item in messagingNav"
          :key="item.to"
          :to="item.to"
          :prepend-icon="item.icon"
          :title="item.title"
          active-color="primary"
        />

        <v-list-subheader class="font-weight-bold text-uppercase mt-3">Gateways & Sim</v-list-subheader>
        <v-list-item
          to="/phones"
          prepend-icon="mdi-cellphone-link"
          title="Phones & SIMs"
          active-color="primary"
        />

        <v-list-subheader class="font-weight-bold text-uppercase mt-3">Developer & APIs</v-list-subheader>
        <v-list-item
          v-for="item in devNav"
          :key="item.to"
          :to="item.to"
          :prepend-icon="item.icon"
          :title="item.title"
          active-color="primary"
        />

        <v-list-subheader class="font-weight-bold text-uppercase mt-3">Management</v-list-subheader>
        <v-list-item
          v-for="item in mgmtNav"
          :key="item.to"
          :to="item.to"
          :prepend-icon="item.icon"
          :title="item.title"
          active-color="primary"
        />
      </v-list>

      <template #append>
        <div class="pa-3 text-center">
          <span class="text-caption text-medium-emphasis">ZSMS Foundation v1.0.0</span>
        </div>
      </template>
    </v-navigation-drawer>

    <!-- Page Content -->
    <v-main>
      <v-container fluid class="pa-6">
        <slot />
      </v-container>
    </v-main>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useTheme } from 'vuetify'
import { useAuthStore } from '~/stores/auth'

const authStore = useAuthStore()
const drawer = ref(true)
const theme = useTheme()

const isDark = computed(() => theme.global.current.value.dark)

function toggleTheme() {
  theme.global.name.value = isDark.value ? 'light' : 'dark'
}

const apiStatus = ref<'checking' | 'online' | 'degraded' | 'offline'>('checking')

const apiStatusColor = computed(() => {
  switch (apiStatus.value) {
    case 'online': return 'success'
    case 'degraded': return 'warning'
    case 'offline': return 'error'
    default: return 'grey'
  }
})

const apiStatusIcon = computed(() => {
  switch (apiStatus.value) {
    case 'online': return 'mdi-check-circle-outline'
    case 'degraded': return 'mdi-alert-circle-outline'
    case 'offline': return 'mdi-close-circle-outline'
    default: return 'mdi-sync'
  }
})

const apiStatusText = computed(() => {
  switch (apiStatus.value) {
    case 'online': return 'Online'
    case 'degraded': return 'Degraded'
    case 'offline': return 'Offline'
    default: return 'Checking...'
  }
})

const messagingNav = [
  { title: 'Dashboard', to: '/dashboard', icon: 'mdi-view-dashboard-outline' },
  { title: 'Messages', to: '/messages', icon: 'mdi-message-text-outline' },
  { title: 'Conversations', to: '/conversations', icon: 'mdi-forum-outline' },
  { title: 'Contacts', to: '/contacts', icon: 'mdi-account-box-multiple-outline' },
  { title: 'Campaigns', to: '/campaigns', icon: 'mdi-bullhorn-outline' },
  { title: 'Scheduled', to: '/scheduled', icon: 'mdi-calendar-clock-outline' },
]

const devNav = [
  { title: 'API Keys', to: '/api-keys', icon: 'mdi-key-variant' },
  { title: 'Webhooks', to: '/webhooks', icon: 'mdi-webhook' },
  { title: 'API Docs', to: '/docs', icon: 'mdi-book-open-page-variant-outline' },
]

const mgmtNav = [
  { title: 'Analytics', to: '/analytics', icon: 'mdi-chart-line' },
  { title: 'Usage', to: '/usage', icon: 'mdi-chart-bar' },
  { title: 'Users & Roles', to: '/users', icon: 'mdi-account-group-outline' },
  { title: 'Settings', to: '/settings', icon: 'mdi-cog-outline' },
  { title: 'Administration', to: '/admin', icon: 'mdi-shield-crown-outline' },
]

onMounted(async () => {
  authStore.initAuth()
  if (!authStore.isAuthenticated) {
    navigateTo('/auth/login')
  }

  const config = useRuntimeConfig()
  const apiEndpoint = (config.public.apiBaseUrl as string) || (config.public.apiUrl as string) || 'https://sms-api.ztechai.us'
  try {
    const res = await fetch(`${apiEndpoint}/livez`, { method: 'GET' })
    if (res.ok) {
      apiStatus.value = 'online'
    } else {
      apiStatus.value = 'degraded'
    }
  } catch {
    apiStatus.value = 'offline'
  }
})
</script>

<style scoped>
.zsms-app-layout {
  min-height: 100vh;
}
</style>
