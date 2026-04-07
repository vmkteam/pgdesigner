<script setup lang="ts">
import { computed, onMounted, nextTick, useTemplateRef } from 'vue'
import { whenever } from '@vueuse/core'
import type { IPKDetail, IUniqueDetail, ICheckDetail, IExcludeDetail } from '@/api/factory'
import ConstraintProperties from './ConstraintProperties.vue'
import { useListKeyboard } from '@/composables/useListKeyboard'
import { useInlineEdit } from '@/composables/useInlineEdit'

const props = defineProps<{
  pk?: IPKDetail | null
  uniques: IUniqueDetail[]
  checks: ICheckDetail[]
  excludes: IExcludeDetail[]
  columns: string[]       // table column names
  tableName: string       // for auto-generated names
  focusItem?: string | null
}>()

const emit = defineEmits<{
  updatePK: [pk: IPKDetail | null]
  updateUniques: [uniques: IUniqueDetail[]]
  updateChecks: [checks: ICheckDetail[]]
  updateExcludes: [excludes: IExcludeDetail[]]
  contextChange: [context: string]
}>()

type ConstraintItem =
  | { kind: 'pk'; data: IPKDetail }
  | { kind: 'unique'; data: IUniqueDetail; index: number }
  | { kind: 'check'; data: ICheckDetail; index: number }
  | { kind: 'exclude'; data: IExcludeDetail; index: number }

const listRef = useTemplateRef<HTMLElement>('listRef')

const items = computed<ConstraintItem[]>(() => {
  const list: ConstraintItem[] = []
  if (props.pk) list.push({ kind: 'pk', data: props.pk })
  ;(props.uniques || []).forEach((u, i) => list.push({ kind: 'unique', data: u, index: i }))
  ;(props.checks || []).forEach((c, i) => list.push({ kind: 'check', data: c, index: i }))
  ;(props.excludes || []).forEach((e, i) => list.push({ kind: 'exclude', data: e, index: i }))
  return list
})

const selected = computed(() => selectedIdx.value != null ? items.value[selectedIdx.value] : null)

function itemLabel(item: ConstraintItem): string {
  switch (item.kind) {
    case 'pk': return 'PK'
    case 'unique': return 'UQ'
    case 'check': return 'CHK'
    case 'exclude': return 'EXCL'
  }
}

function itemDetail(item: ConstraintItem): string {
  switch (item.kind) {
    case 'pk': return item.data.columns.join(', ')
    case 'unique': return item.data.columns.join(', ')
    case 'check': return item.data.expression
    case 'exclude': return item.data.elements.map(el => `${el.expression || el.column} WITH ${el.with}`).join(', ')
  }
}

function addUnique() {
  const u = props.uniques || []
  const name = `uq_${props.tableName}_${u.length + 1}`
  emit('updateUniques', [...u, { name, columns: [], nullsDistinct: false }])
  const newIdx = (props.pk ? 1 : 0) + u.length
  selectedIdx.value = newIdx
}

function addCheck() {
  const u = props.uniques || []
  const c = props.checks || []
  const name = `chk_${props.tableName}_${c.length + 1}`
  emit('updateChecks', [...c, { name, expression: '' }])
  const newIdx = (props.pk ? 1 : 0) + u.length + c.length
  selectedIdx.value = newIdx
}

function addExclude() {
  const u = props.uniques || []
  const c = props.checks || []
  const e = props.excludes || []
  const name = `excl_${props.tableName}_${e.length + 1}`
  emit('updateExcludes', [...e, { name, using: 'gist', elements: [{ column: '', expression: '', opclass: '', with: '=' }], where: '' }])
  const newIdx = (props.pk ? 1 : 0) + u.length + c.length + e.length
  selectedIdx.value = newIdx
}

function deleteSelected() {
  if (selected.value == null) return
  const item = selected.value
  if (item.kind === 'pk') {
    emit('updatePK', null)
  } else if (item.kind === 'unique') {
    emit('updateUniques', (props.uniques || []).filter((_, i) => i !== item.index))
  } else if (item.kind === 'check') {
    emit('updateChecks', (props.checks || []).filter((_, i) => i !== item.index))
  } else if (item.kind === 'exclude') {
    emit('updateExcludes', (props.excludes || []).filter((_, i) => i !== item.index))
  }
  selectedIdx.value = null
}

// Properties callbacks
function onUpdatePK(pk: IPKDetail) { emit('updatePK', pk) }
function onUpdateUnique(index: number, data: IUniqueDetail) {
  const copy = [...props.uniques]
  copy[index] = data
  emit('updateUniques', copy)
}
function onUpdateCheck(index: number, data: ICheckDetail) {
  const copy = [...props.checks]
  copy[index] = data
  emit('updateChecks', copy)
}
function onUpdateExclude(index: number, data: IExcludeDetail) {
  const copy = [...props.excludes]
  copy[index] = data
  emit('updateExcludes', copy)
}

const { editingIdx, editName, editError, startEdit: startEditName, commit: commitEditName, onEditKeydown } = useInlineEdit({
  getName: (i) => items.value[i]!.data.name,
  onCommit: (i, name) => {
    const item = items.value[i]!
    if (item.kind === 'pk') onUpdatePK({ ...item.data, name })
    else if (item.kind === 'unique') onUpdateUnique(item.index, { ...item.data, name })
    else if (item.kind === 'check') onUpdateCheck(item.index, { ...item.data, name })
  },
  listRef,
  inputClass: 'cl-name-input',
})

const { selectedIdx, onKeydown: onListKeydown } = useListKeyboard({
  count: () => items.value.length,
  onAdd: addUnique,
  onDelete: deleteSelected,
  onEdit: startEditName,
})

onMounted(() => nextTick(() => listRef.value?.focus()))

// Auto-select by name from GoTo
whenever(() => props.focusItem, (name) => {
  const idx = items.value.findIndex(i => i.data.name === name)
  if (idx >= 0) selectedIdx.value = idx
}, { immediate: true })

function onKeydown(e: KeyboardEvent) {
  if (editingIdx.value != null) return
  onListKeydown(e)
}
</script>

<template>
  <div class="cl-split">
    <div class="cl-list">
      <table
        ref="listRef"
        class="cl-table"
        tabindex="0"
        @keydown="onKeydown"
        @focus="emit('contextChange', 'constraints')"
      >
        <thead>
          <tr>
            <th style="width: 3rem">Type</th>
            <th>Name</th>
            <th>Detail</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(item, i) in items" :key="item.data.name || i"
            :class="{ 'row-selected': selectedIdx === i }"
            @click="selectedIdx = i"
          >
            <td class="cl-type">{{ itemLabel(item) }}</td>
            <td class="cl-name" @dblclick.stop="startEditName(i)">
              <input v-if="editingIdx === i" v-model="editName" class="cl-name-input" :class="{ 'edit-error': editError }" :title="editError || ''" @blur="commitEditName" @keydown="onEditKeydown" />
              <span v-else>{{ item.data.name }}</span>
            </td>
            <td class="cl-detail">{{ itemDetail(item) }}</td>
          </tr>
        </tbody>
      </table>

      <div v-if="items.length === 0" class="cl-empty">No constraints defined</div>

      <div class="cl-toolbar">
        <button class="cl-btn" title="Add UNIQUE (+)" @click="addUnique">+ Unique</button>
        <button class="cl-btn" title="Add CHECK" @click="addCheck">+ Check</button>
        <button class="cl-btn" title="Add EXCLUDE" @click="addExclude">+ Exclude</button>
        <button class="cl-btn" title="Delete (−)" :disabled="selected == null || selected.kind === 'pk'" @click="deleteSelected">− Delete</button>
      </div>
    </div>

    <div v-if="selected" class="cl-props">
      <ConstraintProperties
        :item="selected"
        :columns="columns"
        @update-p-k="onUpdatePK"
        @update-unique="onUpdateUnique"
        @update-check="onUpdateCheck"
        @update-exclude="onUpdateExclude"
      />
    </div>
  </div>
</template>

<style scoped>
.cl-split { display: flex; height: 100%; }
.cl-list { flex: 1; display: flex; flex-direction: column; overflow: auto; min-width: 0; border-right: 1px solid var(--color-border); }
.cl-props { width: 21.538rem; flex-shrink: 0; overflow-y: auto; }
.cl-table { width: 100%; border-collapse: collapse; font-size: 0.923rem; color: var(--color-text-primary); table-layout: fixed; outline: none; }
.cl-table thead { background: var(--color-bg-app); position: sticky; top: 0; z-index: 1; }
.cl-table th { padding: 0 0.308rem; font-weight: 600; font-size: 0.846rem; border-bottom: 1px solid var(--color-border); text-align: left; line-height: 1.308rem; }
.cl-table td { padding: 0 0.308rem; border-bottom: 1px solid var(--color-border-subtle); line-height: 1.308rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 0.923rem; }
.cl-table tbody tr { cursor: default; }
.cl-table tbody tr:hover { background: var(--color-bg-hover); }
.row-selected { background: var(--color-bg-selected) !important; }
.cl-type { font-weight: 600; font-size: 0.769rem; color: var(--color-text-muted); }
.cl-name { cursor: text; }
.cl-name-input {
  width: 100%; padding: 0 0.231rem; font-size: 0.923rem; line-height: 1.154rem; height: 1.308rem;
  border: 1px solid var(--color-accent); box-sizing: border-box;
  background: var(--color-bg-surface); color: var(--color-text-primary); outline: none;
}
.cl-detail { color: var(--color-text-secondary); font-size: 0.846rem; }
.cl-empty { padding: 1rem; text-align: center; color: var(--color-text-muted); font-size: 0.923rem; }
.cl-toolbar {
  padding: 0.308rem 0.462rem; border-top: 1px solid var(--color-border);
  display: flex; gap: 0.308rem; background: var(--color-bg-app); flex-shrink: 0;
}
.cl-btn {
  padding: 0.154rem 0.615rem; font-size: 0.846rem;
  border: 1px solid var(--color-menu-border); background: var(--color-bg-surface);
  color: var(--color-text-primary);
}
.cl-btn:hover:not(:disabled) { background: var(--color-bg-hover); }
.cl-btn:disabled { opacity: 0.5; }
</style>
