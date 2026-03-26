<script setup lang="ts">
import { computed } from 'vue'
import { useClipboard, whenever } from '@vueuse/core'
import { DialogRoot, DialogOverlay, DialogContent, DialogTitle, DialogClose } from 'reka-ui'
import { useProjectStore } from '@/stores/project'
import { useUiStore } from '@/stores/ui'
import api from '@/api/factory'
import { appSaveAs } from '@/composables/useSaveDialog'
import { showToast } from '@/composables/useToast'
import SqlViewer from './SqlViewer.vue'

const store = useProjectStore()
const ui = useUiStore()
const { copy, copied } = useClipboard({ copiedDuring: 2000 })

const isOpen = computed(() => ui.activeDialog === 'ddl')

whenever(isOpen, () => {
  if (!store.ddl) store.loadDDL()
})

function close() {
  ui.closeDialog()
}

function copyDDL() {
  copy(store.ddl)
}

async function saveSQL() {
  if (!store.ddl) return
  const name = store.info?.name || 'schema'
  const fp = store.info?.filePath || ''
  const defaultDir = fp ? fp.substring(0, fp.lastIndexOf('/')) : ''
  const path = await appSaveAs(defaultDir, name + '.sql', '.sql')
  if (!path) return
  try {
    await api.project.saveTextFile({ path, content: store.ddl })
    showToast('Saved to ' + path.substring(path.lastIndexOf('/') + 1))
  } catch (e) {
    showToast('Save failed: ' + (e instanceof Error ? e.message : e), 'error')
  }
}
</script>

<template>
  <DialogRoot :open="isOpen">
    <DialogOverlay class="dlg-overlay" @click="close" />
    <DialogContent class="dlg-box" @escape-key-down="close">
      <div class="dlg-header">
        <DialogTitle class="text-xs font-semibold">Generate Database — DDL Preview</DialogTitle>
        <DialogClose class="dlg-close" @click="close">&times;</DialogClose>
      </div>
      <div class="dlg-body">
        <SqlViewer v-if="store.ddl" :value="store.ddl" />
        <div v-else class="p-4 text-xs" style="color: var(--color-text-muted)">Loading DDL...</div>
      </div>
      <div class="dlg-footer">
        <span v-if="store.ddl" class="text-xs" style="color: var(--color-text-muted)">{{ store.ddl.split('\n').length }} lines</span>
        <span v-else />
        <div class="flex gap-1">
          <button class="dlg-btn" :disabled="!store.ddl" @click="saveSQL">Save .sql...</button>
          <button class="dlg-btn" :disabled="!store.ddl" @click="copyDDL">
            {{ copied ? 'Copied!' : 'Copy' }}
          </button>
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
  color: var(--color-text-primary);
}
.dlg-close {
  width: 1.538rem; height: 1.538rem; display: flex; align-items: center; justify-content: center;
  color: var(--color-text-secondary); font-size: 1.077rem;
}
.dlg-close:hover { background: var(--color-bg-hover); }
.dlg-body { flex: 1; min-height: 0; overflow: auto; }
.dlg-footer {
  height: 2.154rem; background: var(--color-bg-app); border-top: 1px solid var(--color-border);
  display: flex; align-items: center; justify-content: space-between; padding: 0 0.615rem; flex-shrink: 0;
}
.dlg-btn {
  padding: 0 0.923rem; height: 1.538rem; font-size: 0.923rem;
  border: 1px solid var(--color-menu-border); background: var(--color-bg-surface);
  color: var(--color-text-primary);
}
.dlg-btn:hover { background: var(--color-bg-hover); }
</style>
