<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import api from '@/api/factory'
import type { IDemoSchema, IRecentFile, IAboutInfo } from '@/api/factory'
import { useProjectStore } from '@/stores/project'
import { useUiStore } from '@/stores/ui'
import { showToast } from '@/composables/useToast'

const store = useProjectStore()
const ui = useUiStore()

const demos = ref<IDemoSchema[]>([])
const recentFiles = ref<IRecentFile[]>([])
const about = ref<IAboutInfo | null>(null)
const loadingDemo = ref<string | null>(null)

const topRecent = computed(() => recentFiles.value.filter(f => f.exists).slice(0, 2))

onMounted(async () => {
  try {
    const [d, r, a] = await Promise.all([
      api.app.listDemoSchemas(),
      api.app.getRecentFilesInfo(),
      api.app.about(),
    ])
    demos.value = d
    recentFiles.value = r ?? []
    about.value = a
  } catch { /* ignore */ }
})

async function openDemo(name: string) {
  loadingDemo.value = name
  try {
    await api.app.openDemo({ name })
    await store.loadAll()
  } catch (e) {
    showToast(e instanceof Error ? e.message : String(e), 'error')
  } finally {
    loadingDemo.value = null
  }
}

async function openRecent(path: string) {
  try {
    await api.app.openFile({ path })
    await store.loadAll()
  } catch (e) {
    showToast(e instanceof Error ? e.message : String(e), 'error')
  }
}

async function newProject() {
  try {
    await api.app.newProject()
    await store.loadAll()
    ui.isWelcome = false
    ui.settingsOpen = true
  } catch (e) {
    showToast(e instanceof Error ? e.message : String(e), 'error')
  }
}

function openFile() {
  ui.openDialogOpen = true
}
</script>

<template>
  <div class="ws-root">
    <div class="ws-card">
      <div class="ws-header">
        <svg class="ws-icon" width="40" height="40" viewBox="0 0 64 64" fill="none">
          <rect x="14" y="14" width="36" height="36" rx="10" fill="#2F5D7C"/>
          <rect x="22" y="24" width="20" height="3" rx="1.5" fill="#FFFFFF"/>
          <rect x="22" y="30" width="14" height="3" rx="1.5" fill="#FFFFFF" opacity="0.7"/>
          <rect x="22" y="36" width="18" height="3" rx="1.5" fill="#FFFFFF" opacity="0.4"/>
        </svg>
        <div>
          <div class="ws-title">PgDesigner</div>
          <div class="ws-subtitle">Visual PostgreSQL Schema Designer</div>
        </div>
      </div>

      <div class="ws-section">
        <div class="ws-section-title">Open Demo Schema</div>
        <div class="ws-demos">
          <button
            v-for="d in demos" :key="d.name"
            class="ws-demo-btn"
            :disabled="loadingDemo !== null"
            @click="openDemo(d.name)"
          >
            <span class="ws-demo-name">{{ d.title }}</span>
            <span class="ws-demo-info">{{ d.tables }} tables, {{ d.fks }} FK</span>
            <span v-if="loadingDemo === d.name" class="ws-demo-spinner"></span>
          </button>
        </div>
      </div>

      <div class="ws-section">
        <button class="ws-action-btn ws-action-primary" @click="newProject">
          <span class="ws-action-icon">✦</span>
          New Project
          <span class="ws-action-hint">Empty schema with defaults</span>
        </button>
        <button class="ws-action-btn" @click="openFile">
          <span class="ws-action-icon">+</span>
          Import / Open Schema
          <span class="ws-action-hint">.pgd .pdd .dbs .dm2 .sql PostgreSQL</span>
        </button>
      </div>

      <div v-if="topRecent.length" class="ws-section">
        <div class="ws-section-title">Recent Files</div>
        <div class="ws-recent">
          <button v-for="f in topRecent" :key="f.path" class="ws-recent-item" :title="f.path" @click="openRecent(f.path)">
            <span class="ws-recent-name">{{ f.name }}</span>
            <span class="ws-recent-path">{{ f.path }}</span>
          </button>
        </div>
      </div>

      <div class="ws-footer">
        <div v-if="!store.info?.isRegistered">Unregistered — non-commercial use only. <a href="https://pgdesigner.io/pricing" target="_blank" class="ws-footer-link">Buy License $19</a></div>
        <div v-if="about" class="ws-version">{{ about.version }}</div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ws-root {
  flex: 1; display: flex; align-items: center; justify-content: center;
  background: var(--color-bg-app);
}
.ws-card {
  width: 26rem; display: flex; flex-direction: column; gap: 1.231rem;
  background: var(--color-bg-surface); border: 1px solid var(--color-border);
  padding: 1.846rem; box-shadow: 0 2px 12px rgba(0,0,0,.15);
}

.ws-header { display: flex; align-items: center; gap: 0.769rem; }
.ws-icon { flex-shrink: 0; }
.ws-title { font-size: 1.385rem; font-weight: 700; color: var(--color-text-primary); }
.ws-subtitle { font-size: 0.846rem; color: var(--color-text-secondary); margin-top: 0.077rem; }

.ws-section { display: flex; flex-direction: column; gap: 0.308rem; }
.ws-section-title {
  font-size: 0.769rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em;
  color: var(--color-text-muted); margin-bottom: 0.154rem;
}

.ws-demos { display: flex; flex-direction: column; gap: 0.154rem; }
.ws-demo-btn {
  display: flex; align-items: center; gap: 0.462rem;
  padding: 0.462rem 0.615rem; text-align: left;
  background: transparent; border: 1px solid transparent;
  color: var(--color-text-primary); font-size: 0.923rem; cursor: pointer;
}
.ws-demo-btn:hover { background: var(--color-bg-hover); border-color: var(--color-border); }
.ws-demo-btn:disabled { opacity: 0.6; cursor: wait; }
.ws-demo-name { font-weight: 600; min-width: 8rem; }
.ws-demo-info { font-size: 0.769rem; color: var(--color-text-secondary); }
.ws-demo-spinner {
  width: 0.923rem; height: 0.923rem; border: 2px solid var(--color-border);
  border-top-color: var(--color-accent); border-radius: 50%;
  animation: spin 0.6s linear infinite; margin-left: auto;
}

.ws-action-btn {
  display: flex; align-items: center; gap: 0.462rem;
  padding: 0.615rem; text-align: left;
  background: transparent; border: 1px solid var(--color-border);
  color: var(--color-text-primary); font-size: 0.923rem; cursor: pointer;
}
.ws-action-btn:hover { background: var(--color-bg-hover); }
.ws-action-primary { border-color: var(--color-accent); font-weight: 600; }
.ws-action-icon { font-size: 1.077rem; font-weight: 700; color: var(--color-accent); width: 1.231rem; text-align: center; }
.ws-action-hint { font-size: 0.692rem; color: var(--color-text-muted); margin-left: auto; }

.ws-recent { display: flex; flex-direction: column; gap: 0.154rem; }
.ws-recent-item {
  display: flex; flex-direction: column; gap: 0.077rem;
  padding: 0.385rem 0.615rem;
  background: transparent; border: 1px solid transparent;
  text-align: left; width: 100%; cursor: pointer;
}
.ws-recent-item:hover { background: var(--color-bg-hover); border-color: var(--color-border); }
.ws-recent-name { font-size: 0.846rem; font-weight: 600; color: var(--color-text-primary); }
.ws-recent-path {
  font-size: 0.692rem; color: var(--color-text-muted);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}

.ws-footer {
  margin-top: 0.615rem; padding-top: 0.769rem; border-top: 1px solid var(--color-border);
  font-size: 0.769rem; color: var(--color-text-muted); text-align: center;
}
.ws-version { margin-top: 0.231rem; }
.ws-footer-link { color: var(--color-accent); text-decoration: none; }
.ws-footer-link:hover { text-decoration: underline; }

@keyframes spin { to { transform: rotate(360deg); } }
</style>
