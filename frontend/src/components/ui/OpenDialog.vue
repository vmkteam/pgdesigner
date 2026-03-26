<script setup lang="ts">
import { ref, nextTick, useTemplateRef, computed } from 'vue'
import { whenever } from '@vueuse/core'
import { DialogRoot, DialogPortal, DialogOverlay, DialogContent, DialogTitle } from 'reka-ui'
import { useProjectStore } from '@/stores/project'
import { useUiStore } from '@/stores/ui'
import api from '@/api/factory'
import type { IRecentFile, IDemoSchema } from '@/api/factory'
import { showToast } from '@/composables/useToast'
import FileBrowser from './FileBrowser.vue'

const store = useProjectStore()
const ui = useUiStore()

type Tab = 'recent' | 'browse' | 'dsn'
const activeTab = ref<Tab>('recent')
const loading = ref(false)

// Recent tab
const recentFiles = ref<IRecentFile[]>([])
const demos = ref<IDemoSchema[]>([])
const selectedRecent = ref('')

// Browse tab
const browserRef = useTemplateRef<InstanceType<typeof FileBrowser>>('browserRef')
const browseInitialDir = ref('')

// DSN tab
const dsn = ref('')
const dsnInputRef = useTemplateRef<HTMLInputElement>('dsnInputRef')

const canOpen = computed(() => {
  if (loading.value) return false
  if (activeTab.value === 'recent') return !!selectedRecent.value
  if (activeTab.value === 'browse') return !!browserRef.value?.getSelectedPath()
  if (activeTab.value === 'dsn') return !!dsn.value.trim()
  return false
})

whenever(() => ui.openDialogOpen, async () => {
  selectedRecent.value = ''
  dsn.value = ''
  loading.value = false
  browseInitialDir.value = ''

  try {
    const [files, demoList] = await Promise.all([
      api.app.getRecentFilesInfo(),
      api.app.listDemoSchemas(),
    ])
    recentFiles.value = files ?? []
    demos.value = demoList ?? []
    activeTab.value = (files && files.length > 0) ? 'recent' : 'browse'

    if (activeTab.value === 'browse') {
      browseInitialDir.value = files?.length ? parentDir(files[0]!.path) : ''
    }
  } catch { /* ignore */ }
})

function parentDir(path: string): string {
  const i = path.lastIndexOf('/')
  return i > 0 ? path.substring(0, i) : '/'
}

async function switchTab(tab: Tab) {
  activeTab.value = tab
  if (tab === 'browse' && !browseInitialDir.value) {
    browseInitialDir.value = recentFiles.value.length ? parentDir(recentFiles.value[0]!.path) : ''
  }
  if (tab === 'dsn') {
    nextTick(() => dsnInputRef.value?.focus())
  }
}

function onRecentClick(file: IRecentFile) {
  if (!file.exists) {
    showToast('File not found: ' + file.path, 'error')
    return
  }
  selectedRecent.value = file.path
}

function onRecentDblClick(file: IRecentFile) {
  if (!file.exists) return
  selectedRecent.value = file.path
  onOpen()
}

async function removeRecent(e: Event, path: string) {
  e.stopPropagation()
  try {
    await api.app.removeRecentFile({ path })
    recentFiles.value = recentFiles.value.filter(f => f.path !== path)
    if (selectedRecent.value === path) selectedRecent.value = ''
  } catch { /* ignore */ }
}

async function openDemo(name: string) {
  loading.value = true
  try {
    await api.app.openDemo({ name })
    await store.loadAll()
    ui.openDialogOpen = false
  } catch (e) {
    showToast('Open failed: ' + (e instanceof Error ? e.message : e), 'error')
  } finally {
    loading.value = false
  }
}

function onBrowseSelect(path: string) {
  doOpenFile(path)
}

async function onOpen() {
  let path = ''
  if (activeTab.value === 'recent') path = selectedRecent.value
  else if (activeTab.value === 'browse') path = browserRef.value?.getSelectedPath() ?? ''
  else if (activeTab.value === 'dsn') path = dsn.value.trim()
  if (!path) return
  doOpenFile(path)
}

async function doOpenFile(path: string) {
  loading.value = true
  try {
    await api.app.openFile({ path })
    await store.loadAll()
    ui.openDialogOpen = false
  } catch (e) {
    showToast('Open failed: ' + (e instanceof Error ? e.message : e), 'error')
  } finally {
    loading.value = false
  }
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && activeTab.value === 'recent') {
    e.preventDefault()
    onOpen()
  }
  if (e.key === 'Enter' && activeTab.value === 'dsn') {
    e.preventDefault()
    onOpen()
  }
  if (activeTab.value === 'browse') {
    browserRef.value?.onKeydown(e)
  }
}

function formatSize(bytes: number): string {
  if (bytes < 0) return ''
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(0) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function relativeTime(isoDate: string): string {
  if (!isoDate) return ''
  const diff = Date.now() - new Date(isoDate).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return mins + 'm ago'
  const hours = Math.floor(mins / 60)
  if (hours < 24) return hours + 'h ago'
  const days = Math.floor(hours / 24)
  if (days < 7) return days + 'd ago'
  const weeks = Math.floor(days / 7)
  if (weeks < 5) return weeks + 'w ago'
  return new Date(isoDate).toLocaleDateString()
}
</script>

<template>
  <DialogRoot :open="ui.openDialogOpen">
    <DialogPortal>
      <DialogOverlay class="od-overlay" @click="ui.openDialogOpen = false" />
      <DialogContent class="od-box" @escape-key-down="ui.openDialogOpen = false" @keydown="onKeydown">
        <DialogTitle class="od-title">Open</DialogTitle>

        <!-- Tabs -->
        <div class="od-tabs">
          <button
            class="od-tab" :class="{ 'od-tab-active': activeTab === 'recent' }"
            @click="switchTab('recent')"
          >Recent</button>
          <button
            class="od-tab" :class="{ 'od-tab-active': activeTab === 'browse' }"
            @click="switchTab('browse')"
          >Browse</button>
          <button
            class="od-tab" :class="{ 'od-tab-active': activeTab === 'dsn' }"
            @click="switchTab('dsn')"
          >PostgreSQL DSN</button>
        </div>

        <div class="od-body">
          <!-- Recent tab -->
          <div v-if="activeTab === 'recent'" class="od-recent">
            <div v-if="recentFiles.length" class="od-list-section">
              <div class="od-list-label">Recent Files</div>
              <div class="od-list">
                <div
                  v-for="f in recentFiles" :key="f.path"
                  class="od-recent-item"
                  :class="{ 'od-selected': selectedRecent === f.path, 'od-missing': !f.exists }"
                  @click="onRecentClick(f)"
                  @dblclick="onRecentDblClick(f)"
                >
                  <div class="od-recent-name">{{ f.name }}</div>
                  <div class="od-recent-meta">
                    <span class="od-recent-path">{{ f.path }}</span>
                    <template v-if="f.exists">
                      <span class="od-dot">&middot;</span>
                      <span>{{ formatSize(f.size) }}</span>
                      <span class="od-dot">&middot;</span>
                      <span>{{ relativeTime(f.modTime) }}</span>
                    </template>
                    <span v-else class="od-missing-label">(missing)</span>
                  </div>
                  <button class="od-remove-btn" title="Remove from list" @click="removeRecent($event, f.path)">&times;</button>
                </div>
              </div>
            </div>

            <div class="od-list-section">
              <div class="od-list-label">Demo Schemas</div>
              <div class="od-list">
                <div
                  v-for="d in demos" :key="d.name"
                  class="od-demo-item"
                  @click="openDemo(d.name)"
                >
                  <span class="od-demo-name">{{ d.title }}</span>
                  <span class="od-demo-info">{{ d.tables }} tables, {{ d.fks }} FK</span>
                </div>
              </div>
            </div>
          </div>

          <!-- Browse tab -->
          <div v-if="activeTab === 'browse'" class="od-browse">
            <FileBrowser
              ref="browserRef"
              mode="open"
              :initial-dir="browseInitialDir"
              @select="onBrowseSelect"
            />
          </div>

          <!-- DSN tab -->
          <div v-if="activeTab === 'dsn'" class="od-dsn">
            <div class="od-dsn-label"><span class="od-dsn-label-link" @click="dsn = dsn || 'postgres://user:pass@localhost:5432/dbname'">Connection string</span></div>
            <input
              ref="dsnInputRef"
              v-model="dsn"
              class="od-input"
              placeholder="postgres://user:pass@localhost:5432/dbname"
              @keydown.enter.prevent="onOpen"
            />
            <div class="od-dsn-hint">Imports schema from a live PostgreSQL database via reverse engineering (read-only, no changes to DB).</div>
          </div>
        </div>

        <div class="od-footer">
          <button class="od-btn" @click="ui.openDialogOpen = false">Cancel</button>
          <button
            class="od-btn od-btn-primary"
            :disabled="!canOpen"
            @click="onOpen"
          >{{ loading ? 'Opening...' : 'Open' }}</button>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<style>
.od-overlay { position: fixed; inset: 0; background: rgba(0,0,0,.3); z-index: 60; }
.od-box {
  position: fixed; z-index: 70;
  top: 50%; left: 50%; transform: translate(-50%, -50%);
  width: 36rem; height: 32rem;
  background: var(--color-bg-surface); border: 1px solid var(--color-menu-border);
  box-shadow: 0 4px 16px rgba(0,0,0,.25);
  display: flex; flex-direction: column;
}
.od-title {
  padding: 0.615rem 0.923rem;
  font-size: 0.923rem; font-weight: 600; color: var(--color-text-primary);
  background: var(--color-bg-app); border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

/* Tabs */
.od-tabs {
  display: flex; gap: 0; border-bottom: 1px solid var(--color-border);
  background: var(--color-bg-app); flex-shrink: 0;
}
.od-tab {
  padding: 0.462rem 0.923rem; font-size: 0.846rem;
  background: transparent; border: none; border-bottom: 2px solid transparent;
  color: var(--color-text-secondary); cursor: pointer;
}
.od-tab:hover { color: var(--color-text-primary); }
.od-tab-active {
  color: var(--color-text-primary); font-weight: 600;
  border-bottom-color: var(--color-accent);
}

.od-body { flex: 1; overflow-y: auto; padding: 0.769rem 0.923rem; min-height: 0; display: flex; flex-direction: column; }

/* Recent tab */
.od-recent { display: flex; flex-direction: column; gap: 0.923rem; }
.od-list-section { display: flex; flex-direction: column; gap: 0.308rem; }
.od-list-label {
  font-size: 0.692rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em;
  color: var(--color-text-muted);
}
.od-list { display: flex; flex-direction: column; gap: 0.077rem; }

.od-recent-item {
  padding: 0.385rem 0.538rem; cursor: pointer; position: relative;
  display: flex; flex-direction: column; gap: 0.077rem;
  border: 1px solid transparent;
}
.od-recent-item:hover { background: var(--color-bg-hover); }
.od-recent-item.od-selected { background: var(--color-bg-hover); border-color: var(--color-accent); }
.od-recent-item.od-missing { opacity: 0.4; }
.od-recent-name { font-size: 0.846rem; font-weight: 600; color: var(--color-text-primary); }
.od-recent-meta {
  font-size: 0.692rem; color: var(--color-text-muted);
  display: flex; align-items: center; gap: 0.308rem;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.od-recent-path { overflow: hidden; text-overflow: ellipsis; flex-shrink: 1; min-width: 0; }
.od-dot { color: var(--color-text-muted); }
.od-missing-label { color: var(--color-text-muted); font-style: italic; }
.od-remove-btn {
  position: absolute; top: 0.308rem; right: 0.308rem;
  width: 1.231rem; height: 1.231rem; font-size: 0.923rem; line-height: 1;
  background: transparent; border: none; color: var(--color-text-muted);
  cursor: pointer; display: none; align-items: center; justify-content: center;
}
.od-recent-item:hover .od-remove-btn { display: flex; }
.od-remove-btn:hover { color: var(--color-text-primary); }

.od-demo-item {
  display: flex; align-items: center; gap: 0.462rem;
  padding: 0.385rem 0.538rem; cursor: pointer;
}
.od-demo-item:hover { background: var(--color-bg-hover); }
.od-demo-name { font-size: 0.846rem; font-weight: 600; color: var(--color-text-primary); min-width: 8rem; }
.od-demo-info { font-size: 0.692rem; color: var(--color-text-secondary); }

/* Browse tab */
.od-browse { display: flex; flex-direction: column; flex: 1; min-height: 0; }

/* DSN tab */
.od-dsn { display: flex; flex-direction: column; gap: 0.462rem; padding: 0.462rem 0; }
.od-dsn-label { font-size: 0.846rem; font-weight: 600; color: var(--color-text-primary); }
.od-dsn-label-link {
  border-bottom: 1px dashed var(--color-text-muted); cursor: pointer;
}
.od-dsn-label-link:hover { border-bottom-color: var(--color-accent); color: var(--color-accent); }
.od-input {
  width: 100%; padding: 0.308rem 0.462rem; font-size: 0.846rem;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-primary); outline: none; box-sizing: border-box;
}
.od-input:focus { border-color: var(--color-accent); }
.od-dsn-hint { font-size: 0.692rem; color: var(--color-text-secondary); font-style: italic; }

/* Footer */
.od-footer {
  padding: 0.462rem 0.923rem;
  background: var(--color-bg-app); border-top: 1px solid var(--color-border);
  display: flex; justify-content: flex-end; gap: 0.308rem;
  flex-shrink: 0;
}
.od-btn {
  padding: 0.231rem 0.923rem; font-size: 0.923rem;
  border: 1px solid var(--color-menu-border); background: var(--color-bg-surface);
  color: var(--color-text-primary); cursor: default;
}
.od-btn:hover:not(:disabled) { background: var(--color-bg-hover); }
.od-btn:disabled { opacity: 0.5; }
.od-btn-primary { font-weight: 600; }
</style>
