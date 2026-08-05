<template>
  <div class="landing-root">
    <header class="ld-nav">
      <div class="ld-wrap ld-nav-inner">
        <router-link class="ld-brand" to="/">
          <img src="/logo.png" alt="Agent Hub" @error="($event.target as HTMLImageElement).style.display='none'" />
          <span>Agent Hub</span>
        </router-link>
        <nav class="ld-nav-links">
          <a :href="productHref">{{ t('landing.nav.product') }}</a>
          <router-link to="/docs" :class="{ 'ld-nav-active': isDocs }">{{ t('landing.nav.docs') }}</router-link>
          <a :href="hubRepo" target="_blank" rel="noopener">{{ t('landing.nav.github') }}</a>
        </nav>
        <div class="ld-nav-actions">
          <button type="button" class="ld-lang" @click="toggleLang">{{ locale === 'zh' ? 'EN' : '中' }}</button>
          <template v-if="loggedIn">
            <router-link class="ld-btn ld-btn-primary" to="/app">{{ t('landing.nav.app') }}</router-link>
          </template>
          <template v-else>
            <router-link class="ld-btn ld-btn-ghost" to="/login">{{ t('landing.nav.login') }}</router-link>
            <router-link class="ld-btn ld-btn-primary" to="/login?mode=register">{{ t('landing.nav.start') }}</router-link>
          </template>
        </div>
      </div>
    </header>

    <slot />

    <footer class="ld-footer">
      <div class="ld-wrap ld-footer-inner">
        <div>© {{ year }} {{ t('landing.footer.rights') }} · Agent Hub</div>
        <div class="ld-footer-links">
          <a :href="hubRepo" target="_blank" rel="noopener">{{ t('landing.footer.hubRepo') }}</a>
          <a :href="afRepo" target="_blank" rel="noopener">{{ t('landing.footer.afRepo') }}</a>
          <router-link to="/docs">{{ t('landing.footer.setup') }}</router-link>
          <a :href="afSetupDoc" target="_blank" rel="noopener">agentflow-setup</a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from '@/i18n'
import '@/styles/landing.css'

const { t, locale, setLocale } = useI18n()
const route = useRoute()
const loggedIn = ref(!!localStorage.getItem('token'))
const year = new Date().getFullYear()

const hubRepo = 'https://github.com/toustifer/agent-hub'
const afRepo = 'https://github.com/toustifer/agentflow'
const afSetupDoc = computed(() => `${window.location.origin}/agentflow-setup.md`)

const isDocs = computed(() => route.path === '/docs' || route.path.startsWith('/docs/'))
const productHref = computed(() => (isDocs.value ? '/#product' : '#product'))

function toggleLang() {
  setLocale(locale.value === 'zh' ? 'en' : 'zh')
}

onMounted(() => {
  loggedIn.value = !!localStorage.getItem('token')
})
</script>
