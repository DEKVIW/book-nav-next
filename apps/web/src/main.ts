import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './app/App.vue'
import { router } from './app/router'
import { usePortalStore } from './shared/stores/portal'
import './shared/styles/tokens.css'
import './shared/styles/base.css'
import './shared/styles/mecha.css'
import './shared/styles/cursors.css'
import './shared/styles/admin.css' // console redesign (independent of portal mecha)

const app = createApp(App)
const pinia = createPinia()
app.use(pinia)

if (window.location.pathname === '/') {
  void usePortalStore(pinia).loadHome()
}

app.use(router)
app.mount('#app')
