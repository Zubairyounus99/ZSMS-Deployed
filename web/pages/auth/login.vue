<template>
  <div class="auth-wrapper d-flex align-center justify-center">
    <v-card width="440" rounded="xl" elevation="4" class="pa-8">
      <!-- Brand Header -->
      <div class="text-center mb-6">
        <v-avatar color="primary" size="56" class="mb-3">
          <v-icon size="32" color="white">mdi-cellphone-wireless</v-icon>
        </v-avatar>
        <h2 class="text-h5 font-weight-bold text-primary">ZSMS Gateway</h2>
        <p class="text-caption text-medium-emphasis">ZTechAI Enterprise SMS/MMS Platform</p>
      </div>

      <!-- Auth Mode Tabs -->
      <v-tabs v-model="tab" grow class="mb-6" color="primary">
        <v-tab value="login">Sign In</v-tab>
        <v-tab value="register">Register</v-tab>
      </v-tabs>

      <!-- Error Alert -->
      <v-alert
        v-if="errorMessage"
        type="error"
        variant="tonal"
        closable
        density="compact"
        class="mb-4"
        @click:close="errorMessage = ''"
      >
        {{ errorMessage }}
      </v-alert>

      <!-- Form -->
      <v-form @submit.prevent="handleSubmit">
        <v-text-field
          v-model="email"
          label="Email Address"
          type="email"
          variant="outlined"
          density="comfortable"
          prepend-inner-icon="mdi-email-outline"
          class="mb-3"
          :rules="[v => !!v || 'Email is required']"
          required
        />

        <v-text-field
          v-model="password"
          label="Password"
          type="password"
          variant="outlined"
          density="comfortable"
          prepend-inner-icon="mdi-lock-outline"
          class="mb-4"
          :rules="[v => !!v || 'Password is required']"
          required
        />

        <v-btn
          type="submit"
          color="primary"
          block
          size="large"
          rounded="lg"
          :loading="authStore.loading"
          class="text-none font-weight-bold"
        >
          {{ tab === 'login' ? 'Sign In to ZSMS' : 'Create Account & Sign In' }}
        </v-btn>
      </v-form>

      <div class="text-center mt-6">
        <span class="text-caption text-medium-emphasis">
          {{ tab === 'login' ? 'First time here? Switch to the "Register" tab to set up your account.' : 'Already have an account? Switch to the "Sign In" tab.' }}
        </span>
      </div>
    </v-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAuthStore } from '~/stores/auth'

definePageMeta({
  layout: false,
})

const authStore = useAuthStore()
const tab = ref<'login' | 'register'>('login')
const email = ref('admin@ztechai.us')
const password = ref('Admin1234!')
const errorMessage = ref('')

onMounted(() => {
  authStore.initAuth()
  if (authStore.isAuthenticated) {
    navigateTo('/dashboard')
  }
})

async function handleSubmit() {
  errorMessage.value = ''
  try {
    if (tab.value === 'login') {
      await authStore.login(email.value, password.value)
    } else {
      await authStore.register(email.value, password.value)
    }
    navigateTo('/dashboard')
  } catch (err: any) {
    errorMessage.value = err.message || 'Authentication failed'
  }
}
</script>

<style scoped>
.auth-wrapper {
  min-height: 100vh;
  background-color: #f4f6f8;
}
</style>
