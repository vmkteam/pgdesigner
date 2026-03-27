<script setup lang="ts">
import { ref, nextTick, useTemplateRef, computed, shallowRef, triggerRef, type ShallowRef } from 'vue'
import { whenever } from '@vueuse/core'
import { DialogRoot, DialogPortal, DialogOverlay, DialogContent, DialogTitle } from 'reka-ui'
import { useProjectStore } from '@/stores/project'
import { useUiStore } from '@/stores/ui'
import api from '@/api/factory'
import type { IRecentFile, IDemoSchema, IDSNPreview } from '@/api/factory'
import { showToast } from '@/composables/useToast'
import FileBrowser from './FileBrowser.vue'
import { formatSize } from '@/utils/format'

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
type DSNStep = 'connect' | 'select-tables'
const dsnStep = ref<DSNStep>('connect')
const dsnConnecting = ref(false)
const dsnPreview = ref<IDSNPreview | null>(null)
const selectedSchemas = shallowRef(new Set<string>())
const selectedCategories = shallowRef(new Set<string>())
const selectedTables = shallowRef(new Set<string>())

// Category metadata for data-driven rendering
const categoryDefs = [
  { key: 'views', label: 'Views', field: 'views' as const },
  { key: 'matviews', label: 'Mat. Views', field: 'matViews' as const },
  { key: 'functions', label: 'Functions', field: 'functions' as const },
  { key: 'triggers', label: 'Triggers', field: 'triggers' as const },
  { key: 'enums', label: 'Enums', field: 'enums' as const },
  { key: 'domains', label: 'Domains', field: 'domains' as const },
  { key: 'sequences', label: 'Sequences', field: 'sequences' as const },
  { key: 'extensions', label: 'Extensions', field: 'extensions' as const },
]

type DSNPreviewArrayField = 'views' | 'matViews' | 'functions' | 'triggers' | 'enums' | 'domains' | 'sequences' | 'extensions'

const visibleCategories = computed(() => {
  if (!dsnPreview.value) return []
  const p = dsnPreview.value
  return categoryDefs.filter(c => p[c.field as DSNPreviewArrayField].length > 0)
})

function toggleSchema(key: string) { toggleSet(selectedSchemas, key) }
function toggleCategory(key: string) { toggleSet(selectedCategories, key) }
function toggleTable(key: string) { toggleSet(selectedTables, key) }

function toggleSet(setRef: ShallowRef<Set<string>>, key: string) {
  const s = setRef.value
  if (s.has(key)) s.delete(key); else s.add(key)
  triggerRef(setRef)
}

function setAllInSchema(schemaName: string, selected: boolean) {
  if (!dsnPreview.value) return
  const schema = dsnPreview.value.schemas.find(s => s.name === schemaName)
  if (!schema) return
  const s = selectedTables.value
  for (const t of schema.tables) {
    const key = schemaName + '.' + t.name
    if (selected) s.add(key); else s.delete(key)
  }
  triggerRef(selectedTables)
}

const dsnSelectedTableCount = computed(() => {
  if (dsnStep.value === 'select-tables') return selectedTables.value.size
  let count = 0
  for (const s of dsnPreview.value?.schemas ?? []) {
    if (selectedSchemas.value.has(s.name)) count += s.tables.length
  }
  return count
})

const dsnSelectedSchemaCount = computed(() => selectedSchemas.value.size)

const schemaTableCounts = computed(() => {
  if (!dsnPreview.value) return new Map<string, number>()
  const m = new Map<string, number>()
  for (const s of dsnPreview.value.schemas) {
    if (!selectedSchemas.value.has(s.name)) continue
    m.set(s.name, s.tables.filter(t => selectedTables.value.has(s.name + '.' + t.name)).length)
  }
  return m
})

const canOpen = computed(() => {
  if (loading.value) return false
  if (activeTab.value === 'recent') return !!selectedRecent.value
  if (activeTab.value === 'browse') return !!browserRef.value?.getSelectedPath()
  if (activeTab.value === 'dsn') {
    if (!dsnPreview.value) return !!dsn.value.trim()
    return dsnSelectedTableCount.value > 0
  }
  return false
})

whenever(() => ui.openDialogOpen, async () => {
  selectedRecent.value = ''
  dsn.value = ''
  loading.value = false
  browseInitialDir.value = ''
  dsnStep.value = 'connect'
  dsnPreview.value = null
  dsnConnecting.value = false
  selectedSchemas.value = new Set()
  selectedCategories.value = new Set()
  selectedTables.value = new Set()

  try {
    const [files, demoList] = await Promise.all([
      api.app.getRecentFilesInfo(),
      api.app.listDemoSchemas(),
    ])
    recentFiles.value = files ?? []
    demos.value = demoList ?? []
    activeTab.value = (files && files.length > 0) ? 'recent' : 'browse'

    if (activeTab.value === 'browse') {
      browseInitialDir.value = files?.length ? parentDir(files[0]!.path) : (store.info?.workDir || '')
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
    browseInitialDir.value = recentFiles.value.length ? parentDir(recentFiles.value[0]!.path) : (store.info?.workDir || '')
  }
  if (tab === 'dsn') {
    nextTick(() => dsnInputRef.value?.focus())
  }
}

// --- Recent tab ---

function onRecentClick(file: IRecentFile) {
  if (!file.exists) { showToast('File not found: ' + file.path, 'error'); return }
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

// --- Browse tab ---

function onBrowseSelect(path: string) { doOpenFile(path) }

// --- DSN tab ---

async function dsnConnect() {
  const val = dsn.value.trim()
  if (!val) return
  dsnConnecting.value = true
  try {
    const preview = await api.app.introspectDSN({ dsn: val })
    dsnPreview.value = preview
    selectedSchemas.value = new Set(preview.schemas.map(s => s.name))
    selectedCategories.value = new Set()
    selectedTables.value = new Set()
    dsnStep.value = 'connect'
  } catch (e) {
    showToast('Connection failed: ' + (e instanceof Error ? e.message : e), 'error')
  } finally {
    dsnConnecting.value = false
  }
}

function goToSelectTables() {
  if (!dsnPreview.value) return
  const tables = new Set<string>()
  for (const s of dsnPreview.value.schemas) {
    if (selectedSchemas.value.has(s.name)) {
      for (const t of s.tables) tables.add(s.name + '.' + t.name)
    }
  }
  selectedTables.value = tables
  dsnStep.value = 'select-tables'
}

async function dsnImport() {
  if (!dsnPreview.value) return
  loading.value = true
  try {
    const schemas = [...selectedSchemas.value]
    const tables = dsnStep.value === 'select-tables' ? [...selectedTables.value] : []
    const categories = [...selectedCategories.value]
    await api.app.importDSN({ dsn: dsn.value.trim(), schemas, tables, categories })
    await store.loadAll()
    ui.openDialogOpen = false
  } catch (e) {
    showToast('Import failed: ' + (e instanceof Error ? e.message : e), 'error')
  } finally {
    loading.value = false
  }
}

// --- Common ---

async function onOpen() {
  if (activeTab.value === 'dsn') {
    if (!dsnPreview.value) { dsnConnect(); return }
    dsnImport(); return
  }
  let path = ''
  if (activeTab.value === 'recent') path = selectedRecent.value
  else if (activeTab.value === 'browse') path = browserRef.value?.getSelectedPath() ?? ''
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
  if (e.key === 'Enter' && activeTab.value === 'recent') { e.preventDefault(); onOpen() }
  if (e.key === 'Enter' && activeTab.value === 'dsn' && !dsnPreview.value) { e.preventDefault(); dsnConnect() }
  if (activeTab.value === 'browse') { browserRef.value?.onKeydown(e) }
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

        <div class="od-tabs">
          <button class="od-tab" :class="{ 'od-tab-active': activeTab === 'recent' }" @click="switchTab('recent')">Recent</button>
          <button class="od-tab" :class="{ 'od-tab-active': activeTab === 'browse' }" @click="switchTab('browse')">Browse</button>
          <button class="od-tab" :class="{ 'od-tab-active': activeTab === 'dsn' }" @click="switchTab('dsn')">PostgreSQL</button>
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
                  @click="onRecentClick(f)" @dblclick="onRecentDblClick(f)"
                >
                  <div class="od-recent-name">{{ f.name }}</div>
                  <div class="od-recent-meta">
                    <span class="od-recent-path">{{ f.path }}</span>
                    <template v-if="f.exists">
                      <span class="od-dot">&middot;</span><span>{{ formatSize(f.size) }}</span>
                      <span class="od-dot">&middot;</span><span>{{ relativeTime(f.modTime) }}</span>
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
                <div v-for="d in demos" :key="d.name" class="od-demo-item" @click="openDemo(d.name)">
                  <span class="od-demo-name">{{ d.title }}</span>
                  <span class="od-demo-info">{{ d.tables }} tables, {{ d.fks }} FK</span>
                </div>
              </div>
            </div>
          </div>

          <!-- Browse tab -->
          <div v-if="activeTab === 'browse'" class="od-browse">
            <FileBrowser ref="browserRef" mode="open" :initial-dir="browseInitialDir" @select="onBrowseSelect" />
          </div>

          <!-- DSN tab -->
          <div v-if="activeTab === 'dsn'" class="od-dsn-wizard">
            <div class="od-dsn-connect">
              <div class="od-dsn-label"><span class="od-dsn-label-link" @click="dsn = dsn || 'postgres://user:pass@localhost:5432/dbname'">Connection string</span></div>
              <div class="od-dsn-row">
                <input ref="dsnInputRef" v-model="dsn" class="od-input od-dsn-input" placeholder="postgres://user:pass@localhost:5432/dbname" :disabled="dsnConnecting" @keydown.enter.prevent="dsnConnect" />
                <button class="od-btn od-btn-connect" :disabled="!dsn.trim() || dsnConnecting" @click="dsnConnect">{{ dsnConnecting ? 'Connecting...' : 'Connect' }}</button>
              </div>
              <div v-if="dsnPreview" class="od-dsn-db-info">{{ dsnPreview.database }} &middot; PostgreSQL {{ dsnPreview.pgVersion }}</div>
            </div>

            <!-- Step 1: two-column preview -->
            <div v-if="dsnPreview && dsnStep === 'connect'" class="od-dsn-preview">
              <div class="od-dsn-col">
                <div class="od-dsn-section">
                  <div class="od-list-label">Schemas</div>
                  <div class="od-dsn-objects">
                    <label v-for="s in dsnPreview.schemas" :key="s.name" class="od-dsn-check">
                      <input type="checkbox" :checked="selectedSchemas.has(s.name)" @change="toggleSchema(s.name)" />
                      <span class="od-dsn-check-name">{{ s.name }}</span>
                      <span class="od-dsn-check-count">{{ s.tables.length }}</span>
                    </label>
                  </div>
                </div>
              </div>
              <div class="od-dsn-col">
                <div v-if="visibleCategories.length" class="od-dsn-section">
                  <div class="od-list-label">Include</div>
                  <div class="od-dsn-objects">
                    <label v-for="c in visibleCategories" :key="c.key" class="od-dsn-check">
                      <input type="checkbox" :checked="selectedCategories.has(c.key)" @change="toggleCategory(c.key)" />
                      <span>{{ c.label }}</span>
                      <span class="od-dsn-check-count">{{ dsnPreview![c.field as DSNPreviewArrayField].length }}</span>
                    </label>
                  </div>
                </div>
                <div v-if="dsnPreview.roles.length" class="od-dsn-section">
                  <div class="od-list-label">Security <span class="od-dsn-coming">(coming soon)</span></div>
                  <div class="od-dsn-objects od-dsn-disabled">
                    <label class="od-dsn-check"><input type="checkbox" disabled /><span>Roles</span><span class="od-dsn-check-count">{{ dsnPreview.roles.length }}</span></label>
                    <label v-if="dsnPreview.grants" class="od-dsn-check"><input type="checkbox" disabled /><span>Grants</span><span class="od-dsn-check-count">{{ dsnPreview.grants }}</span></label>
                    <label v-if="dsnPreview.defaultPrivileges" class="od-dsn-check"><input type="checkbox" disabled /><span>Default Privs</span><span class="od-dsn-check-count">{{ dsnPreview.defaultPrivileges }}</span></label>
                  </div>
                </div>
              </div>
            </div>

            <!-- Step 2: table selection -->
            <div v-if="dsnPreview && dsnStep === 'select-tables'" class="od-dsn-tables">
              <template v-for="s in dsnPreview.schemas" :key="s.name">
                <div v-if="selectedSchemas.has(s.name)" class="od-dsn-schema-group">
                  <div class="od-dsn-schema-header">
                    <span class="od-dsn-schema-name">{{ s.name }}</span>
                    <span class="od-dsn-schema-count">{{ schemaTableCounts.get(s.name) ?? 0 }} / {{ s.tables.length }}</span>
                    <button class="od-dsn-link" @click="setAllInSchema(s.name, true)">All</button>
                    <button class="od-dsn-link" @click="setAllInSchema(s.name, false)">None</button>
                  </div>
                  <div class="od-dsn-table-list">
                    <label v-for="t in s.tables" :key="t.name" class="od-dsn-table-item">
                      <input type="checkbox" :checked="selectedTables.has(s.name + '.' + t.name)" @change="toggleTable(s.name + '.' + t.name)" />
                      <span class="od-dsn-table-name">{{ t.name }}</span>
                      <span class="od-dsn-table-meta">{{ t.columns }} cols &middot; {{ t.indexes }} idx &middot; {{ t.fks }} FK</span>
                    </label>
                  </div>
                </div>
              </template>
            </div>

            <div v-if="!dsnPreview && !dsnConnecting" class="od-dsn-hint">
              Imports schema from a live PostgreSQL database via reverse engineering (read-only, no changes to DB).
            </div>
          </div>
        </div>

        <!-- Footer -->
        <div class="od-footer">
          <template v-if="activeTab === 'dsn' && dsnPreview">
            <div class="od-footer-info">{{ dsnSelectedTableCount }} tables from {{ dsnSelectedSchemaCount }} schemas</div>
            <div class="od-footer-spacer"></div>
            <button class="od-btn" @click="ui.openDialogOpen = false">Cancel</button>
            <template v-if="dsnStep === 'connect'">
              <button class="od-btn" :disabled="dsnSelectedTableCount === 0" @click="goToSelectTables">Select Tables...</button>
              <button class="od-btn od-btn-primary" :disabled="loading || dsnSelectedTableCount === 0" @click="dsnImport">{{ loading ? 'Importing...' : 'Import All' }}</button>
            </template>
            <template v-else>
              <button class="od-btn" @click="dsnStep = 'connect'">&larr; Back</button>
              <button class="od-btn od-btn-primary" :disabled="loading || dsnSelectedTableCount === 0" @click="dsnImport">{{ loading ? 'Importing...' : 'Import Selected' }}</button>
            </template>
          </template>
          <template v-else>
            <button class="od-btn" @click="ui.openDialogOpen = false">Cancel</button>
            <button class="od-btn od-btn-primary" :disabled="!canOpen" @click="onOpen">{{ loading ? 'Opening...' : activeTab === 'dsn' ? 'Connect' : 'Open' }}</button>
          </template>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<style>
.od-overlay { position: fixed; inset: 0; background: rgba(0,0,0,.3); z-index: 60; }
.od-box { position: fixed; z-index: 70; top: 50%; left: 50%; transform: translate(-50%, -50%); width: 36rem; height: 32rem; background: var(--color-bg-surface); border: 1px solid var(--color-menu-border); box-shadow: 0 4px 16px rgba(0,0,0,.25); display: flex; flex-direction: column; }
.od-title { padding: 0.615rem 0.923rem; font-size: 0.923rem; font-weight: 600; color: var(--color-text-primary); background: var(--color-bg-app); border-bottom: 1px solid var(--color-border); flex-shrink: 0; }
.od-tabs { display: flex; gap: 0; border-bottom: 1px solid var(--color-border); background: var(--color-bg-app); flex-shrink: 0; }
.od-tab { padding: 0.462rem 0.923rem; font-size: 0.846rem; background: transparent; border: none; border-bottom: 2px solid transparent; color: var(--color-text-secondary); cursor: pointer; }
.od-tab:hover { color: var(--color-text-primary); }
.od-tab-active { color: var(--color-text-primary); font-weight: 600; border-bottom-color: var(--color-accent); }
.od-body { flex: 1; overflow-y: auto; padding: 0.769rem 0.923rem; min-height: 0; display: flex; flex-direction: column; }
.od-recent { display: flex; flex-direction: column; gap: 0.923rem; }
.od-list-section { display: flex; flex-direction: column; gap: 0.308rem; }
.od-list-label { font-size: 0.692rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em; color: var(--color-text-muted); }
.od-list { display: flex; flex-direction: column; gap: 0.077rem; }
.od-recent-item { padding: 0.385rem 0.538rem; cursor: pointer; position: relative; display: flex; flex-direction: column; gap: 0.077rem; border: 1px solid transparent; }
.od-recent-item:hover { background: var(--color-bg-hover); }
.od-recent-item.od-selected { background: var(--color-bg-hover); border-color: var(--color-accent); }
.od-recent-item.od-missing { opacity: 0.4; }
.od-recent-name { font-size: 0.846rem; font-weight: 600; color: var(--color-text-primary); }
.od-recent-meta { font-size: 0.692rem; color: var(--color-text-muted); display: flex; align-items: center; gap: 0.308rem; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.od-recent-path { overflow: hidden; text-overflow: ellipsis; flex-shrink: 1; min-width: 0; }
.od-dot { color: var(--color-text-muted); }
.od-missing-label { color: var(--color-text-muted); font-style: italic; }
.od-remove-btn { position: absolute; top: 0.308rem; right: 0.308rem; width: 1.231rem; height: 1.231rem; font-size: 0.923rem; line-height: 1; background: transparent; border: none; color: var(--color-text-muted); cursor: pointer; display: none; align-items: center; justify-content: center; }
.od-recent-item:hover .od-remove-btn { display: flex; }
.od-remove-btn:hover { color: var(--color-text-primary); }
.od-demo-item { display: flex; align-items: center; gap: 0.462rem; padding: 0.385rem 0.538rem; cursor: pointer; }
.od-demo-item:hover { background: var(--color-bg-hover); }
.od-demo-name { font-size: 0.846rem; font-weight: 600; color: var(--color-text-primary); min-width: 8rem; }
.od-demo-info { font-size: 0.692rem; color: var(--color-text-secondary); }
.od-browse { display: flex; flex-direction: column; flex: 1; min-height: 0; }
.od-dsn-wizard { display: flex; flex-direction: column; gap: 0.615rem; flex: 1; min-height: 0; }
.od-dsn-connect { display: flex; flex-direction: column; gap: 0.308rem; flex-shrink: 0; }
.od-dsn-label { font-size: 0.846rem; font-weight: 600; color: var(--color-text-primary); }
.od-dsn-label-link { border-bottom: 1px dashed var(--color-text-muted); cursor: pointer; }
.od-dsn-label-link:hover { border-bottom-color: var(--color-accent); color: var(--color-accent); }
.od-dsn-row { display: flex; gap: 0.308rem; }
.od-dsn-input { flex: 1; }
.od-btn-connect { white-space: nowrap; }
.od-dsn-db-info { font-size: 0.769rem; color: var(--color-text-secondary); font-weight: 600; }
.od-dsn-hint { font-size: 0.692rem; color: var(--color-text-secondary); font-style: italic; }
.od-dsn-coming { font-weight: 400; font-style: italic; text-transform: none; letter-spacing: 0; }
.od-dsn-preview { display: flex; gap: 0.923rem; flex: 1; min-height: 0; overflow-y: auto; }
.od-dsn-col { flex: 1; display: flex; flex-direction: column; gap: 0.615rem; min-width: 0; overflow-y: auto; }
.od-dsn-section { display: flex; flex-direction: column; gap: 0.231rem; }
.od-dsn-objects { display: flex; flex-direction: column; gap: 0.077rem; }
.od-dsn-disabled { opacity: 0.4; }
.od-dsn-check { display: flex; align-items: center; gap: 0.385rem; font-size: 0.846rem; color: var(--color-text-primary); cursor: pointer; padding: 0.154rem 0; }
.od-dsn-check input { margin: 0; cursor: pointer; }
.od-dsn-check-name { font-weight: 600; }
.od-dsn-check-count { font-size: 0.692rem; color: var(--color-text-muted); }
.od-dsn-tables { display: flex; flex-direction: column; gap: 0.615rem; flex: 1; overflow-y: auto; min-height: 0; }
.od-dsn-schema-group { display: flex; flex-direction: column; gap: 0.154rem; }
.od-dsn-schema-header { display: flex; align-items: center; gap: 0.462rem; padding: 0.231rem 0; }
.od-dsn-schema-name { font-size: 0.846rem; font-weight: 700; color: var(--color-text-primary); }
.od-dsn-schema-count { font-size: 0.692rem; color: var(--color-text-muted); margin-right: auto; }
.od-dsn-link { font-size: 0.692rem; color: var(--color-accent); background: none; border: none; cursor: pointer; text-decoration: underline; padding: 0; }
.od-dsn-link:hover { color: var(--color-text-primary); }
.od-dsn-table-list { display: flex; flex-direction: column; gap: 0; border: 1px solid var(--color-border); max-height: 12rem; overflow-y: auto; }
.od-dsn-table-item { display: flex; align-items: center; gap: 0.385rem; padding: 0.231rem 0.462rem; font-size: 0.769rem; cursor: pointer; }
.od-dsn-table-item:hover { background: var(--color-bg-hover); }
.od-dsn-table-item input { margin: 0; cursor: pointer; }
.od-dsn-table-name { font-weight: 600; color: var(--color-text-primary); flex: 1; }
.od-dsn-table-meta { font-size: 0.692rem; color: var(--color-text-muted); }
.od-input { width: 100%; padding: 0.308rem 0.462rem; font-size: 0.846rem; border: 1px solid var(--color-border); background: var(--color-bg-surface); color: var(--color-text-primary); outline: none; box-sizing: border-box; }
.od-input:focus { border-color: var(--color-accent); }
.od-footer { padding: 0.462rem 0.923rem; background: var(--color-bg-app); border-top: 1px solid var(--color-border); display: flex; align-items: center; gap: 0.308rem; flex-shrink: 0; }
.od-footer-info { font-size: 0.692rem; color: var(--color-text-muted); }
.od-footer-spacer { flex: 1; }
.od-btn { padding: 0.231rem 0.923rem; font-size: 0.923rem; border: 1px solid var(--color-menu-border); background: var(--color-bg-surface); color: var(--color-text-primary); cursor: default; }
.od-btn:hover:not(:disabled) { background: var(--color-bg-hover); }
.od-btn:disabled { opacity: 0.5; }
.od-btn-primary { font-weight: 600; }
</style>
