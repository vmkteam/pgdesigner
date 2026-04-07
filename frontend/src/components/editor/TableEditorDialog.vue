<script setup lang="ts">
import { ref, computed, watch, nextTick, useTemplateRef } from 'vue'
import { useEventListener } from '@vueuse/core'
import { DialogRoot, DialogPortal, DialogOverlay, DialogContent, DialogTitle } from 'reka-ui'
import { useUiStore } from '@/stores/ui'
import { useEditorStore } from '@/stores/editor'
import { useProjectStore } from '@/stores/project'
import type { IColumnDetail, IPKDetail } from '@/api/factory'
import { statusBarHints } from '@/shortcuts'
import { appConfirm } from '@/composables/useAppDialog'
import GeneralTab from './tabs/GeneralTab.vue'
import ColumnGrid from './tabs/ColumnGrid.vue'
import ColumnProperties from './tabs/ColumnProperties.vue'
import ConstraintList from './tabs/ConstraintList.vue'
import IndexList from './tabs/IndexList.vue'
import FKList from './tabs/FKList.vue'
import LintTab from './tabs/LintTab.vue'
import DiffTab from './tabs/DiffTab.vue'
import SqlViewer from '../ui/SqlViewer.vue'

const ui = useUiStore()
const editor = useEditorStore()
const project = useProjectStore()
const activeTab = ref('columns')

// All table names for FK target dropdown
const allTableNames = computed(() => {
  if (!project.schema?.tables) return []
  return project.schema.tables.map(t => t.name)
})
const selectedCol = ref<number | null>(null)
const gridContext = ref('grid')

const tabs = [
  { id: 'general', label: 'General' },
  { id: 'columns', label: 'Columns' },
  { id: 'constraints', label: 'Constraints' },
  { id: 'indexes', label: 'Indexes' },
  { id: 'fk', label: 'Foreign Keys' },
  { id: 'ddl', label: 'DDL' },
  { id: 'diff', label: 'Diff' },
  { id: 'lint', label: 'Lint' },
]

function tabBadge(tabId: string): number {
  if (tabId === 'lint') return editor.lintIssues.length
  if (tabId === 'diff') return editor.diffChanges.length
  return editor.tabErrors(tabId).length
}

watch(() => ui.tableEditorName, async (name) => {
  if (name) {
    selectedCol.value = null
    await editor.openTable(name)
    activeTab.value = ui.tableEditorTab || 'columns'
    // Auto-select column from GoTo
    const itemName = ui.tableEditorFocusItem
    if (itemName && editor.draft && activeTab.value === 'columns') {
      const idx = editor.draft.columns.findIndex(c => c.name === itemName)
      if (idx >= 0) setTimeout(() => { selectedCol.value = idx }, 100)
    }
  } else {
    editor.close()
  }
})

async function onClose() {
  if (editor.isDirty && !await appConfirm('Discard unsaved changes?', 'Unsaved Changes')) return
  ui.closeTableEditor()
}

async function onApply() {
  try {
    await editor.apply()
    project.loadAll()
  } catch { /* serverErrors populated by store */ }
}

async function onSave() {
  try {
    await editor.saveAndClose()
    project.loadAll()
    ui.closeTableEditor()
  } catch { /* serverErrors populated by store */ }
}

const LENGTH_TYPES = ['varchar', 'character varying', 'char', 'character', 'bit', 'varbit', 'bit varying']
const PRECISION_TYPES = ['numeric', 'decimal', 'float', 'time', 'timetz', 'timestamp', 'timestamptz', 'interval',
  'time with time zone', 'timestamp without time zone', 'time without time zone', 'timestamp with time zone']
const COLLATION_TYPES = ['varchar', 'character varying', 'char', 'character', 'text', 'name', 'citext']

function onColumnPropUpdate(index: number, field: string, value: string | number | boolean | null | object) {
  if (!editor.draft) return
  const oldName = editor.draft.columns[index]!.name
  const col = { ...editor.draft.columns[index]!, [field]: value } as IColumnDetail

  // Auto-clear modifiers when type changes
  if (field === 'type') {
    let base = (value as string).toLowerCase()
    if (base.endsWith('[]')) base = base.slice(0, -2)
    if (!LENGTH_TYPES.includes(base)) col.length = 0
    if (!PRECISION_TYPES.includes(base)) { col.precision = 0; col.scale = 0 }
    if (!COLLATION_TYPES.includes(base)) col.collation = ''
  }

  editor.draft.columns = [
    ...editor.draft.columns.slice(0, index),
    col,
    ...editor.draft.columns.slice(index + 1),
  ]

  // Update references when column is renamed
  if (field === 'name' && oldName && value && oldName !== value) {
    const newName = value as string
    const rename = (arr: string[]) => arr.map(c => c === oldName ? newName : c)
    if (editor.draft.pk) editor.draft.pk.columns = rename(editor.draft.pk.columns)
    if (editor.draft.uniques) {
      for (const u of editor.draft.uniques) u.columns = rename(u.columns)
    }
    if (editor.draft.indexes) {
      for (const ix of editor.draft.indexes) {
        ix.columns = (ix.columns || []).map(c => c.name === oldName ? { ...c, name: newName } : c)
      }
    }
    if (editor.draft.fks) {
      for (const fk of editor.draft.fks) {
        fk.columns = fk.columns.map(c => c.name === oldName ? { ...c, name: newName } : c)
      }
    }
  }
}

const gridRef = useTemplateRef<InstanceType<typeof ColumnGrid>>('gridRef')

function onAddColumn() {
  if (!editor.draft) return
  // pk, fk are display-only (derived from constraints), included for IColumnDetail type compatibility
  const defaultNullable = project.info?.defaultNullable ?? true
  editor.draft.columns.push({ name: '', type: 'text', length: 0, precision: 0, scale: 0, nullable: defaultNullable, default: '', pk: false, fk: false, identity: '', generated: '', generatedStored: false, comment: '', compression: '', storage: '', collation: '' })
  const newIdx = editor.draft.columns.length - 1
  selectedCol.value = newIdx
  nextTick(() => {
    gridRef.value?.editName(newIdx)
  })
}

function onDeleteColumn(idx: number) {
  if (!editor.draft) return
  const colName = editor.draft.columns[idx]?.name
  editor.draft.columns.splice(idx, 1)

  // Clean up references to deleted column
  if (colName) {
    // PK
    if (editor.draft.pk) {
      editor.draft.pk.columns = editor.draft.pk.columns.filter(c => c !== colName)
      if (editor.draft.pk.columns.length === 0) editor.draft.pk = undefined!
    }
    // Update column pk flags
    const pkCols = new Set(editor.draft.pk?.columns || [])
    editor.draft.columns = editor.draft.columns.map(c => ({ ...c, pk: pkCols.has(c.name) }))
    // UNIQUE
    if (editor.draft.uniques) {
      for (const u of editor.draft.uniques) u.columns = u.columns.filter(c => c !== colName)
    }
    // Indexes
    if (editor.draft.indexes) {
      for (const ix of editor.draft.indexes) ix.columns = (ix.columns || []).filter(c => c.name !== colName)
    }
    // FK column mappings
    if (editor.draft.fks) {
      for (const fk of editor.draft.fks) fk.columns = fk.columns.filter(c => c.name !== colName)
    }
  }

  if (selectedCol.value !== null && selectedCol.value >= editor.draft.columns.length) {
    selectedCol.value = editor.draft.columns.length > 0 ? editor.draft.columns.length - 1 : null
  }
}

function onMoveColumn(idx: number, direction: number) {
  if (!editor.draft) return
  const newIdx = idx + direction
  if (newIdx < 0 || newIdx >= editor.draft.columns.length) return
  const cols = [...editor.draft.columns]
  ;[cols[idx], cols[newIdx]] = [cols[newIdx]!, cols[idx]!]
  editor.draft.columns = cols
  selectedCol.value = newIdx
}

function onTogglePK(columnName: string) {
  if (!editor.draft) return

  const pkCols = editor.draft.pk?.columns || []
  const idx = pkCols.indexOf(columnName)
  let newPkCols: string[]
  if (idx >= 0) {
    newPkCols = pkCols.filter(c => c !== columnName)
  } else {
    newPkCols = [...pkCols, columnName]
  }

  if (newPkCols.length === 0) {
    editor.draft.pk = undefined!
  } else {
    const tableName = editor.tableName?.replace(/.*\./, '') || 'table'
    editor.draft.pk = {
      name: editor.draft.pk?.name || 'pk_' + tableName,
      columns: newPkCols,
    }
  }

  editor.draft.columns = editor.draft.columns.map(col => ({
    ...col,
    pk: newPkCols.includes(col.name),
  }))
}

function onGoToIndex(indexName: string) {
  activeTab.value = 'indexes'
  ui.tableEditorFocusItem = indexName
}

function onCreateIndex(columnName: string) {
  if (!editor.draft) return
  const tableName = editor.draft.name.replace(/.*\./, '')
  const name = `ix_${tableName}_${columnName}`
  if (editor.draft.indexes?.some(ix => ix.name === name)) {
    onGoToIndex(name)
    return
  }
  const newIndex = {
    name, unique: false, nullsDistinct: false, using: 'btree',
    columns: [{ name: columnName, order: '', nulls: '', opclass: '' }], expressions: [], with: [], where: '', include: [],
  }
  editor.draft.indexes = [...(editor.draft.indexes || []), newIndex]
  activeTab.value = 'indexes'
  ui.tableEditorFocusItem = name
}

function onConstraintPKUpdate(pk: IPKDetail | null) {
  if (!editor.draft) return
  editor.draft.pk = pk ?? undefined
  const pkCols = pk?.columns || []
  editor.draft.columns = editor.draft.columns.map(col => ({
    ...col,
    pk: pkCols.includes(col.name),
  }))
}

// StatusBar hint text
const statusHint = ref('')

function onContextChange(ctx: string) {
  gridContext.value = ctx
  statusHint.value = statusBarHints(ctx)
}

function onEscapeKey(e: Event) {
  // If grid is editing, let the grid handle Escape — don't close dialog
  if (gridContext.value === 'grid-edit') {
    e.preventDefault()
    return
  }
  onClose()
}

function onKeydown(e: KeyboardEvent) {
  if (!editor.isOpen) return

  // Ctrl+S — save
  if ((e.metaKey || e.ctrlKey) && e.key === 's') {
    e.preventDefault(); onSave()
    return
  }

  // Ctrl+Enter — apply
  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
    e.preventDefault(); onApply()
    return
  }

  // Ctrl+1..6 — switch tabs
  if ((e.metaKey || e.ctrlKey) && e.key >= '1' && e.key <= '6') {
    const idx = parseInt(e.key) - 1
    if (idx < tabs.length) {
      e.preventDefault()
      activeTab.value = tabs[idx]!.id
    }
    return
  }
}

useEventListener(document, 'keydown', onKeydown)
</script>

<template>
  <DialogRoot :open="editor.isOpen">
    <DialogPortal>
      <DialogOverlay class="dlg-overlay" @click="onClose" />
      <DialogContent class="dlg-box" @escape-key-down="onEscapeKey">
        <div class="dlg-header">
          <DialogTitle class="dlg-title">
            Table Editor: {{ editor.tableName }}{{ editor.isDirty ? ' *' : '' }}
          </DialogTitle>
          <button class="dlg-close" @click="onClose">✕</button>
        </div>

        <div class="dlg-tabs">
          <button
            v-for="(tab, idx) in tabs" :key="tab.id"
            class="dlg-tab" :class="activeTab === tab.id ? 'active' : ''"
            :title="`Ctrl+${idx + 1}`"
            @click="activeTab = tab.id; tab.id === 'lint' && editor.loadLint(); tab.id === 'diff' && editor.loadDiff()"
          >{{ tab.label }}<span v-if="tabBadge(tab.id) > 0" class="tab-badge" :class="{ 'tab-badge-error': tab.id !== 'lint' && tab.id !== 'diff' }">{{ tabBadge(tab.id) }}</span></button>
        </div>

        <div class="dlg-body">
          <div v-if="editor.loading" class="loading-msg">Loading...</div>
          <template v-else-if="editor.draft">
            <GeneralTab v-if="activeTab === 'general'" />
            <div v-else-if="activeTab === 'columns'" class="columns-split">
              <div class="columns-grid">
                <ColumnGrid
                  ref="gridRef"
                  :columns="editor.draft.columns"
                  :selected="selectedCol"
                  @select="selectedCol = $event"
                  @update="onColumnPropUpdate"
                  @toggle-p-k="onTogglePK"
                  @add="onAddColumn"
                  @delete="onDeleteColumn"
                  @move-up="onMoveColumn($event, -1)"
                  @move-down="onMoveColumn($event, 1)"
                  @context-change="onContextChange"
                />
              </div>
              <div v-if="selectedCol != null && editor.draft.columns[selectedCol]" class="columns-props">
                <ColumnProperties
                  :column="editor.draft.columns[selectedCol]!"
                  :index="selectedCol"
                  :indexes="editor.draft.indexes || []"
                  :table-name="editor.draft.name.replace(/.*\./, '')"
                  @update="onColumnPropUpdate"
                  @toggle-p-k="onTogglePK"
                  @go-to-index="onGoToIndex"
                  @create-index="onCreateIndex"
                />
              </div>
            </div>
            <ConstraintList
              v-else-if="activeTab === 'constraints'"
              :pk="editor.draft.pk"
              :uniques="editor.draft.uniques"
              :checks="editor.draft.checks"
              :excludes="editor.draft.excludes"
              :columns="editor.draft.columns.map(c => c.name)"
              :table-name="editor.draft.name.replace(/.*\./, '')"
              :focus-item="ui.tableEditorFocusItem"
              @update-p-k="onConstraintPKUpdate"
              @update-uniques="v => editor.draft!.uniques = v"
              @update-checks="v => editor.draft!.checks = v"
              @update-excludes="v => editor.draft!.excludes = v"
              @context-change="onContextChange"
            />
            <IndexList
              v-else-if="activeTab === 'indexes'"
              :indexes="editor.draft.indexes"
              :columns="editor.draft.columns.map(c => c.name)"
              :table-name="editor.draft.name.replace(/.*\./, '')"
              :focus-item="ui.tableEditorFocusItem"
              @update-indexes="v => editor.draft!.indexes = v"
              @context-change="onContextChange"
            />
            <FKList
              v-else-if="activeTab === 'fk'"
              :fks="editor.draft.fks"
              :columns="editor.draft.columns.map(c => c.name)"
              :tables="allTableNames"
              :table-name="editor.draft.name.replace(/.*\./, '')"
              :default-on-delete="project.settings?.defaultOnDelete"
              :default-on-update="project.settings?.defaultOnUpdate"
              :focus-f-k="ui.tableEditorFocusFK"
              @update-f-ks="v => editor.draft!.fks = v"
              @context-change="onContextChange"
            />
            <SqlViewer v-else-if="activeTab === 'ddl'" :value="editor.draft.ddl || '-- DDL not available'" />
            <DiffTab v-else-if="activeTab === 'diff'" />
            <LintTab v-else-if="activeTab === 'lint'" />
          </template>
        </div>

        <div v-if="editor.serverErrors.length" class="dlg-errors">
          <span v-for="(err, i) in editor.serverErrors" :key="i" class="dlg-error-msg">{{ err }}</span>
          <button class="dlg-error-close" @click="editor.clearServerErrors()">x</button>
        </div>

        <div class="dlg-footer">
          <div class="footer-left">
            <span class="footer-hints">{{ statusHint }}</span>
          </div>
          <div class="footer-right">
            <button class="dlg-btn primary" :title="editor.hasErrors ? `${editor.errors.length} validation error(s) — fix before saving` : 'Save (Ctrl+S)'" :disabled="!editor.isDirty || editor.saving || editor.hasErrors" @click="onSave">Save</button>
            <button class="dlg-btn" :title="editor.hasErrors ? `${editor.errors.length} validation error(s) — fix before saving` : 'Apply (Ctrl+Enter)'" :disabled="!editor.isDirty || editor.saving || editor.hasErrors" @click="onApply">
              {{ editor.saving ? 'Applying...' : 'Apply' }}
            </button>
            <button class="dlg-btn" :disabled="!editor.isDirty" @click="editor.revert()">Revert</button>
            <button class="dlg-btn" title="Close (Escape)" @click="onClose">Cancel</button>
          </div>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<style>
.dlg-overlay { position: fixed; inset: 0; background: rgba(0,0,0,.2); z-index: 40; }
.dlg-box {
  position: fixed; z-index: 50; top: 10%; left: 15%; width: 70%; height: 75%;
  min-width: 46.154rem; min-height: 30.769rem;
  background: var(--color-bg-surface); border: 1px solid var(--color-menu-border);
  display: flex; flex-direction: column; box-shadow: 0 4px 12px rgba(0,0,0,.2);
}
.dlg-header {
  height: 2.154rem; background: var(--color-bg-app); border-bottom: 1px solid var(--color-border);
  display: flex; align-items: center; padding: 0 0.923rem; flex-shrink: 0; user-select: none;
  color: var(--color-text-primary);
}
.dlg-title { font-size: 0.923rem; font-weight: 600; flex: 1; }
.dlg-close {
  width: 1.538rem; height: 1.538rem; display: flex; align-items: center; justify-content: center;
  color: var(--color-text-secondary); font-size: 1.077rem;
}
.dlg-close:hover { background: var(--color-bg-hover); }
.dlg-tabs {
  height: 1.846rem; background: var(--color-bg-app); border-bottom: 1px solid var(--color-border);
  display: flex; align-items: center; flex-shrink: 0; user-select: none;
}
.dlg-tab {
  padding: 0 0.923rem; height: 100%; font-size: 0.923rem; border-right: 1px solid var(--color-border);
  color: var(--color-text-primary);
}
.dlg-tab:hover { background: var(--color-bg-hover); }
.dlg-tab.active { background: var(--color-bg-surface); font-weight: 600; }
.dlg-body { flex: 1; overflow: auto; }
.loading-msg {
  display: flex; align-items: center; justify-content: center; height: 100%;
  font-size: 1rem; color: var(--color-text-muted);
}
.dlg-footer {
  height: 2.462rem; background: var(--color-bg-app); border-top: 1px solid var(--color-border);
  display: flex; align-items: center; justify-content: space-between; padding: 0 0.923rem; flex-shrink: 0;
}
.footer-left { display: flex; flex: 1; min-width: 0; }
.footer-right { display: flex; gap: 0.308rem; flex-shrink: 0; }
.footer-hints {
  font-size: 0.769rem; color: var(--color-text-muted);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.dlg-btn {
  padding: 0 0.769rem; height: 1.692rem; font-size: 0.923rem;
  border: 1px solid var(--color-menu-border); background: var(--color-bg-surface);
  color: var(--color-text-primary); cursor: default;
}
.dlg-btn:hover:not(:disabled) { background: var(--color-bg-hover); }
.dlg-btn:disabled { opacity: 0.5; }
.dlg-btn.primary { font-weight: 600; }

.columns-split { display: flex; height: 100%; }
.columns-grid { flex: 1; overflow: auto; min-width: 0; border-right: 1px solid var(--color-border); }
.columns-props { width: 21.538rem; flex-shrink: 0; overflow-y: auto; }

.tab-badge {
  margin-left: 0.308rem; padding: 0 0.308rem; font-size: 0.692rem; font-weight: 600;
  border-radius: 0.462rem; background: var(--color-text-muted); color: var(--color-bg-surface);
  vertical-align: middle;
}
.tab-badge-error { background: #cc3333; }

.dlg-errors {
  padding: 0.308rem 0.923rem; background: #fff0f0; border-top: 1px solid #cc3333;
  display: flex; align-items: center; gap: 0.462rem; flex-shrink: 0; flex-wrap: wrap;
}
.dlg-error-msg { font-size: 0.846rem; color: #cc3333; }
.dlg-error-close {
  margin-left: auto; font-size: 0.846rem; color: #cc3333; cursor: pointer;
  background: none; border: none; padding: 0 0.308rem;
}
</style>
