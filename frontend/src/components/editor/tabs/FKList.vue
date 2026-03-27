<script setup lang="ts">
import { onMounted, nextTick, useTemplateRef } from 'vue'
import { whenever } from '@vueuse/core'
import type { IFKDetail } from '@/api/factory'
import FKProperties from './FKProperties.vue'
import { useListKeyboard } from '@/composables/useListKeyboard'
import { useInlineEdit } from '@/composables/useInlineEdit'

const props = defineProps<{
  fks: IFKDetail[]
  columns: string[]
  tables: string[]
  tableName: string
  focusFK?: string | null
  defaultOnDelete?: string
  defaultOnUpdate?: string
}>()

const emit = defineEmits<{
  updateFKs: [fks: IFKDetail[]]
  contextChange: [context: string]
}>()

function addFK() {
  const fkList = props.fks || []
  const name = `fk_${props.tableName}_${fkList.length + 1}`
  emit('updateFKs', [...fkList, { name, toTable: '', onDelete: props.defaultOnDelete || 'no action', onUpdate: props.defaultOnUpdate || 'no action', deferrable: false, initially: '', columns: [{ name: '', references: '' }] }])
  selectedIdx.value = fkList.length
}

function deleteFK() {
  if (selectedIdx.value == null) return
  emit('updateFKs', (props.fks || []).filter((_, i) => i !== selectedIdx.value))
  selectedIdx.value = null
}

function onUpdateFK(idx: number, data: IFKDetail) {
  const copy = [...props.fks]
  copy[idx] = data
  emit('updateFKs', copy)
}

const tableRef = useTemplateRef<HTMLElement>('tableRef')

const { editingIdx, editName, editError, startEdit: startEditName, commit: commitEditName, onEditKeydown } = useInlineEdit({
  getName: (i) => (props.fks || [])[i]?.name || '',
  onCommit: (i, name) => onUpdateFK(i, { ...(props.fks || [])[i]!, name }),
  listRef: tableRef,
  inputClass: 'fl-name-input',
})

const { selectedIdx, onKeydown: onListKeydown } = useListKeyboard({
  count: () => (props.fks || []).length,
  onAdd: addFK,
  onDelete: deleteFK,
  onEdit: startEditName,
})

// Auto-select FK by name when focusFK is set
whenever(() => props.focusFK, (name) => {
  if (!props.fks) return
  const idx = props.fks.findIndex(fk => fk.name === name)
  if (idx >= 0) selectedIdx.value = idx
}, { immediate: true })

onMounted(() => nextTick(() => tableRef.value?.focus()))

function onKeydown(e: KeyboardEvent) {
  if (editingIdx.value != null) return
  onListKeydown(e)
}
</script>

<template>
  <div class="fl-split">
    <div class="fl-list">
      <table ref="tableRef" class="fl-table" tabindex="0" @keydown="onKeydown" @focus="emit('contextChange', 'constraints')">
        <thead>
          <tr>
            <th>Name</th>
            <th>Target</th>
            <th>Mapping</th>
            <th style="width: 5rem">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(fk, i) in (fks || [])" :key="fk.name || i"
            :class="{ 'row-selected': selectedIdx === i }"
            @click="selectedIdx = i"
          >
            <td class="fl-name" @dblclick.stop="startEditName(i)">
              <input v-if="editingIdx === i" v-model="editName" class="fl-name-input" :class="{ 'edit-error': editError }" :title="editError || ''" @blur="commitEditName" @keydown="onEditKeydown" />
              <span v-else>{{ fk.name }}</span>
            </td>
            <td class="fl-target">{{ fk.toTable }}</td>
            <td class="fl-detail">{{ fk.columns?.map(c => `${c.name}→${c.references}`).join(', ') }}</td>
            <td class="fl-detail">{{ fk.onDelete }}/{{ fk.onUpdate }}</td>
          </tr>
        </tbody>
      </table>

      <div v-if="!(fks || []).length" class="fl-empty">No foreign keys defined</div>

      <div class="fl-toolbar">
        <button class="fl-btn" title="Add FK (+)" @click="addFK">+ Add</button>
        <button class="fl-btn" title="Delete (−)" :disabled="selectedIdx == null" @click="deleteFK">− Delete</button>
      </div>
    </div>

    <div v-if="selectedIdx != null && fks[selectedIdx]" class="fl-props">
      <FKProperties :fk="fks[selectedIdx]!" :idx="selectedIdx" :columns="columns" :tables="tables" @update="onUpdateFK" />
    </div>
  </div>
</template>

<style scoped>
.fl-split { display: flex; height: 100%; }
.fl-list { flex: 1; display: flex; flex-direction: column; overflow: auto; min-width: 0; border-right: 1px solid var(--color-border); }
.fl-props { width: 21.538rem; flex-shrink: 0; overflow-y: auto; }
.fl-table { width: 100%; border-collapse: collapse; font-size: 0.923rem; color: var(--color-text-primary); table-layout: fixed; outline: none; }
.fl-table thead { background: var(--color-bg-app); position: sticky; top: 0; z-index: 1; }
.fl-table th { padding: 0 0.308rem; font-weight: 600; font-size: 0.846rem; border-bottom: 1px solid var(--color-border); text-align: left; line-height: 1.308rem; }
.fl-table td { padding: 0 0.308rem; border-bottom: 1px solid var(--color-border-subtle); line-height: 1.308rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 0.923rem; }
.fl-table tbody tr { cursor: default; }
.fl-table tbody tr:hover { background: var(--color-bg-hover); }
.row-selected { background: var(--color-bg-selected) !important; }
.fl-name { cursor: text; }
.fl-name-input {
  width: 100%; padding: 0 0.231rem; font-size: 0.923rem; line-height: 1.154rem; height: 1.308rem;
  border: 1px solid var(--color-accent); box-sizing: border-box;
  background: var(--color-bg-surface); color: var(--color-text-primary); outline: none;
}
.fl-target { color: var(--color-accent); }
.fl-detail { color: var(--color-text-secondary); font-size: 0.846rem; }
.fl-empty { padding: 1rem; text-align: center; color: var(--color-text-muted); font-size: 0.923rem; }
.fl-toolbar { padding: 0.308rem 0.462rem; border-top: 1px solid var(--color-border); display: flex; gap: 0.308rem; background: var(--color-bg-app); flex-shrink: 0; }
.fl-btn { padding: 0.154rem 0.615rem; font-size: 0.846rem; border: 1px solid var(--color-menu-border); background: var(--color-bg-surface); color: var(--color-text-primary); }
.fl-btn:hover:not(:disabled) { background: var(--color-bg-hover); }
.fl-btn:disabled { opacity: 0.5; }
</style>
