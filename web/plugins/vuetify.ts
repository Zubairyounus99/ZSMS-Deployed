import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'

export default defineNuxtPlugin((nuxtApp) => {
  const vuetify = createVuetify({
    ssr: true,
    components,
    directives,
    theme: {
      defaultTheme: 'light',
      themes: {
        light: {
          dark: false,
          colors: {
            primary: '#0052CC',      // ZTechAI Blue
            secondary: '#172B4D',    // Deep Slate
            accent: '#00B8D9',       // Cyan Accent
            success: '#36B37E',      // Green
            warning: '#FFAB00',      // Amber
            error: '#FF5630',        // Red
            info: '#0065FF',         // Info
            background: '#F4F5F7',   // Light Gray Surface
            surface: '#FFFFFF',      // Pure White Card
          },
        },
        dark: {
          dark: true,
          colors: {
            primary: '#4C9AFF',      // High-contrast Light Blue
            secondary: '#B3D4FF',
            accent: '#00B8D9',
            success: '#57D9A3',
            warning: '#FFE380',
            error: '#FF8F73',
            info: '#4C9AFF',
            background: '#0D1117',   // GitHub / Dokploy Dark
            surface: '#161B22',      // Elevated Card Dark
          },
        },
      },
    },
  })

  nuxtApp.vueApp.use(vuetify)
})
