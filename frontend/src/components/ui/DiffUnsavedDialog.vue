<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useClipboard } from '@vueuse/core'
import { DialogRoot, DialogOverlay, DialogContent, DialogTitle, DialogClose } from 'reka-ui'
import { useUiStore } from '@/stores/ui'
import type { IDiffUnsavedResult } from '@/api/factory'
import api from '@/api/factory'
import { appSaveAs } from '@/composables/useSaveDialog'
import { showToast } from '@/composables/useToast'
import SqlViewer from './SqlViewer.vue'

const ui = useUiStore()
const result = ref<IDiffUnsavedResult | null>(null)
const loading = ref(false)
const { copy, copied } = useClipboard({ copiedDuring: 2000 })

const isOpen = computed(() => ui.activeDialog === 'diff')

watch(isOpen, async (open, _old, onCleanup) => {
  if (!open) { result.value = null; return }
  let cancelled = false
  onCleanup(() => { cancelled = true })
  loading.value = true
  try {
    const data = await api.project.diffUnsaved()
    if (!cancelled) result.value = data
  } catch {
    if (!cancelled) result.value = null
  } finally {
    if (!cancelled) loading.value = false
  }
})

function close() { ui.closeDialog() }

const hasChanges = computed(() => (result.value?.changes?.length || 0) > 0)

const hazardCount = computed(() => {
  if (!result.value?.changes) return 0
  let n = 0
  for (const c of result.value.changes) n += (c.hazards?.length || 0)
  return n
})

function copySQL() {
  if (!result.value?.sql) return
  copy(result.value.sql)
}

async function saveSQL() {
  if (!result.value?.sql) return
  const defaultName = new Date().toISOString().slice(0, 10) + '.sql'
  const path = await appSaveAs(undefined, defaultName, '.sql')
  if (!path) return
  try {
    await api.project.saveTextFile({ path, content: result.value.sql })
    showToast('Saved to ' + path.substring(path.lastIndexOf('/') + 1))
  } catch (e) {
    showToast('Save failed: ' + (e instanceof Error ? e.message : e), 'error')
  }
}

const actionClass: Record<string, string> = { add: 'act-add', drop: 'act-drop', alter: 'act-alter' }
const hazardClass: Record<string, string> = { dangerous: 'hz-dangerous', warning: 'hz-warning', info: 'hz-info' }
</script>

<template>
  <DialogRoot :open="isOpen">
    <DialogOverlay class="dlg-overlay" @click="close" />
    <DialogContent class="dlg-box" @escape-key-down="close">
      <div class="dlg-header">
        <DialogTitle class="dlg-title">
          Unsaved Changes
          <span v-if="hasChanges" class="dlg-count">({{ result!.changes.length }} change{{ result!.changes.length > 1 ? 's' : '' }})</span>
          <span v-if="hazardCount > 0" class="dlg-hazard-badge">{{ hazardCount }} hazard{{ hazardCount > 1 ? 's' : '' }}</span>
        </DialogTitle>
        <DialogClose class="dlg-close" @click="close">&times;</DialogClose>
      </div>

      <div class="dlg-body">
        <div v-if="loading" class="dlg-empty">Loading diff...</div>
        <div v-else-if="!hasChanges" class="dlg-empty">No unsaved changes</div>
        <template v-else>
          <!-- Changes list -->
          <div class="df-changes">
            <div v-for="(ch, i) in result!.changes" :key="i" class="df-change">
              <span class="df-action" :class="actionClass[ch.action]">{{ ch.action.toUpperCase() }}</span>
              <span class="df-object">{{ ch.object }}</span>
              <span class="df-name">{{ ch.table ? ch.table + '.' : '' }}{{ ch.name }}</span>
              <template v-if="ch.hazards?.length">
                <span v-for="(h, j) in ch.hazards" :key="j" class="df-hazard" :class="hazardClass[h.level]">{{ h.code }}</span>
              </template>
            </div>
          </div>
          <!-- Full SQL -->
          <div class="df-sql">
            <SqlViewer :value="result!.sql" />
          </div>
        </template>
      </div>

      <div class="dlg-footer">
        <span v-if="hasChanges" class="dlg-info">{{ result!.sql.split('\n').length }} lines</span>
        <span v-else />
        <div class="dlg-actions">
          <button class="dlg-btn" :disabled="!hasChanges" @click="saveSQL">Save .sql...</button>
          <button class="dlg-btn" :disabled="!hasChanges" @click="copySQL">{{ copied ? 'Copied!' : 'Copy SQL' }}</button>
          <button class="dlg-btn" @click="close">Close</button>
        </div>
      </div>
    </DialogContent>
  </DialogRoot>
</template>

<style scoped>
.dlg-overlay { position: fixed; inset: 0; background: rgba(0,0,0,.3); z-index: 40; }
.dlg-box {
  position: fixed; top: 5%; left: 10%; right: 10%; bottom: 5%; z-index: 50;
  background: var(--color-bg-surface); border: 1px solid var(--color-menu-border);
  display: flex; flex-direction: column; box-shadow: 0 4px 12px rgba(0,0,0,.2);
}
.dlg-header {
  height: 2.154rem; background: var(--color-bg-app); border-bottom: 1px solid var(--color-border);
  display: flex; align-items: center; justify-content: space-between; padding: 0 0.615rem; flex-shrink: 0;
  color: var(--color-text-primary); user-select: none;
}
.dlg-title { font-size: 0.923rem; font-weight: 600; display: flex; align-items: center; gap: 0.462rem; }
.dlg-count { font-weight: 400; color: var(--color-text-secondary); }
.dlg-hazard-badge {
  font-size: 0.692rem; font-weight: 600; padding: 0.077rem 0.385rem;
  border-radius: 0.308rem; background: #cc8800; color: white;
}
.dlg-close {
  width: 1.538rem; height: 1.538rem; display: flex; align-items: center; justify-content: center;
  color: var(--color-text-secondary); font-size: 1.077rem;
}
.dlg-close:hover { background: var(--color-bg-hover); }
.dlg-body { flex: 1; min-height: 0; display: flex; flex-direction: column; overflow: hidden; }
.dlg-empty { padding: 1.538rem; text-align: center; color: var(--color-text-muted); font-size: 0.923rem; }
.dlg-footer {
  height: 2.154rem; background: var(--color-bg-app); border-top: 1px solid var(--color-border);
  display: flex; align-items: center; justify-content: space-between; padding: 0 0.615rem; flex-shrink: 0;
}
.dlg-info { font-size: 0.769rem; color: var(--color-text-muted); }
.dlg-actions { display: flex; gap: 0.308rem; }
.dlg-btn {
  padding: 0 0.923rem; height: 1.538rem; font-size: 0.923rem;
  border: 1px solid var(--color-menu-border); background: var(--color-bg-surface);
  color: var(--color-text-primary);
}
.dlg-btn:hover:not(:disabled) { background: var(--color-bg-hover); }
.dlg-btn:disabled { opacity: 0.5; }

.df-changes {
  padding: 0.462rem; display: flex; flex-wrap: wrap; gap: 0.308rem;
  border-bottom: 1px solid var(--color-border); flex-shrink: 0; max-height: 30%; overflow-y: auto;
  user-select: none;
}
.df-change {
  display: inline-flex; align-items: center; gap: 0.308rem;
  padding: 0.154rem 0.462rem; border: 1px solid var(--color-border-subtle);
  border-radius: 0.231rem; font-size: 0.769rem;
}
.df-action {
  font-size: 0.615rem; font-weight: 700; padding: 0.077rem 0.231rem;
  border-radius: 0.154rem; text-transform: uppercase;
}
.act-add { background: #2d7a2d; color: white; }
.act-drop { background: #cc3333; color: white; }
.act-alter { background: #cc8800; color: white; }
.df-object { color: var(--color-text-secondary); }
.df-name { font-weight: 600; color: var(--color-text-primary); }
.df-hazard {
  font-size: 0.615rem; font-weight: 600; padding: 0.077rem 0.231rem; border-radius: 0.154rem;
}
.hz-dangerous { background: rgba(204, 51, 51, 0.15); color: #cc3333; }
.hz-warning { background: rgba(204, 136, 0, 0.15); color: #cc8800; }
.hz-info { background: rgba(128, 128, 128, 0.1); color: var(--color-text-secondary); }

.df-sql { flex: 1; min-height: 0; overflow: auto; }
</style>
