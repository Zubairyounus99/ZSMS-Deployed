<template>
  <div class="roadmap-placeholder">
    <v-card elevation="1" class="pa-8 text-center rounded-lg">
      <v-avatar color="primary" variant="tonal" size="72" class="mb-4">
        <v-icon size="36" color="primary">{{ icon }}</v-icon>
      </v-avatar>

      <h1 class="text-h5 font-weight-bold mb-2">{{ title }}</h1>
      <p class="text-body-1 text-medium-emphasis mb-4" style="max-width: 600px; margin: 0 auto;">
        {{ description }}
      </p>

      <v-chip color="primary" variant="outlined" class="font-weight-medium">
        Target Milestone: {{ stage }}
      </v-chip>

      <v-divider class="my-6" />

      <div class="text-caption text-medium-emphasis">
        ZTechAI ZSMS Architecture Specification — Clean-room implementation adheres strictly to planned milestones. No placeholder mock data.
      </div>
    </v-card>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  description: string
  stage: string
  icon: string
}>()
</script>
