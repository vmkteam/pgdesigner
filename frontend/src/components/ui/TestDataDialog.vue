<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useClipboard } from '@vueuse/core'
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
const seed = ref(0)
const rows = ref(50)

const isOpen = computed(() => ui.activeDialog === 'testdata')

watch(isOpen, (open) => {
  if (open) {
    store.clearTestData()
  }
})

function close() {
  ui.closeDialog()
}

function generate() {
  store.loadTestData(seed.value, rows.value)
}

function copySQL() {
  copy(store.testData)
}

async function saveSQL() {
  if (!store.testData) return
  const path = await appSaveAs(undefined, 'testdata.sql', '.sql')
  if (!path) return
  try {
    await api.project.saveTextFile({ path, content: store.testData })
    showToast('Saved to ' + path.substring(path.lastIndexOf('/') + 1))
  } catch (e) {
    showToast('Save failed: ' + (e instanceof Error ? e.message : e), 'error')
  }
}

const lineCount = computed(() => store.testData ? store.testData.split('\n').length : 0)

const rowOptions = [10, 25, 50, 100, 250, 500, 1000]
</script>

<template>
  <DialogRoot :open="isOpen">
    <DialogOverlay class="dlg-overlay" @click="close" />
    <DialogContent class="dlg-box" @escape-key-down="close">
      <div class="dlg-header">
        <DialogTitle class="text-xs font-semibold">Generate Test Data</DialogTitle>
        <DialogClose class="dlg-close" @click="close">&times;</DialogClose>
      </div>

      <div class="td-settings">
        <label class="td-label">
          <span>Rows per table</span>
          <select v-model.number="rows" class="td-select">
            <option v-for="n in rowOptions" :key="n" :value="n">{{ n }}</option>
          </select>
        </label>
        <label class="td-label">
          <span>Seed</span>
          <input v-model.number="seed" type="number" min="0" class="td-input" placeholder="0 = random" />
        </label>
        <button class="dlg-btn td-gen-btn" :disabled="store.testDataLoading" @click="generate">
          {{ store.testDataLoading ? 'Generating...' : 'Generate' }}
        </button>
      </div>

      <div class="dlg-body">
        <SqlViewer v-if="store.testData" :value="store.testData" />
        <div v-else-if="store.testDataLoading" class="p-4 text-xs" style="color: var(--color-text-muted)">
          Generating test data...
        </div>
        <div v-else class="p-4 text-xs" style="color: var(--color-text-muted)">
          Configure options and click Generate to preview INSERT statements.
        </div>
      </div>

      <div class="dlg-footer">
        <span v-if="store.testData" class="text-xs" style="color: var(--color-text-muted)">{{ lineCount }} lines</span>
        <span v-else />
        <div class="flex gap-1">
          <button class="dlg-btn" :disabled="!store.testData" @click="saveSQL">Save .sql...</button>
          <button class="dlg-btn" :disabled="!store.testData" @click="copySQL">
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

.td-settings {
  display: flex; align-items: center; gap: 0.769rem; padding: 0.462rem 0.615rem;
  background: var(--color-bg-app); border-bottom: 1px solid var(--color-border); flex-shrink: 0;
}
.td-label {
  display: flex; align-items: center; gap: 0.308rem;
  font-size: 0.846rem; color: var(--color-text-secondary);
}
.td-select, .td-input {
  height: 1.538rem; padding: 0 0.308rem; font-size: 0.846rem;
  border: 1px solid var(--color-menu-border); background: var(--color-bg-surface);
  color: var(--color-text-primary);
}
.td-select { width: 5rem; }
.td-input { width: 5.5rem; }
.td-gen-btn { margin-left: auto; font-weight: 600; }
</style>
