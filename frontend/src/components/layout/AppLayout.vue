<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useEventListener, useIntervalFn, useActiveElement } from '@vueuse/core'
import {
  SplitterGroup,
  SplitterPanel,
  SplitterResizeHandle,
} from 'reka-ui'
import { useProjectStore } from '@/stores/project'
import { useCanvasStore } from '@/stores/canvas'
import { useUiStore } from '@/stores/ui'
import api from '@/api/factory'
import { useFileActions } from '@/composables/useFileActions'
import MenuBar from './MenuBar.vue'
import Toolbar from './Toolbar.vue'
import ObjectTree from '../tree/ObjectTree.vue'
import DiagramCanvas from '../canvas/DiagramCanvas.vue'
import GenerateDDLDialog from '../ui/GenerateDDLDialog.vue'
import TestDataDialog from '../ui/TestDataDialog.vue'
import CheckDiagramDialog from '../ui/CheckDiagramDialog.vue'
import DiffUnsavedDialog from '../ui/DiffUnsavedDialog.vue'
import TableEditorDialog from '../editor/TableEditorDialog.vue'
import KeyboardReferenceDialog from '../ui/KeyboardReferenceDialog.vue'
import AboutDialog from '../ui/AboutDialog.vue'
import ProjectSettingsDialog from '../ui/ProjectSettingsDialog.vue'
import WelcomeScreen from '../ui/WelcomeScreen.vue'
import OpenDialog from '../ui/OpenDialog.vue'
import SaveDialog from '../ui/SaveDialog.vue'
import AppDialog from '../ui/AppDialog.vue'
import ToastContainer from '../ui/ToastContainer.vue'

const store = useProjectStore()
const canvasStore = useCanvasStore()
const ui = useUiStore()
const { fileNew, fileOpen, fileSaveAs, fileClose } = useFileActions()

// isWelcome is now directly controlled by ui.isWelcome
// Set by: initial load (auto), MenuBar (fileNew/fileClose), WelcomeScreen
watch(() => store.info, (info) => {
  if (!info) { ui.isWelcome = true; return }
  if (info.tables > 0 || info.filePath) ui.isWelcome = false
}, { immediate: true })

function onGlobalKeydown(e: KeyboardEvent) {
  // Ctrl+Shift+S — Save As
  if ((e.metaKey || e.ctrlKey) && e.shiftKey && (e.key === 's' || e.key === 'S') && !ui.tableEditorName) {
    e.preventDefault()
    fileSaveAs()
    return
  }
  // Ctrl+S — save project (when Table Editor is not open)
  if ((e.metaKey || e.ctrlKey) && !e.shiftKey && e.key === 's' && !ui.tableEditorName) {
    e.preventDefault()
    store.saveProject()
  }
  // Ctrl+N — New, Ctrl+O — Open
  if ((e.metaKey || e.ctrlKey) && !e.shiftKey && e.key === 'n') {
    e.preventDefault()
    fileNew()
    return
  }
  if ((e.metaKey || e.ctrlKey) && !e.shiftKey && e.key === 'o') {
    e.preventDefault()
    fileOpen()
    return
  }
  // Ctrl+W — Close
  if ((e.metaKey || e.ctrlKey) && !e.shiftKey && e.key === 'w' && !ui.tableEditorName) {
    e.preventDefault()
    fileClose()
    return
  }
  // Zoom: Ctrl+= / Ctrl+- / Ctrl+0 (only when Table Editor is not open)
  if ((e.metaKey || e.ctrlKey) && !ui.tableEditorName) {
    if (e.key === '=' || e.key === '+') { e.preventDefault(); canvasStore.zoomIn(); return }
    if (e.key === '-') { e.preventDefault(); canvasStore.zoomOut(); return }
    if (e.key === '0') { e.preventDefault(); canvasStore.resetZoom(); return }
  }
  // Ctrl+L — Check Diagram
  if ((e.metaKey || e.ctrlKey) && e.key === 'l' && !ui.tableEditorName) {
    e.preventDefault(); store.loadLint(); ui.openLint(); return
  }
  // Ctrl+G — Generate DDL
  if ((e.metaKey || e.ctrlKey) && e.key === 'g' && !ui.tableEditorName) {
    e.preventDefault(); store.loadDDL(); ui.openDDL(); return
  }
  // Ctrl+, — Project Settings
  if ((e.metaKey || e.ctrlKey) && e.key === ',' && !ui.tableEditorName) {
    e.preventDefault(); ui.settingsOpen = true; return
  }
  // Ctrl+Shift+D — Toggle Dark Theme
  if ((e.metaKey || e.ctrlKey) && e.shiftKey && (e.key === 'D' || e.key === 'd')) {
    e.preventDefault(); ui.toggleTheme(); return
  }
  // Escape — reset canvas tool
  if (e.key === 'Escape' && canvasStore.activeTool !== 'pointer' && !ui.tableEditorName) {
    e.preventDefault(); canvasStore.resetTool(); return
  }
  // Canvas tool hotkeys (T/F/M — toggle tools)
  if (!e.metaKey && !e.ctrlKey && !ui.tableEditorName && !ui.isWelcome && !isInputFocused()) {
    if (e.key === 't') { canvasStore.setTool(canvasStore.activeTool === 'createTable' ? 'pointer' : 'createTable'); return }
    if (e.key === 'f') { canvasStore.setTool(canvasStore.activeTool === 'createFK' ? 'pointer' : 'createFK'); return }
    if (e.key === 'm') { canvasStore.setTool(canvasStore.activeTool === 'createM2M' ? 'pointer' : 'createM2M'); return }
  }

  // ? — open keyboard reference (only when not typing in an input)
  if (e.key === '?' && !isInputFocused()) {
    e.preventDefault()
    ui.keyboardRefOpen = true
  }
}

function timeAgo(date: Date): string {
  const s = Math.floor((Date.now() - date.getTime()) / 1000)
  if (s < 5) return 'just now'
  if (s < 60) return `${s}s ago`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m ago`
  return `${Math.floor(m / 60)}h ago`
}

// Refresh timeAgo display
const _tick = ref(0)
useIntervalFn(() => { _tick.value++ }, 10000)

const activeEl = useActiveElement()
function isInputFocused(): boolean {
  const el = activeEl.value
  if (!el) return false
  const tag = el.tagName
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || (el as HTMLElement).isContentEditable
}

function onBeforeUnload(e: BeforeUnloadEvent) {
  // Warn if unsaved changes
  // Check synchronously via store (can't await RPC here)
  // The dirty state is approximate — real check is server-side via IsDirty RPC
  // sendBeacon for quit
  const body = JSON.stringify({ jsonrpc: '2.0', id: 0, method: 'app.quit', params: {} })
  navigator.sendBeacon('/rpc/', body)
  // Show browser confirmation if project has a file (not welcome screen)
  if (store.info?.filePath) {
    e.preventDefault()
  }
}

useEventListener(window, 'beforeunload', onBeforeUnload)
useEventListener(window, 'keydown', onGlobalKeydown)

onMounted(() => {
  // Cancel any pending quit from a previous beforeunload (e.g. Ctrl+R reload).
  api.app.ping().catch(() => {})
  store.loadAll()
  ui.checkForUpdate()
})
</script>

<template>
  <div class="flex flex-col h-screen">
    <MenuBar />
    <Toolbar v-if="!ui.isWelcome" />

    <div v-if="store.loading" class="flex-1 flex items-center justify-center text-sm" style="color: var(--color-text-muted)">Loading schema...</div>
    <div v-else-if="store.error" class="flex-1 flex items-center justify-center text-sm" style="color: #cc3333">Error: {{ store.error }}</div>

    <WelcomeScreen v-else-if="ui.isWelcome" class="flex-1" />

    <SplitterGroup v-else direction="horizontal" class="flex-1 min-h-0">
      <!-- Left: Tree + Minimap -->
      <SplitterPanel :default-size="15" :min-size="8" :max-size="30">
        <ObjectTree />
      </SplitterPanel>

      <SplitterResizeHandle />

      <!-- Center: Vue Flow Canvas -->
      <SplitterPanel :default-size="85">
        <DiagramCanvas />
      </SplitterPanel>
    </SplitterGroup>

    <div v-if="!ui.isWelcome" class="statusbar">
      <div class="sb-left">
        <span class="sb-dot" :class="store.dirty ? 'sb-dirty' : 'sb-clean'" :title="store.dirty ? 'Unsaved changes' : 'All saved'" />
        <span v-if="store.info">{{ store.info.name }}</span>
        <span v-if="store.info" class="sb-muted">PG{{ store.info.pgVersion }}</span>
        <span v-if="store.info?.filePath" class="sb-muted sb-path" :title="store.info.filePath">{{ store.info.filePath }}</span>
      </div>
      <div class="sb-center">
        <span v-if="canvasStore.activeTool !== 'pointer'" class="sb-tool">
          <template v-if="canvasStore.activeTool === 'createTable'">Click on canvas to place table</template>
          <template v-else-if="!canvasStore.toolSourceNode">Click source table</template>
          <template v-else>Click target table · Esc to cancel</template>
        </span>
      </div>
      <div class="sb-right">
        <template v-if="store.info">
          <span class="sb-stat">{{ store.info.tables }}T</span>
          <span class="sb-stat">{{ store.info.references }}FK</span>
          <span class="sb-stat">{{ store.info.indexes }}Ix</span>
          <span class="sb-sep" />
          <span v-if="store.autoSave" class="sb-autosave" title="Auto Save enabled">⟳ Auto</span>
          <span v-if="store.saveStatus === 'saving'" class="sb-saving">Saving...</span>
          <span v-else-if="store.saveStatus === 'saved'" class="sb-saved">Saved ✓</span>
          <span v-else-if="store.dirty" class="sb-modified">Modified *</span>
          <span v-else class="sb-saved">Saved ✓</span>
          <span v-if="store.lastSaved" class="sb-muted sb-time" :data-t="_tick" :title="store.lastSaved.toLocaleString()">{{ timeAgo(store.lastSaved) }}</span>
          <template v-if="ui.updateInfo?.updateAvailable && !ui.updateDismissed">
            <span class="sb-sep" />
            <a :href="ui.updateInfo.releaseURL" target="_blank" rel="noopener noreferrer" class="sb-update" :title="`Update available: ${ui.updateInfo.latestVersion}`" @click="ui.dismissUpdate()">&#8593; {{ ui.updateInfo.latestVersion }}</a>
          </template>
        </template>
      </div>
    </div>
    <div v-else class="statusbar">
      <span class="sb-muted">PgDesigner</span>
      <span v-if="ui.updateInfo?.updateAvailable && !ui.updateDismissed" class="sb-update-welcome">
        <a :href="ui.updateInfo.releaseURL" target="_blank" rel="noopener noreferrer" class="sb-update" :title="`Update available: ${ui.updateInfo.latestVersion}`" @click="ui.dismissUpdate()">&#8593; {{ ui.updateInfo.latestVersion }} available</a>
      </span>
    </div>

    <GenerateDDLDialog />
    <TestDataDialog />
    <CheckDiagramDialog />
    <DiffUnsavedDialog />
    <TableEditorDialog />
    <KeyboardReferenceDialog :open="ui.keyboardRefOpen" @close="ui.keyboardRefOpen = false" />
    <AboutDialog :open="ui.aboutOpen" @close="ui.aboutOpen = false" />
    <ProjectSettingsDialog :open="ui.settingsOpen" @close="ui.settingsOpen = false" />

    <OpenDialog />
    <SaveDialog />
    <AppDialog />
    <ToastContainer />

    <!-- Export overlay -->
    <div v-if="ui.exporting" class="export-overlay">
      <div class="export-spinner"></div>
      <div class="export-text">Exporting diagram...</div>
    </div>
  </div>
</template>

<style scoped>
.statusbar {
  height: 1.538rem; background: var(--color-bg-app); border-top: 1px solid var(--color-border);
  display: flex; align-items: center; padding: 0 0.615rem;
  font-size: 0.769rem; color: var(--color-text-secondary); flex-shrink: 0;
  user-select: none; gap: 0.308rem;
}
.sb-left, .sb-center, .sb-right { display: flex; align-items: center; gap: 0.385rem; }
.sb-left { flex: 1; min-width: 0; }
.sb-center { flex-shrink: 0; }
.sb-right { flex: 1; justify-content: flex-end; min-width: 0; }
.sb-dot { width: 0.538rem; height: 0.538rem; border-radius: 50%; flex-shrink: 0; }
.sb-clean { background: #4a4; }
.sb-dirty { background: #ca3; }
.sb-muted { color: var(--color-text-muted); }
.sb-path { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 15rem; font-size: 0.692rem; }
.sb-stat { font-weight: 600; }
.sb-sep { width: 1px; height: 0.769rem; background: var(--color-border); margin: 0 0.154rem; }
.sb-tool { color: var(--color-accent); font-style: italic; }
.sb-autosave { color: var(--color-text-muted); font-size: 0.692rem; }
.sb-saving { color: var(--color-accent); }
.sb-saved { color: #4a4; }
.sb-modified { color: #ca3; }
.sb-time { font-size: 0.692rem; }
.sb-update {
  color: var(--color-accent); font-weight: 600; text-decoration: none; cursor: pointer;
  border: 1px solid var(--color-accent); padding: 0 0.308rem; border-radius: 2px; font-size: 0.692rem;
}
.sb-update:hover { opacity: 0.8; }
.sb-update-welcome { margin-left: auto; }
.export-overlay {
  position: fixed; inset: 0; z-index: 100;
  background: rgba(0, 0, 0, 0.5);
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 0.769rem;
}
.export-text { color: white; font-size: 1.077rem; font-weight: 600; }
.export-spinner {
  width: 2.462rem; height: 2.462rem; border: 3px solid rgba(255,255,255,0.3);
  border-top-color: white; border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
