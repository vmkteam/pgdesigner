<script setup lang="ts">
import { ref, computed, nextTick, useTemplateRef } from 'vue'
import { useClipboard } from '@vueuse/core'
import { useProjectStore } from '@/stores/project'
import { useCanvasStore } from '@/stores/canvas'
import { useUiStore } from '@/stores/ui'
import api from '@/api/factory'
import type { IProjectUpdateTableParams } from '@/api/factory'
import { appConfirm, appPrompt } from '@/composables/useAppDialog'
import { showToast } from '@/composables/useToast'
import { createTableWithPK } from '@/composables/useCreateTable'
import { identifierError } from '@/composables/useIdentifierValidation'
import TreeContextMenu, { type ContextMenuItem } from './TreeContextMenu.vue'

const store = useProjectStore()
const canvasStore = useCanvasStore()
const ui = useUiStore()

const allTables = computed(() => store.schema?.tables || [])
const references = computed(() => [...(store.schema?.references || [])].sort((a, b) => a.name.localeCompare(b.name)))

// Group tables by schema, sorted within each group (include empty schemas from info)
const schemas = computed(() => {
  const map = new Map<string, typeof allTables.value>()
  // Seed with all known schemas (including empty ones)
  for (const s of store.info?.schemas || []) {
    map.set(s, [])
  }
  for (const t of allTables.value) {
    const s = t.schema || 'public'
    if (!map.has(s)) map.set(s, [])
    map.get(s)!.push(t)
  }
  return [...map.entries()]
    .sort((a, b) => a[0].localeCompare(b[0]))
    .map(([name, tables]) => [name, [...tables].sort((a, b) => shortName(a.name).localeCompare(shortName(b.name)))] as const)
})
const hasMultipleSchemas = computed(() => schemas.value.length > 1)

function shortName(name: string) {
  const dot = name.indexOf('.')
  return dot >= 0 ? name.substring(dot + 1) : name
}

// --- Selected + Keyboard ---
const selectedTable = ref<string | null>(null)

function selectTable(name: string) {
  selectedTable.value = name
  canvasStore.focusNode(name)
}

function onTableKeydown(e: KeyboardEvent, tableName: string) {
  if (e.key === 'F2') {
    e.preventDefault()
    startRename(tableName)
  } else if (e.key === 'Enter') {
    e.preventDefault()
    ui.openTableEditor(tableName)
  } else if (e.key === 'Delete' || e.key === 'Backspace') {
    e.preventDefault()
    deleteTable(tableName)
  }
}

// --- Create / Delete Table ---
async function createTable(schemaName: string) {
  const name = await appPrompt('New table name:', 'Create Table')
  if (!name) return
  try {
    const defaultSchema = store.info?.schemas?.[0] || 'public'
    await createTableWithPK(schemaName, name, defaultSchema)
    await store.loadAll()
  } catch (e: unknown) {
    showToast('Create failed: ' + (e instanceof Error ? e.message : e))
  }
}

async function deleteTable(name: string) {
  if (!await appConfirm(`Delete table "${name}"?`, 'Delete Table')) return
  try {
    await api.project.deleteTable({ name })
    if (selectedTable.value === name) selectedTable.value = null
    await store.loadAll()
  } catch (e: unknown) {
    showToast('Delete failed: ' + (e instanceof Error ? e.message : e))
  }
}

// --- Create / Delete Schema ---
async function createSchema() {
  const name = await appPrompt('New schema name:', 'Create Schema')
  if (!name) return
  try {
    await api.project.createSchema({ name: name })
    await store.loadAll()
  } catch (e: unknown) {
    showToast('Create schema failed: ' + (e instanceof Error ? e.message : e))
  }
}

async function deleteSchema(name: string) {
  if (!await appConfirm(`Delete schema "${name}"? (must be empty)`, 'Delete Schema')) return
  try {
    await api.project.deleteSchema({ name })
    await store.loadAll()
  } catch (e: unknown) {
    showToast('Delete schema failed: ' + (e instanceof Error ? e.message : e))
  }
}

// --- Context Menus ---
const ctxMenuRef = useTemplateRef<InstanceType<typeof TreeContextMenu>>('ctxMenuRef')
const { copy } = useClipboard()

// Collapse/expand state for schemas
const collapsedSchemas = ref(new Set<string>())

function toggleSchemaCollapse(name: string) {
  const s = new Set(collapsedSchemas.value)
  if (s.has(name)) s.delete(name); else s.add(name)
  collapsedSchemas.value = s
}

function collapseAll() {
  collapsedSchemas.value = new Set(schemas.value.map(([name]) => name))
}

function expandAll() {
  collapsedSchemas.value = new Set()
}

function showTableContextMenu(e: MouseEvent, tableName: string, schemaName: string) {
  const items: ContextMenuItem[] = [
    { label: 'Open Editor', shortcut: 'Enter', action: () => ui.openTableEditor(tableName) },
    { separator: true },
    { label: 'Rename...', shortcut: 'F2', action: () => startRename(tableName) },
    { separator: true },
  ]

  // Move to submenu
  const others = (store.info?.schemas || []).filter(s => s !== schemaName)
  if (others.length > 0) {
    items.push({
      label: 'Move to',
      children: others.map(s => ({ label: s, action: () => moveTableTo(tableName, s) })),
    })
    items.push({ separator: true })
  }

  items.push({ label: 'Copy DDL', action: () => copyTableDDL(tableName) })
  items.push({ separator: true })
  items.push({ label: 'Delete', shortcut: 'Del', action: () => deleteTable(tableName) })

  ctxMenuRef.value?.show(e, items)
}

function showSchemaContextMenu(e: MouseEvent, schemaName: string, tableCount: number) {
  const items: ContextMenuItem[] = [
    { label: 'Add Table...', action: () => createTable(schemaName) },
    { separator: true },
    { label: 'Delete Schema', action: () => deleteSchema(schemaName), disabled: tableCount > 0 },
  ]
  ctxMenuRef.value?.show(e, items)
}

function showRefContextMenu(e: MouseEvent, refName: string, fromTable: string, toTable: string) {
  const items: ContextMenuItem[] = [
    { label: 'Go to Source Table', action: () => { selectTable(fromTable); canvasStore.focusNode(fromTable) } },
    { label: 'Go to Target Table', action: () => { selectTable(toTable); canvasStore.focusNode(toTable) } },
    { separator: true },
    { label: 'Open in Table Editor', action: () => ui.openTableEditor(fromTable, 'fk', refName) },
  ]
  ctxMenuRef.value?.show(e, items)
}

function showBackgroundContextMenu(e: MouseEvent) {
  const defaultSchema = schemas.value[0]?.[0] || 'public'
  const items: ContextMenuItem[] = [
    { label: 'New Table...', action: () => createTable(defaultSchema) },
    { label: 'New Schema...', action: () => createSchema() },
  ]
  if (hasMultipleSchemas.value) {
    items.push({ separator: true })
    items.push({ label: 'Collapse All', action: collapseAll })
    items.push({ label: 'Expand All', action: expandAll })
  }
  ctxMenuRef.value?.show(e, items)
}

async function moveTableTo(tableName: string, toSchema: string) {
  try {
    await api.project.moveTable({ name: tableName, toSchema })
    await store.loadAll()
  } catch (e: unknown) {
    showToast('Move failed: ' + (e instanceof Error ? e.message : e))
  }
}

async function copyTableDDL(tableName: string) {
  try {
    const ddl = await api.project.getTableDDL({ name: tableName })
    copy(ddl)
    showToast('DDL copied')
  } catch (e: unknown) {
    showToast('Copy DDL failed: ' + (e instanceof Error ? e.message : e))
  }
}

// --- Inline Rename ---
const renaming = ref<string | null>(null)
const renameValue = ref('')

function tableShortName(name: string): string {
  const dot = name.indexOf('.')
  return dot >= 0 ? name.slice(dot + 1) : name
}

function startRename(name: string) {
  renaming.value = name
  renameValue.value = tableShortName(name)
  nextTick(() => {
    const input = document.querySelector('.tree-rename-input') as HTMLInputElement
    if (input) { input.focus(); input.select() }
  })
}

let renamingJustFinished = false

const renameError = ref<string | null>(null)

async function commitRename() {
  const oldName = renaming.value
  const newName = renameValue.value.trim()
  if (!oldName || !newName || newName === tableShortName(oldName)) { renaming.value = null; renameError.value = null; return }
  const err = identifierError(newName)
  if (err) { renameError.value = err; return }
  renaming.value = null
  renameError.value = null
  // Prevent dblclick from opening editor right after rename blur
  renamingJustFinished = true
  setTimeout(() => { renamingJustFinished = false }, 300)
  try {
    await api.project.updateTable({ name: oldName, general: { name: newName } } as unknown as IProjectUpdateTableParams)
    store.loadAll()
  } catch (e: unknown) {
    console.error('Rename failed:', e instanceof Error ? e.message : e)
  }
}

function cancelRename() {
  renaming.value = null
  renameError.value = null
}

function onRenameKeydown(e: KeyboardEvent) {
  e.stopPropagation() // prevent parent onTableKeydown from catching Enter/Escape
  if (e.key === 'Enter') { e.preventDefault(); commitRename() }
  else if (e.key === 'Escape') { cancelRename() }
}

</script>

<template>
  <div class="tree-panel">
    <!-- Header -->
    <div class="tree-section-header">
      Object Tree View
    </div>

    <!-- Tree content -->
    <div class="tree-content" @contextmenu.prevent.self="showBackgroundContextMenu($event)">
      <!-- Database -->
      <div class="tree-row font-semibold">
        <svg class="tree-icon" viewBox="0 0 14 14"><rect x="1" y="3" width="12" height="9" rx="1" fill="#8899bb" stroke="#556688" stroke-width="0.8"/><rect x="3" y="1" width="8" height="4" rx="0.5" fill="#aabbdd" stroke="#556688" stroke-width="0.5"/></svg>
        {{ store.info?.name || 'Database' }}
      </div>

      <div class="pl-3">
        <!-- Tables grouped by schema -->
        <div class="tree-row font-semibold tree-group-label tree-schema-header">
          <svg class="tree-icon" viewBox="0 0 14 14"><rect x="1" y="1" width="12" height="11" fill="#e8d870" stroke="#886600" stroke-width="0.8"/><line x1="1" y1="5" x2="13" y2="5" stroke="#886600" stroke-width="0.6"/><line x1="5" y1="5" x2="5" y2="12" stroke="#886600" stroke-width="0.4"/></svg>
          Tables ({{ allTables.length }})
          <button v-if="!hasMultipleSchemas && schemas.length" class="tree-action-btn" title="Add table" @click.stop="createTable(schemas[0]![0])">+</button>
          <button class="tree-action-btn" title="Add schema" @click.stop="createSchema()">S+</button>
        </div>

        <template v-for="[schemaName, schemaTables] in schemas" :key="schemaName">
          <!-- Schema header (only if multiple schemas) -->
          <div
            v-if="hasMultipleSchemas"
            class="tree-row font-semibold tree-group-label pl-3 mt-0.5 tree-schema-header"
            @click="toggleSchemaCollapse(schemaName)"
            @contextmenu.prevent="showSchemaContextMenu($event, schemaName, schemaTables.length)"
          >
            <span class="tree-collapse">{{ collapsedSchemas.has(schemaName) ? '&#9656;' : '&#9662;' }}</span>
            <svg class="tree-icon" viewBox="0 0 14 14"><rect x="1" y="2" width="12" height="10" rx="0.5" fill="#7799bb" stroke="#556688" stroke-width="0.8"/><line x1="4" y1="5" x2="10" y2="5" stroke="#fff" stroke-width="0.6"/><line x1="4" y1="8" x2="10" y2="8" stroke="#fff" stroke-width="0.6"/></svg>
            {{ schemaName }} ({{ schemaTables.length }})
            <button class="tree-action-btn" title="Add table" @click.stop="createTable(schemaName)">+</button>
            <button v-if="schemaTables.length === 0" class="tree-delete-btn" title="Delete schema" @click.stop="deleteSchema(schemaName)">&times;</button>
          </div>

          <div v-show="!collapsedSchemas.has(schemaName)" :class="hasMultipleSchemas ? 'pl-6' : 'pl-3'">
            <div
              v-for="table in schemaTables"
              :key="table.name"
              class="tree-row tree-item"
              :class="{ 'tree-selected': selectedTable === table.name }"
              tabindex="0"
              @click="selectTable(table.name)"
              @dblclick.prevent="!renamingJustFinished && ui.openTableEditor(table.name)"
              @contextmenu.prevent="showTableContextMenu($event, table.name, schemaName)"
              @keydown="onTableKeydown($event, table.name)"
            >
              <svg class="tree-icon" viewBox="0 0 14 14"><rect x="1" y="1" width="12" height="11" fill="#e8d870" stroke="#886600" stroke-width="0.8"/><line x1="1" y1="5" x2="13" y2="5" stroke="#886600" stroke-width="0.6"/><line x1="5" y1="5" x2="5" y2="12" stroke="#886600" stroke-width="0.4"/></svg>
              <span v-if="renaming === table.name" class="tree-rename-wrap">
                <input
                  v-model="renameValue"
                  class="tree-rename-input"
                  :class="{ 'tree-rename-error': renameError }"
                  maxlength="63"
                  :title="renameError || ''"
                  @blur="commitRename"
                  @keydown="onRenameKeydown"
                />
              </span>
              <template v-else>
                <span class="truncate">{{ table.name }}</span>
                <span v-if="table.partitioned" class="tree-badge">⊞</span>
                <span class="tree-count">{{ table.columns.length }}c<template v-if="table.partitionCount"> · {{ table.partitionCount }}p</template></span>
                <button class="tree-delete-btn" title="Delete table" @click.stop="deleteTable(table.name)">&times;</button>
              </template>
            </div>
          </div>
        </template>

        <!-- References -->
        <div class="tree-row font-semibold tree-group-label mt-1">
          <svg class="tree-icon" viewBox="0 0 14 14"><path d="M3 11 L11 3" stroke="#666" stroke-width="1.5" fill="none"/><path d="M8 3 L11 3 L11 6" stroke="#666" stroke-width="1.2" fill="none"/></svg>
          References ({{ references.length }})
        </div>
        <div class="pl-3">
          <div
            v-for="ref in references"
            :key="ref.name"
            class="tree-row tree-item tree-ref"
            @click="canvasStore.focusNode(ref.from)"
            @contextmenu.prevent="showRefContextMenu($event, ref.name, ref.from, ref.to)"
          >
            <svg class="tree-icon" viewBox="0 0 14 14"><path d="M3 11 L11 3" stroke="#999" stroke-width="1" fill="none"/><path d="M8 3 L11 3 L11 6" stroke="#999" stroke-width="0.8" fill="none"/></svg>
            <span class="truncate">{{ ref.name }}</span>
          </div>
        </div>
      </div>
    </div>

    <TreeContextMenu ref="ctxMenuRef" />
  </div>
</template>

<style scoped>
.tree-panel {
  height: 100%; display: flex; flex-direction: column;
  background: var(--color-bg-surface); border-right: 1px solid var(--color-border);
}
.tree-section-header {
  height: 1.538rem; background: var(--color-bg-app); border-bottom: 1px solid var(--color-border);
  display: flex; align-items: center; padding: 0 0.615rem;
  font-size: 0.846rem; font-weight: 600; flex-shrink: 0;
  color: var(--color-text-primary);
}
.tree-content {
  flex: 1; overflow: auto; padding: 0.308rem; font-size: 0.923rem; user-select: none;
  color: var(--color-text-primary);
}
.tree-row {
  display: flex; align-items: center; gap: 0.308rem;
  padding: 1px 0.308rem; line-height: 1.385rem; white-space: nowrap; min-width: 0;
}
.tree-icon { width: 1.077rem; height: 1.077rem; flex-shrink: 0; }
.tree-group-label { color: var(--color-text-secondary); }
.tree-item { cursor: pointer; outline: none; }
.tree-item:hover { background: var(--color-bg-hover); }
.tree-item:focus { background: var(--color-bg-hover); }
.tree-selected { background: var(--color-bg-selected) !important; }
.tree-count { color: var(--color-text-muted); margin-left: auto; flex-shrink: 0; }
.tree-ref { color: var(--color-text-secondary); }
.tree-rename-wrap { flex: 1; min-width: 0; }
.tree-rename-input {
  width: 100%; padding: 0 0.231rem; font-size: 0.923rem;
  border: 1px solid var(--color-accent); background: var(--color-bg-surface);
  color: var(--color-text-primary); outline: none; line-height: 1.231rem;
}
.tree-rename-error { border-color: #cc3333 !important; }
.tree-schema-header { position: relative; }
.tree-action-btn {
  display: none; border: none; background: none; cursor: pointer;
  color: var(--color-text-muted); font-size: 1rem; line-height: 1; padding: 0 0.231rem;
  flex-shrink: 0;
}
.tree-action-btn:first-of-type { margin-left: auto; }
.tree-action-btn:hover { color: var(--color-accent); }
.tree-schema-header:hover .tree-action-btn { display: inline; }
.tree-delete-btn {
  display: none; border: none; background: none; cursor: pointer;
  color: var(--color-text-muted); font-size: 0.923rem; line-height: 1; padding: 0 0.154rem;
  flex-shrink: 0;
}
.tree-delete-btn:hover { color: #e55; }
.tree-item:hover .tree-delete-btn { display: inline; }
.tree-badge { font-size: 0.846rem; color: var(--color-text-muted); flex-shrink: 0; }
</style>

<style>
.tree-collapse {
  font-size: 0.615rem; width: 0.769rem; text-align: center; flex-shrink: 0;
  color: var(--color-text-muted); cursor: pointer;
}
</style>
