// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2024-04-03',
  devtools: { enabled: false },
  ssr: true,

  css: [
    'vuetify/styles',
    '@mdi/font/css/materialdesignicons.css',
  ],

  build: {
    transpile: ['vuetify'],
  },

  modules: [
    '@pinia/nuxt',
  ],

  runtimeConfig: {
    public: {
      apiBaseUrl: process.env.NUXT_PUBLIC_API_BASE_URL || process.env.NUXT_PUBLIC_API_URL || 'https://sms-api.ztechai.us',
      apiUrl: process.env.NUXT_PUBLIC_API_BASE_URL || process.env.NUXT_PUBLIC_API_URL || 'https://sms-api.ztechai.us',
      appName: process.env.NUXT_PUBLIC_APP_NAME || 'ZSMS',
    },
  },

  app: {
    head: {
      title: 'ZSMS — ZTechAI SMS/MMS Gateway',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'Production Android SMS/MMS Cellular Gateway for ZTechAI' },
      ],
      link: [
        { rel: 'icon', type: 'image/svg+xml', href: '/logo.svg' },
      ],
    },
  },

  typescript: {
    strict: true,
    typeCheck: false,
  },
})
