<template>
  <div class="landing-root">
    <header class="ld-nav">
      <div class="ld-wrap ld-nav-inner">
        <a class="ld-brand" href="/">
          <img src="/logo.png" alt="Agent Hub" @error="($event.target as HTMLImageElement).style.display='none'" />
          <span>Agent Hub</span>
        </a>
        <nav class="ld-nav-links">
          <a href="#product">{{ t('landing.nav.product') }}</a>
          <a :href="setupDoc" target="_blank" rel="noopener">{{ t('landing.nav.docs') }}</a>
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

    <section class="ld-hero">
      <div class="ld-wrap">
        <h1>{{ t('landing.hero.title') }}</h1>
        <p>{{ t('landing.hero.sub') }}</p>
        <div class="ld-hero-cta">
          <router-link class="ld-btn ld-btn-primary ld-btn-lg" :to="loggedIn ? '/app' : '/login?mode=register'">
            {{ loggedIn ? t('landing.nav.app') : t('landing.cta.start') }}
          </router-link>
          <a class="ld-btn ld-btn-ghost ld-btn-lg" :href="setupDoc" target="_blank" rel="noopener">
            {{ t('landing.cta.docs') }}
          </a>
        </div>
      </div>
    </section>

    <section id="product" class="ld-section">
      <div class="ld-wrap">
        <h2>{{ t('landing.pillars.title') }}</h2>
        <div class="ld-grid" style="margin-top: 36px">
          <article class="ld-card">
            <h3>{{ t('landing.pillar.agentflow.title') }}</h3>
            <p>{{ t('landing.pillar.agentflow.body') }}</p>
          </article>
          <article class="ld-card">
            <h3>{{ t('landing.pillar.mcp.title') }}</h3>
            <p>{{ t('landing.pillar.mcp.body') }}</p>
          </article>
          <article class="ld-card">
            <h3>{{ t('landing.pillar.sync.title') }}</h3>
            <p>{{ t('landing.pillar.sync.body') }}</p>
          </article>
          <article class="ld-card">
            <h3>{{ t('landing.pillar.templates.title') }}</h3>
            <p>{{ t('landing.pillar.templates.body') }}</p>
          </article>
          <article class="ld-card">
            <h3>{{ t('landing.pillar.events.title') }}</h3>
            <p>{{ t('landing.pillar.events.body') }}</p>
          </article>
        </div>
      </div>
    </section>

    <section class="ld-section ld-section-alt">
      <div class="ld-wrap">
        <h2>{{ t('landing.how.title') }}</h2>
        <div class="ld-steps" style="margin-top: 36px">
          <div class="ld-step">
            <div class="ld-step-num">1</div>
            <h3>{{ t('landing.how.step1.title') }}</h3>
            <p>{{ t('landing.how.step1.body') }}</p>
          </div>
          <div class="ld-step">
            <div class="ld-step-num">2</div>
            <h3>{{ t('landing.how.step2.title') }}</h3>
            <p>{{ t('landing.how.step2.body') }}</p>
          </div>
          <div class="ld-step">
            <div class="ld-step-num">3</div>
            <h3>{{ t('landing.how.step3.title') }}</h3>
            <p>{{ t('landing.how.step3.body') }}</p>
          </div>
        </div>
      </div>
    </section>

    <section class="ld-section">
      <div class="ld-wrap ld-dev">
        <div class="ld-dev-copy">
          <h2>{{ t('landing.dev.title') }}</h2>
          <p>{{ t('landing.dev.hint') }}</p>
        </div>
        <div class="ld-code">
          <button type="button" class="ld-code-btn" @click="copySnippet">
            {{ copied ? t('landing.dev.copied') : t('landing.dev.copy') }}
          </button>
{{ mcpSnippet }}
        </div>
      </div>
    </section>

    <section class="ld-section ld-section-alt">
      <div class="ld-wrap">
        <h2>{{ t('landing.enterprise.title') }}</h2>
        <div class="ld-enterprise" style="margin-top: 36px">
          <div class="ld-ent-item"><span class="ld-ent-dot" />{{ t('landing.enterprise.m1') }}</div>
          <div class="ld-ent-item"><span class="ld-ent-dot" />{{ t('landing.enterprise.m2') }}</div>
          <div class="ld-ent-item"><span class="ld-ent-dot" />{{ t('landing.enterprise.m3') }}</div>
          <div class="ld-ent-item"><span class="ld-ent-dot" />{{ t('landing.enterprise.m4') }}</div>
        </div>
      </div>
    </section>

    <section class="ld-final">
      <div class="ld-wrap">
        <h2>{{ t('landing.final.title') }}</h2>
        <p>{{ t('landing.final.sub') }}</p>
        <div class="ld-final-actions">
          <router-link class="ld-btn ld-btn-primary ld-btn-lg" :to="loggedIn ? '/app' : '/login?mode=register'">
            {{ loggedIn ? t('landing.nav.app') : t('landing.cta.start') }}
          </router-link>
          <a class="ld-btn ld-btn-ghost ld-btn-lg" :href="hubRepo" target="_blank" rel="noopener">agent-hub</a>
          <a class="ld-btn ld-btn-ghost ld-btn-lg" :href="afRepo" target="_blank" rel="noopener">agentflow</a>
        </div>
      </div>
    </section>

    <footer class="ld-footer">
      <div class="ld-wrap ld-footer-inner">
        <div>© {{ year }} {{ t('landing.footer.rights') }} · Agent Hub</div>
        <div class="ld-footer-links">
          <a :href="hubRepo" target="_blank" rel="noopener">{{ t('landing.footer.hubRepo') }}</a>
          <a :href="afRepo" target="_blank" rel="noopener">{{ t('landing.footer.afRepo') }}</a>
          <a :href="setupDoc" target="_blank" rel="noopener">{{ t('landing.footer.setup') }}</a>
          <a :href="afSetupDoc" target="_blank" rel="noopener">agentflow-setup</a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from '@/i18n'
import '@/styles/landing.css'

const { t, locale, setLocale } = useI18n()
const loggedIn = ref(!!localStorage.getItem('token'))
const copied = ref(false)
const year = new Date().getFullYear()

const setupDoc = 'https://hub.stifer.xyz/agent-setup.md'
const afSetupDoc = 'https://hub.stifer.xyz/agentflow-setup.md'
const hubRepo = 'https://github.com/toustifer/agent-hub'
const afRepo = 'https://github.com/toustifer/agentflow'

const mcpSnippet = `{
  "mcpServers": {
    "hub": {
      "type": "http",
      "url": "https://hub.stifer.xyz/mcp"
    }
  }
}`

function toggleLang() {
  setLocale(locale.value === 'zh' ? 'en' : 'zh')
}

async function copySnippet() {
  try {
    await navigator.clipboard.writeText(mcpSnippet)
    copied.value = true
    setTimeout(() => { copied.value = false }, 1500)
  } catch {
    /* ignore */
  }
}

onMounted(() => {
  document.title = 'Agent Hub — AI Project Management for Teams'
  loggedIn.value = !!localStorage.getItem('token')
})
</script>
