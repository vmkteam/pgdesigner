<script setup lang="ts">
import { computed } from 'vue'
import { useEventListener } from '@vueuse/core'
import { useCanvasStore } from '@/stores/canvas'
import { useProjectStore } from '@/stores/project'
import { useUiStore } from '@/stores/ui'
import api from '@/api/factory'
import { confirmUnsaved } from '@/composables/useConfirmUnsaved'
import { showToast } from '@/composables/useToast'
import GoToDialog from '../ui/GoToDialog.vue'

const canvas = useCanvasStore()
const store = useProjectStore()
const ui = useUiStore()

const schemas = computed(() => {
  if (!store.schema?.tables) return []
  const s = new Set(store.schema.tables.map(t => t.schema).filter(Boolean))
  return [...s].sort()
})
async function fileNew() {
  if (!await confirmUnsaved()) return
  try {
    await api.app.newProject()
    await store.loadAll()
    ui.isWelcome = false
    ui.settingsOpen = true
  } catch (e: unknown) {
    showToast('New failed: ' + (e instanceof Error ? e.message : e))
  }
}

function onKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 'f') {
    e.preventDefault()
    ui.goToOpen = true
  }
  if ((e.metaKey || e.ctrlKey) && !e.shiftKey && e.key === 'd') {
    e.preventDefault()
    ui.openDiff()
  }
}

useEventListener(document, 'keydown', onKeydown)
</script>

<template>
  <div class="toolbar">
    <!-- Standard tools -->
    <button class="tb-btn" title="New (Ctrl+N)" @click="fileNew">
      <svg class="tb-icon" viewBox="0 0 16 16"><path d="M4 1h5l4 4v10H4z" fill="none" stroke="currentColor" stroke-width="1.2"/><path d="M9 1v4h4" fill="none" stroke="currentColor" stroke-width="1"/></svg>
    </button>
    <button class="tb-btn" title="Open (Ctrl+O)" @click="ui.openDialogOpen = true">
      <svg class="tb-icon" viewBox="0 0 16 16"><path d="M1 5h3l1-2h6l1 2h3v8H1z" fill="none" stroke="currentColor" stroke-width="1.2"/></svg>
    </button>
    <button class="tb-btn" title="Save (Ctrl+S)" :disabled="!store.info?.filePath" @click="store.saveProject()">
      <svg class="tb-icon" viewBox="0 0 16 16"><path d="M2 1h10l3 3v11H2z" fill="none" stroke="currentColor" stroke-width="1.2"/><path d="M5 1v4h6V1" fill="none" stroke="currentColor" stroke-width="0.8"/><rect x="4" y="9" width="8" height="4" rx="0.5" fill="none" stroke="currentColor" stroke-width="0.8"/></svg>
    </button>
    <div class="tb-sep"></div>

    <!-- Undo/Redo -->
    <button class="tb-btn" title="Undo (Ctrl+Z)">
      <svg class="tb-icon" viewBox="0 0 16 16"><path d="M5 3L2 6l3 3" fill="none" stroke="currentColor" stroke-width="1.3"/><path d="M2 6h8a4 4 0 010 8H7" fill="none" stroke="currentColor" stroke-width="1.2"/></svg>
    </button>
    <button class="tb-btn" title="Redo (Ctrl+Shift+Z)">
      <svg class="tb-icon" viewBox="0 0 16 16"><path d="M11 3l3 3-3 3" fill="none" stroke="currentColor" stroke-width="1.3"/><path d="M14 6H6a4 4 0 000 8h3" fill="none" stroke="currentColor" stroke-width="1.2"/></svg>
    </button>
    <div class="tb-sep"></div>

    <!-- Zoom -->
    <button class="tb-btn" title="Zoom In (F6)" @click="canvas.zoomIn()">
      <svg class="tb-icon" viewBox="0 0 16 16"><circle cx="7" cy="7" r="4.5" fill="none" stroke="currentColor" stroke-width="1.2"/><line x1="10.5" y1="10.5" x2="14" y2="14" stroke="currentColor" stroke-width="1.3"/><line x1="5" y1="7" x2="9" y2="7" stroke="currentColor" stroke-width="1"/><line x1="7" y1="5" x2="7" y2="9" stroke="currentColor" stroke-width="1"/></svg>
    </button>
    <button class="tb-btn" title="Zoom Out (F7)" @click="canvas.zoomOut()">
      <svg class="tb-icon" viewBox="0 0 16 16"><circle cx="7" cy="7" r="4.5" fill="none" stroke="currentColor" stroke-width="1.2"/><line x1="10.5" y1="10.5" x2="14" y2="14" stroke="currentColor" stroke-width="1.3"/><line x1="5" y1="7" x2="9" y2="7" stroke="currentColor" stroke-width="1"/></svg>
    </button>
    <button class="tb-btn" title="Fit to Screen" @click="canvas.fitToScreen()">
      <svg class="tb-icon" viewBox="0 0 16 16"><path d="M1 5V1h4M11 1h4v4M15 11v4h-4M5 15H1v-4" fill="none" stroke="currentColor" stroke-width="1.2"/></svg>
    </button>
    <button class="tb-zoom" title="Reset to 100%" @click="canvas.resetZoom()">{{ canvas.zoom }}%</button>
    <div class="tb-sep"></div>

    <!-- Schema filter (only shown for multi-schema projects) -->
    <template v-if="schemas.length > 1">
      <div class="tb-sep"></div>
      <select
        class="tb-schema"
        :value="canvas.activeSchema ?? ''"
        @change="canvas.activeSchema = ($event.target as HTMLSelectElement).value || null"
      >
        <option value="">All Schemas</option>
        <option v-for="s in schemas" :key="s" :value="s">{{ s }}</option>
      </select>
    </template>

    <!-- Canvas tools -->
    <button
      class="tb-btn tb-tool" :class="{ 'tb-active': canvas.activeTool === 'createTable' }"
      title="Create Table (click on canvas)"
      @click="canvas.setTool(canvas.activeTool === 'createTable' ? 'pointer' : 'createTable')"
    >⊞T</button>
    <button
      class="tb-btn tb-tool" :class="{ 'tb-active': canvas.activeTool === 'createFK' }"
      title="Create FK (click source → target)"
      @click="canvas.setTool(canvas.activeTool === 'createFK' ? 'pointer' : 'createFK')"
    >→FK</button>
    <button
      class="tb-btn tb-tool" :class="{ 'tb-active': canvas.activeTool === 'createM2M' }"
      title="Create M:N (click table A → table B)"
      @click="canvas.setTool(canvas.activeTool === 'createM2M' ? 'pointer' : 'createM2M')"
    >⊞M:N</button>
    <!-- Tool status hint -->
    <span v-if="canvas.activeTool !== 'pointer'" class="tb-hint">
      <template v-if="canvas.activeTool === 'createTable'">Click on canvas to place table</template>
      <template v-else-if="!canvas.toolSourceNode">Click source table</template>
      <template v-else>Click target table (Esc to cancel)</template>
    </span>
    <div class="tb-sep"></div>

    <!-- Find / Go To -->
    <button class="tb-btn" title="Go To (Ctrl+F)" @click="ui.goToOpen = true">
      <svg class="tb-icon" viewBox="0 0 16 16"><circle cx="7" cy="7" r="4.5" fill="none" stroke="currentColor" stroke-width="1.2"/><line x1="10.5" y1="10.5" x2="14" y2="14" stroke="currentColor" stroke-width="1.3"/></svg>
    </button>
  </div>

  <GoToDialog :open="ui.goToOpen" @close="ui.goToOpen = false" />
</template>

<style scoped>
.toolbar {
  height: 2.154rem; background: var(--color-bg-app); border-bottom: 1px solid var(--color-border);
  display: flex; align-items: center; padding: 0 0.308rem; gap: 0.154rem; flex-shrink: 0; user-select: none;
}
.tb-btn {
  width: 1.846rem; height: 1.846rem; display: flex; align-items: center; justify-content: center; font-size: 0.923rem;
}
.tb-btn:hover { background: var(--color-bg-hover); }
.tb-icon { width: 1.077rem; height: 1.077rem; color: var(--color-text-secondary); }
.tb-btn:hover .tb-icon { color: var(--color-text-primary); }
.tb-btn:disabled .tb-icon { opacity: 0.4; }
.tb-zoom {
  font-size: 0.923rem; color: var(--color-text-secondary); margin: 0 0.308rem; width: 3.077rem; text-align: center; height: 1.846rem;
}
.tb-zoom:hover { background: var(--color-bg-hover); }
.tb-sep { width: 1px; height: 1.231rem; background: var(--color-border); margin: 0 0.308rem; }
.tb-tool { font-size: 0.769rem; font-weight: 600; width: auto; padding: 0 0.385rem; }
.tb-active { background: var(--color-bg-selected) !important; outline: 1px solid var(--color-accent); outline-offset: -1px; }
.tb-hint { font-size: 0.769rem; color: var(--color-accent); font-style: italic; padding: 0 0.308rem; }
.tb-schema {
  height: 1.538rem; font-size: 0.846rem; padding: 0 0.308rem;
  border: 1px solid var(--color-menu-border); background: var(--color-bg-surface);
  color: var(--color-text-primary);
}
</style>
