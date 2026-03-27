<script setup lang="ts">
import { ref, computed, useTemplateRef, watch, onMounted } from 'vue'
import api from '@/api/factory'
import type { IDirEntry } from '@/api/factory'
import { showToast } from '@/composables/useToast'
import { formatSize } from '@/utils/format'

const props = withDefaults(defineProps<{
  initialDir?: string
  initialFileName?: string
  mode?: 'open' | 'save'
  showFilter?: boolean
  defaultExtension?: string
}>(), {
  initialDir: '',
  initialFileName: '',
  mode: 'open',
  showFilter: true,
})

const emit = defineEmits<{
  (e: 'select', path: string): void
  (e: 'save', path: string): void
}>()

const currentDir = ref('')
const entries = ref<IDirEntry[]>([])
const showAllFiles = ref(false)
const loadingDir = ref(false)
const selectedIndex = ref(-1)
const selectedFile = ref('')
const fileName = ref('')
const homePath = ref('')
const pathInputRef = useTemplateRef<HTMLInputElement>('pathInputRef')

const breadcrumbs = computed(() => {
  if (!currentDir.value) return []
  const home = homePath.value
  let display = currentDir.value
  if (home && display.startsWith(home)) {
    display = '~' + display.slice(home.length)
  }
  const parts = display.split('/').filter(Boolean)
  const result: { label: string; path: string }[] = []
  const isAbsolute = display.startsWith('/')
  const isTilde = parts[0]?.startsWith('~')
  if (isAbsolute && !isTilde) {
    result.push({ label: '/', path: '/' })
  }
  for (let i = 0; i < parts.length; i++) {
    let path: string
    if (isTilde) {
      path = i === 0 ? '~' : '~/' + parts.slice(1, i + 1).join('/')
    } else if (isAbsolute) {
      path = '/' + parts.slice(0, i + 1).join('/')
    } else {
      path = parts.slice(0, i + 1).join('/')
    }
    result.push({ label: parts[i]!, path })
  }
  return result
})

const fileExistsInDir = computed(() => {
  if (!fileName.value) return false
  return entries.value.some(e => !e.isDir && e.name === fileName.value)
})

async function init() {
  try {
    homePath.value = await api.app.getHomePath()
  } catch { /* ignore */ }
  const dir = props.initialDir || homePath.value
  fileName.value = props.initialFileName || ''
  if (dir) await loadDir(dir)
}

async function loadDir(path: string) {
  loadingDir.value = true
  selectedFile.value = ''
  selectedIndex.value = -1
  try {
    const result = await api.app.listDirectory({ path, showAll: showAllFiles.value })
    currentDir.value = result.path
    entries.value = result.entries
  } catch (e) {
    showToast('Failed to list directory: ' + (e instanceof Error ? e.message : e), 'error')
  } finally {
    loadingDir.value = false
  }
}

function parentDir(path: string): string {
  const i = path.lastIndexOf('/')
  return i > 0 ? path.substring(0, i) : '/'
}

function goUp() {
  loadDir(parentDir(currentDir.value))
}

function onEntryClick(entry: IDirEntry, index: number) {
  if (entry.isDir) {
    loadDir(currentDir.value + '/' + entry.name)
    return
  }
  selectedIndex.value = index
  if (props.mode === 'save') {
    fileName.value = entry.name
  } else {
    if (!entry.supported) return
    selectedFile.value = currentDir.value + '/' + entry.name
  }
}

function onEntryDblClick(entry: IDirEntry) {
  if (entry.isDir) {
    loadDir(currentDir.value + '/' + entry.name)
    return
  }
  if (props.mode === 'open') {
    if (!entry.supported) return
    selectedFile.value = currentDir.value + '/' + entry.name
    emit('select', selectedFile.value)
  } else {
    fileName.value = entry.name
    doSave()
  }
}

async function onPathSubmit() {
  const val = pathInputRef.value?.value?.trim()
  if (!val) return
  await loadDir(val)
}

async function toggleShowAll() {
  showAllFiles.value = !showAllFiles.value
  if (currentDir.value) await loadDir(currentDir.value)
}

function doSave() {
  let name = fileName.value.trim()
  if (!name) return
  const ext = props.defaultExtension || '.pgd'
  if (!name.includes('.')) name += ext
  fileName.value = name
  const fullPath = currentDir.value + '/' + name
  emit('save', fullPath)
}

function getSelectedPath(): string {
  return selectedFile.value
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Backspace' && document.activeElement?.tagName !== 'INPUT') {
    e.preventDefault()
    goUp()
  }
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    navigateList(1)
  }
  if (e.key === 'ArrowUp') {
    e.preventDefault()
    navigateList(-1)
  }
  if (e.key === 'Enter' && selectedIndex.value >= 0 && props.mode === 'open') {
    e.preventDefault()
    const entry = entries.value[selectedIndex.value]
    if (entry) onEntryDblClick(entry)
  }
}

function navigateList(delta: number) {
  const len = entries.value.length
  if (!len) return
  let idx = selectedIndex.value + delta
  if (idx < 0) idx = 0
  if (idx >= len) idx = len - 1
  selectedIndex.value = idx
  const entry = entries.value[idx]
  if (!entry) return
  if (props.mode === 'save') {
    if (!entry.isDir) fileName.value = entry.name
  } else {
    if (!entry.isDir && entry.supported) {
      selectedFile.value = currentDir.value + '/' + entry.name
    } else {
      selectedFile.value = ''
    }
  }
}


function formatModTime(isoDate: string): string {
  if (!isoDate) return ''
  const d = new Date(isoDate)
  const month = d.toLocaleString('en', { month: 'short' })
  const day = d.getDate()
  const time = d.toLocaleTimeString('en', { hour: '2-digit', minute: '2-digit', hour12: false })
  return `${month} ${day} ${time}`
}

defineExpose({ init, getSelectedPath, doSave, onKeydown, fileExistsInDir, fileName })

onMounted(() => init())
watch(() => props.initialDir, () => { if (props.initialDir) init() })
</script>

<template>
  <div class="fb-root" @keydown="onKeydown">
    <div class="fb-path-bar">
      <div class="fb-breadcrumbs">
        <span
          v-for="(bc, i) in breadcrumbs" :key="i"
          class="fb-breadcrumb"
          @click="loadDir(bc.path)"
        >{{ bc.label }}<span v-if="i < breadcrumbs.length - 1 && bc.label !== '/'" class="fb-sep">/</span></span>
      </div>
      <button class="fb-up-btn" title="Parent directory" @click="goUp">&uarr;</button>
    </div>

    <input
      ref="pathInputRef"
      class="fb-path-input"
      :value="currentDir"
      placeholder="/path/to/directory"
      @keydown.enter.prevent="onPathSubmit"
    />

    <div class="fb-file-list" :class="{ 'fb-loading': loadingDir }">
      <div v-if="!entries.length && !loadingDir" class="fb-empty">No files</div>
      <div
        v-for="(e, i) in entries" :key="e.name"
        class="fb-file-item"
        :class="{
          'fb-file-dir': e.isDir,
          'fb-file-unsupported': !e.isDir && !e.supported && mode === 'open',
          'fb-selected': i === selectedIndex
        }"
        @click="onEntryClick(e, i)"
        @dblclick="onEntryDblClick(e)"
      >
        <svg v-if="e.isDir" class="fb-file-icon" viewBox="0 0 16 16" fill="none"><path d="M1 4h5l1.5-2H15v11H1z" stroke="currentColor" stroke-width="1.2" fill="none"/></svg>
        <svg v-else-if="e.name.endsWith('.pgd')" class="fb-file-icon-pgd" viewBox="0 0 16 16" fill="none"><rect x="1" y="1" width="14" height="14" rx="3" fill="#2F5D7C"/><rect x="4" y="5" width="8" height="1.5" rx=".75" fill="#fff"/><rect x="4" y="7.5" width="6" height="1.5" rx=".75" fill="#fff" opacity=".7"/><rect x="4" y="10" width="7" height="1.5" rx=".75" fill="#fff" opacity=".4"/></svg>
        <svg v-else class="fb-file-icon" viewBox="0 0 16 16" fill="none"><path d="M3 1h7l3 3v11H3z" stroke="currentColor" stroke-width="1.2" fill="none"/><path d="M10 1v3h3" stroke="currentColor" stroke-width="1.2"/></svg>
        <span class="fb-file-name">{{ e.name }}{{ e.isDir ? '/' : '' }}</span>
        <span v-if="!e.isDir" class="fb-file-size">{{ formatSize(e.size) }}</span>
        <span v-if="!e.isDir" class="fb-file-date">{{ formatModTime(e.modTime) }}</span>
      </div>
    </div>

    <div v-if="showFilter" class="fb-footer">
      <span class="fb-filter-hint">.pgd .pdd .dbs .dm2 .sql</span>
      <label class="fb-show-all">
        <input type="checkbox" :checked="showAllFiles" @change="toggleShowAll" />
        Show all files
      </label>
    </div>

    <div v-if="mode === 'save'" class="fb-save-row">
      <label class="fb-save-label">File name:</label>
      <input
        v-model="fileName"
        class="fb-save-input"
        placeholder="schema.pgd"
        @keydown.enter.prevent="doSave"
      />
    </div>
  </div>
</template>

<style>
.fb-root { display: flex; flex-direction: column; gap: 0.462rem; flex: 1; min-height: 0; }

.fb-path-bar { display: flex; align-items: center; gap: 0.308rem; }
.fb-breadcrumbs {
  flex: 1; font-size: 0.769rem; color: var(--color-text-secondary);
  overflow-x: auto; white-space: nowrap;
}
.fb-breadcrumb { cursor: pointer; }
.fb-breadcrumb:hover { color: var(--color-text-primary); }
.fb-sep { margin: 0 0.154rem; color: var(--color-text-muted); }
.fb-up-btn {
  padding: 0.154rem 0.462rem; font-size: 0.846rem;
  background: var(--color-bg-surface); border: 1px solid var(--color-border);
  color: var(--color-text-primary); cursor: pointer;
}
.fb-up-btn:hover { background: var(--color-bg-hover); }

.fb-path-input {
  width: 100%; padding: 0.308rem 0.462rem; font-size: 0.769rem;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-primary); outline: none; box-sizing: border-box;
}
.fb-path-input:focus { border-color: var(--color-accent); }

.fb-file-list {
  flex: 1; overflow-y: auto; min-height: 0;
  border: 1px solid var(--color-border);
}
.fb-file-list.fb-loading { opacity: 0.5; }
.fb-empty {
  padding: 1.538rem; text-align: center;
  font-size: 0.846rem; color: var(--color-text-muted);
}
.fb-file-item {
  display: flex; align-items: center; gap: 0.385rem;
  padding: 0.231rem 0.462rem; font-size: 0.769rem; cursor: pointer;
}
.fb-file-item:hover { background: var(--color-bg-hover); }
.fb-file-item.fb-selected { background: var(--color-bg-hover); outline: 1px solid var(--color-accent); }
.fb-file-item.fb-file-unsupported { opacity: 0.4; cursor: default; }
.fb-file-icon { width: 1rem; height: 1rem; flex-shrink: 0; color: var(--color-text-muted); }
.fb-file-icon-pgd { width: 1rem; height: 1rem; flex-shrink: 0; }
.fb-file-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--color-text-primary); }
.fb-file-dir .fb-file-name { font-weight: 600; }
.fb-file-size { color: var(--color-text-muted); min-width: 3.5rem; text-align: right; }
.fb-file-date { color: var(--color-text-muted); min-width: 6.5rem; text-align: right; }

.fb-footer {
  display: flex; align-items: center; justify-content: space-between;
  font-size: 0.692rem; color: var(--color-text-muted);
}
.fb-filter-hint { display: flex; gap: 0.308rem; }
.fb-show-all {
  display: flex; align-items: center; gap: 0.308rem; cursor: pointer;
  font-size: 0.692rem; color: var(--color-text-secondary);
}
.fb-show-all input { margin: 0; cursor: pointer; }

.fb-save-row {
  display: flex; align-items: center; gap: 0.462rem; flex-shrink: 0;
}
.fb-save-label { font-size: 0.846rem; font-weight: 600; color: var(--color-text-primary); white-space: nowrap; }
.fb-save-input {
  flex: 1; padding: 0.308rem 0.462rem; font-size: 0.846rem;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-primary); outline: none; box-sizing: border-box;
}
.fb-save-input:focus { border-color: var(--color-accent); }
</style>
