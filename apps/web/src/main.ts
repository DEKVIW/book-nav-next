import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './app/App.vue'
import { router } from './app/router'
import './shared/styles/tokens.css'
import './shared/styles/base.css'
import './shared/styles/mecha.css'
import './shared/styles/cursors.css'
import './shared/styles/admin.css' // console redesign (independent of portal mecha)

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
