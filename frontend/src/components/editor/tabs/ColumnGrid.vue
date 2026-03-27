<script setup lang="ts">
import { ref, computed, nextTick, watch, onMounted, useTemplateRef } from 'vue'
import type { IColumnDetail } from '@/api/factory'
import { useEditorStore } from '@/stores/editor'
import TypeAutocomplete from './TypeAutocomplete.vue'

const editor = useEditorStore()

const props = defineProps<{
  columns: IColumnDetail[]
  selected: number | null
}>()

const emit = defineEmits<{
  select: [index: number]
  update: [index: number, field: string, value: string | number | boolean | null | object]
  togglePK: [columnName: string]
  add: []
  delete: [index: number]
  moveUp: [index: number]
  moveDown: [index: number]
  contextChange: [context: string]
}>()

// Navigable columns in order (skip #, FK)
const NAV_COLS = ['name', 'type', 'nn', 'pk', 'default'] as const
type NavCol = typeof NAV_COLS[number]

const editCell = ref<{ row: number; col: string } | null>(null)
const editValue = ref('')
const focusCol = ref<NavCol>('name')
const tableRef = useTemplateRef<HTMLElement>('tableRef')

// Columns that have at least one index
const indexedCols = computed(() => {
  const s = new Set<string>()
  for (const ix of editor.draft?.indexes || []) {
    for (const c of ix.columns || []) s.add(c.name)
  }
  return s
})

// Sync focused row with selected
watch(() => props.selected, (v) => {
  if (v != null) nextTick(() => scrollRowIntoView(v))
})

function scrollRowIntoView(row: number) {
  const tr = tableRef.value?.querySelectorAll('tbody tr')[row] as HTMLElement
  tr?.scrollIntoView({ block: 'nearest' })
}

function cellValue(row: number, col: NavCol): string {
  const c = props.columns[row]
  if (!c) return ''
  switch (col) {
    case 'name': return c.name
    case 'type': return c.type
    case 'default': return c.default || ''
    default: return ''
  }
}

function isToggleCol(col: NavCol): boolean {
  return col === 'nn' || col === 'pk'
}

function isEditableCol(col: NavCol): boolean {
  return col === 'name' || col === 'type' || col === 'default'
}

// --- Navigation ---

function navigate(dRow: number, dCol: number) {
  const row = props.selected ?? 0
  const colIdx = NAV_COLS.indexOf(focusCol.value)

  const newRow = Math.max(0, Math.min(row + dRow, props.columns.length - 1))
  const newColIdx = Math.max(0, Math.min(colIdx + dCol, NAV_COLS.length - 1))

  focusCol.value = NAV_COLS[newColIdx]!
  if (newRow !== row) emit('select', newRow)
}

function navigateToCell(row: number, col: NavCol) {
  focusCol.value = col
  emit('select', row)
}

// --- Editing ---

function startEdit(row: number, col: string, value: string) {
  editCell.value = { row, col }
  editValue.value = value ?? ''
  emit('select', row)
  emit('contextChange', 'grid-edit')
  nextTick(() => {
    const el = tableRef.value?.querySelector('.cg-input') as HTMLInputElement
    if (el) { el.focus(); el.select() }
  })
}

function commitEdit(direction?: 'right' | 'left' | 'down') {
  if (!editCell.value) return
  const { row, col } = editCell.value
  emit('update', row, col, editValue.value)
  editCell.value = null
  emit('contextChange', 'grid')

  // Auto-navigate after commit
  if (direction === 'right' || direction === 'left') {
    const colIdx = NAV_COLS.indexOf(col as NavCol)
    const step = direction === 'right' ? 1 : -1
    let next = colIdx + step
    // Skip toggle columns for text edit navigation
    while (next >= 0 && next < NAV_COLS.length && isToggleCol(NAV_COLS[next]!)) next += step
    if (next >= 0 && next < NAV_COLS.length) {
      const nextCol = NAV_COLS[next]!
      focusCol.value = nextCol
      if (isEditableCol(nextCol)) {
        nextTick(() => startEdit(row, nextCol, cellValue(row, nextCol)))
      }
    }
  } else if (direction === 'down') {
    const nextRow = row + 1
    if (nextRow < props.columns.length) {
      emit('select', nextRow)
      focusCol.value = col as NavCol
    }
  }

  nextTick(() => tableRef.value?.focus())
}

function cancelEdit() {
  editCell.value = null
  emit('contextChange', 'grid')
  nextTick(() => tableRef.value?.focus())
}

function isEditing(row: number, col: string) {
  return editCell.value?.row === row && editCell.value?.col === col
}

// --- Edit mode keydown (inside input) ---

function onEditKeydown(e: KeyboardEvent) {
  e.stopPropagation() // prevent grid navigation handler from firing
  if (e.key === 'Enter') { e.preventDefault(); commitEdit('down') }
  else if (e.key === 'Tab' && !e.shiftKey) { e.preventDefault(); commitEdit('right') }
  else if (e.key === 'Tab' && e.shiftKey) { e.preventDefault(); commitEdit('left') }
  else if (e.key === 'Escape') { cancelEdit() }
}

// --- Grid keydown (navigation mode) ---

function onGridKeydown(e: KeyboardEvent) {
  // Don't handle if editing
  if (editCell.value) return
  if (props.columns.length === 0) return

  const row = props.selected ?? 0
  const col = focusCol.value

  switch (e.key) {
    case 'ArrowUp':
      e.preventDefault()
      if (e.ctrlKey || e.metaKey) { emit('moveUp', row) }
      else { navigate(-1, 0) }
      break
    case 'ArrowDown':
      e.preventDefault()
      if (e.ctrlKey || e.metaKey) { emit('moveDown', row) }
      else { navigate(1, 0) }
      break
    case 'ArrowLeft':
      e.preventDefault(); navigate(0, -1)
      break
    case 'ArrowRight':
      e.preventDefault(); navigate(0, 1)
      break
    case 'Enter':
    case 'F2':
      e.preventDefault()
      if (isEditableCol(col)) startEdit(row, col, cellValue(row, col))
      else if (isToggleCol(col)) toggleCell(row, col)
      break
    case ' ':
      e.preventDefault()
      if (isToggleCol(col)) toggleCell(row, col)
      break
    case 'Delete':
      e.preventDefault()
      if (e.ctrlKey || e.metaKey) { emit('delete', row) }
      else if (isEditableCol(col)) { emit('update', row, col, '') }
      break
    case 'Insert':
    case '+':
      e.preventDefault(); emit('add')
      break
    case '-':
      e.preventDefault()
      if (row >= 0) emit('delete', row)
      break
    case 'Tab':
      e.preventDefault()
      if (e.shiftKey) navigate(0, -1)
      else navigate(0, 1)
      break
    default:
      // Start editing on printable character (except + = which are shortcuts)
      if (e.key.length === 1 && !e.ctrlKey && !e.metaKey && !e.altKey && !'+=-'.includes(e.key) && isEditableCol(col)) {
        startEdit(row, col, '')
        editValue.value = e.key
      }
  }
}

function toggleCell(row: number, col: NavCol) {
  if (col === 'nn') { emit('update', row, 'nullable', !props.columns[row]!.nullable) }
  else if (col === 'pk') { emit('togglePK', props.columns[row]!.name) }
}

function editName(row: number) {
  startEdit(row, 'name', props.columns[row]?.name || '')
}

function displayType(col: IColumnDetail): string {
  let t = col.type
  let suffix = ''
  if (t.endsWith('[]')) { suffix = '[]'; t = t.slice(0, -2) }
  if (col.length > 0) return `${t}(${col.length})${suffix}`
  if (col.precision > 0 && col.scale > 0) return `${t}(${col.precision},${col.scale})${suffix}`
  if (col.precision > 0) return `${t}(${col.precision})${suffix}`
  return `${t}${suffix}`
}

function isFocused(row: number, col: NavCol): boolean {
  return props.selected === row && focusCol.value === col && !editCell.value
}

onMounted(() => nextTick(() => tableRef.value?.focus()))

defineExpose({ editName })
</script>

<template>
  <div class="cg-wrap">
    <table
      ref="tableRef"
      class="cg-table"
      tabindex="0"
      @keydown="onGridKeydown"
      @focus="emit('contextChange', 'grid')"
    >
      <colgroup>
        <col style="width: 28px" />
        <col />
        <col style="width: 28%" />
        <col style="width: 28px" />
        <col style="width: 28px" />
        <col style="width: 28px" />
        <col style="width: 28px" />
        <col style="width: 18%" />
      </colgroup>
      <thead>
        <tr>
          <th class="text-center">#</th>
          <th>Name</th>
          <th>Type</th>
          <th class="text-center">NN</th>
          <th class="text-center">PK</th>
          <th class="text-center">FK</th>
          <th class="text-center">Ix</th>
          <th>Default</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(col, i) in columns" :key="col.name || i"
          :class="{ 'row-selected': selected === i }"
          @click="emit('select', i)"
        >
          <td class="text-center cg-num">{{ i + 1 }}</td>

          <td
            class="cg-cell"
            :class="{ 'cell-focused': isFocused(i, 'name'), 'cell-error': editor.fieldHasError(`col.${i}.name`) }"
            :title="editor.errors.find((e: any) => e.field === `col.${i}.name`)?.message"
            @dblclick.stop="startEdit(i, 'name', col.name)"
            @click.stop="navigateToCell(i, 'name')"
          >
            <input v-if="isEditing(i, 'name')" v-model="editValue" class="cg-input" maxlength="63" @blur="commitEdit()" @keydown="onEditKeydown" />
            <span v-else :class="{ bold: !col.nullable }">{{ col.name }}</span>
          </td>

          <td
            class="cg-cell cg-type-cell"
            :class="{ 'cell-focused': isFocused(i, 'type'), 'cell-error': editor.fieldHasError(`col.${i}.type`) }"
            :title="editor.errors.find((e: any) => e.field === `col.${i}.type`)?.message"
            @dblclick.stop="startEdit(i, 'type', col.type)"
            @click.stop="navigateToCell(i, 'type')"
          >
            <TypeAutocomplete
              v-if="isEditing(i, 'type')"
              :model-value="editValue"
              @update:model-value="editValue = $event"
              @commit="commitEdit()"
              @cancel="cancelEdit"
            />
            <span v-else>{{ displayType(col) }}</span>
          </td>

          <td
            class="text-center cg-toggle"
            :class="{ 'cell-focused': isFocused(i, 'nn') }"
            @click.stop="emit('update', i, 'nullable', !col.nullable); navigateToCell(i, 'nn')"
          >
            <span v-if="!col.nullable" class="nn-check">✓</span>
          </td>

          <td
            class="text-center cg-toggle"
            :class="{ 'cell-focused': isFocused(i, 'pk') }"
            @click.stop="emit('togglePK', col.name); navigateToCell(i, 'pk')"
          >
            <span v-if="col.pk" class="pk-icon">🔑</span>
            <span v-else class="pk-placeholder">·</span>
          </td>

          <td class="text-center"><span v-if="col.fk" class="fk-icon">↗</span></td>

          <td class="text-center"><span v-if="indexedCols.has(col.name)" class="ix-icon">▤</span></td>

          <td
            class="cg-cell cg-muted"
            :class="{ 'cell-focused': isFocused(i, 'default') }"
            @dblclick.stop="startEdit(i, 'default', col.default || '')"
            @click.stop="navigateToCell(i, 'default')"
          >
            <input v-if="isEditing(i, 'default')" v-model="editValue" class="cg-input" @blur="commitEdit()" @keydown="onEditKeydown" />
            <span v-else>{{ col.default }}</span>
          </td>
        </tr>
      </tbody>
    </table>

    <div class="cg-toolbar">
      <button class="cg-btn" title="Add column (Insert)" @click="emit('add')">+ Add</button>
      <button class="cg-btn" title="Delete column (Ctrl+Delete)" :disabled="selected == null" @click="selected != null && emit('delete', selected)">⊘ Delete</button>
    </div>
  </div>
</template>

<style scoped>
.cg-wrap { height: 100%; display: flex; flex-direction: column; }
.cg-table { width: 100%; border-collapse: collapse; font-size: 0.923rem; color: var(--color-text-primary); table-layout: fixed; outline: none; }
.cg-table thead { background: var(--color-bg-app); position: sticky; top: 0; z-index: 1; }
.cg-table th { padding: 0 0.308rem; font-weight: 600; font-size: 0.846rem; border-bottom: 1px solid var(--color-border); text-align: left; line-height: 1.308rem; }
.cg-table td { padding: 0 0.308rem; border-bottom: 1px solid var(--color-border-subtle); line-height: 1.308rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 0.923rem; }
.cg-table tbody tr { cursor: default; }
.cg-table tbody tr:hover { background: var(--color-bg-hover); }
.row-selected { background: var(--color-bg-selected) !important; }
.cell-focused { outline: 1px solid var(--color-accent); outline-offset: -1px; }
.cell-error { background: rgba(204, 51, 51, 0.08); outline: 1px solid #cc3333; outline-offset: -1px; }
.cg-num { color: var(--color-text-muted); font-size: 0.846rem; }
.bold { font-weight: 600; }
.nn-check { color: #cc0000; font-weight: 700; }
.pk-icon { font-size: 0.846rem; }
.pk-placeholder { color: var(--color-text-muted); font-size: 0.769rem; }
.fk-icon { color: #3366aa; font-weight: 700; }
.ix-icon { color: #668833; font-size: 0.769rem; }
.cg-cell { cursor: text; }
.cg-muted { color: var(--color-text-secondary); }
.cg-type-cell { position: relative; }
.cg-toggle { cursor: pointer; user-select: none; }
.cg-input {
  width: 100%; padding: 0 0.231rem; font-size: 0.923rem; line-height: 1.154rem; height: 1.308rem;
  border: 1px solid var(--color-accent); box-sizing: border-box;
  background: var(--color-bg-surface); color: var(--color-text-primary); outline: none;
}

.cg-toolbar {
  padding: 0.308rem 0.462rem; border-top: 1px solid var(--color-border);
  display: flex; gap: 0.308rem; background: var(--color-bg-app); flex-shrink: 0;
}
.cg-btn {
  padding: 0.154rem 0.615rem; font-size: 0.846rem;
  border: 1px solid var(--color-menu-border); background: var(--color-bg-surface);
  color: var(--color-text-primary);
}
.cg-btn:hover:not(:disabled) { background: var(--color-bg-hover); }
.cg-btn:disabled { opacity: 0.5; }
</style>
