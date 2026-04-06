<script setup lang="ts">
import { onMounted, nextTick, useTemplateRef } from 'vue'
import { whenever } from '@vueuse/core'
import type { IIndexDetail } from '@/api/factory'
import IndexProperties from './IndexProperties.vue'
import { useListKeyboard } from '@/composables/useListKeyboard'
import { useInlineEdit } from '@/composables/useInlineEdit'

const props = defineProps<{
  indexes: IIndexDetail[]
  columns: string[]
  tableName: string
  focusItem?: string | null
}>()

const emit = defineEmits<{
  updateIndexes: [indexes: IIndexDetail[]]
  contextChange: [context: string]
}>()

function addIndex() {
  const idxs = props.indexes || []
  const name = `ix_${props.tableName}_${idxs.length + 1}`
  emit('updateIndexes', [...idxs, { name, unique: false, nullsDistinct: false, using: 'btree', columns: [], expressions: [], with: [], where: '', include: [] }])
  selectedIdx.value = idxs.length
}

function deleteIndex() {
  if (selectedIdx.value == null) return
  emit('updateIndexes', (props.indexes || []).filter((_, i) => i !== selectedIdx.value))
  selectedIdx.value = null
}

function onUpdateIndex(idx: number, data: IIndexDetail) {
  const copy = [...props.indexes]
  copy[idx] = data
  emit('updateIndexes', copy)
}

const tableRef = useTemplateRef<HTMLElement>('tableRef')

const { editingIdx, editName, editError, startEdit: startEditName, commit: commitEditName, onEditKeydown } = useInlineEdit({
  getName: (i) => (props.indexes || [])[i]?.name || '',
  onCommit: (i, name) => onUpdateIndex(i, { ...(props.indexes || [])[i]!, name }),
  listRef: tableRef,
  inputClass: 'il-name-input',
})

const { selectedIdx, onKeydown: onListKeydown } = useListKeyboard({
  count: () => (props.indexes || []).length,
  onAdd: addIndex,
  onDelete: deleteIndex,
  onEdit: startEditName,
})

onMounted(() => nextTick(() => tableRef.value?.focus()))

whenever(() => props.focusItem, (name) => {
  const idx = (props.indexes || []).findIndex(i => i.name === name)
  if (idx >= 0) selectedIdx.value = idx
}, { immediate: true })

function onKeydown(e: KeyboardEvent) {
  if (editingIdx.value != null) return
  onListKeydown(e)
}
</script>

<template>
  <div class="il-split">
    <div class="il-list">
      <table ref="tableRef" class="il-table" tabindex="0" @keydown="onKeydown" @focus="emit('contextChange', 'constraints')">
        <thead>
          <tr>
            <th>Name</th>
            <th style="width: 4rem">Method</th>
            <th>Columns</th>
            <th style="width: 2rem" class="text-center">UQ</th>
            <th>WHERE</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(idx, i) in (indexes || [])" :key="idx.name || i"
            :class="{ 'row-selected': selectedIdx === i }"
            @click="selectedIdx = i"
          >
            <td class="il-name" @dblclick.stop="startEditName(i)">
              <input v-if="editingIdx === i" v-model="editName" class="il-name-input" :class="{ 'edit-error': editError }" :title="editError || ''" @blur="commitEditName" @keydown="onEditKeydown" />
              <span v-else>{{ idx.name }}</span>
            </td>
            <td class="il-method">{{ idx.using || 'btree' }}</td>
            <td class="il-detail">
              {{ (idx.columns || []).map((c: any) => typeof c === 'string' ? c : c.name).join(', ') }}
              <span v-if="idx.expressions?.length" class="il-expr">{{ idx.expressions.join(', ') }}</span>
            </td>
            <td class="text-center"><span v-if="idx.unique">✓</span></td>
            <td class="il-detail il-mono">{{ idx.where }}</td>
          </tr>
        </tbody>
      </table>

      <div v-if="!(indexes || []).length" class="il-empty">No indexes defined</div>

      <div class="il-toolbar">
        <button class="il-btn" title="Add index (+)" @click="addIndex">+ Add</button>
        <button class="il-btn" title="Delete (−)" :disabled="selectedIdx == null" @click="deleteIndex">− Delete</button>
      </div>
    </div>

    <div v-if="selectedIdx != null && indexes[selectedIdx]" class="il-props">
      <IndexProperties :index="indexes[selectedIdx]!" :idx="selectedIdx" :columns="columns" @update="onUpdateIndex" />
    </div>
  </div>
</template>

<style scoped>
.il-split { display: flex; height: 100%; }
.il-list { flex: 1; display: flex; flex-direction: column; overflow: auto; min-width: 0; border-right: 1px solid var(--color-border); }
.il-props { width: 21.538rem; flex-shrink: 0; overflow-y: auto; }
.il-table { width: 100%; border-collapse: collapse; font-size: 0.923rem; color: var(--color-text-primary); table-layout: fixed; outline: none; }
.il-table thead { background: var(--color-bg-app); position: sticky; top: 0; z-index: 1; }
.il-table th { padding: 0 0.308rem; font-weight: 600; font-size: 0.846rem; border-bottom: 1px solid var(--color-border); text-align: left; line-height: 1.308rem; }
.il-table td { padding: 0 0.308rem; border-bottom: 1px solid var(--color-border-subtle); line-height: 1.308rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 0.923rem; }
.il-table tbody tr { cursor: default; }
.il-table tbody tr:hover { background: var(--color-bg-hover); }
.row-selected { background: var(--color-bg-selected) !important; }
.il-name { cursor: text; }
.il-name-input {
  width: 100%; padding: 0 0.231rem; font-size: 0.923rem; line-height: 1.154rem; height: 1.308rem;
  border: 1px solid var(--color-accent); box-sizing: border-box;
  background: var(--color-bg-surface); color: var(--color-text-primary); outline: none;
}
.il-method { font-size: 0.846rem; color: var(--color-text-muted); }
.il-detail { color: var(--color-text-secondary); font-size: 0.846rem; }
.il-expr { font-family: monospace; color: var(--color-text-muted); }
.il-mono { font-family: monospace; }
.il-empty { padding: 1rem; text-align: center; color: var(--color-text-muted); font-size: 0.923rem; }
.il-toolbar { padding: 0.308rem 0.462rem; border-top: 1px solid var(--color-border); display: flex; gap: 0.308rem; background: var(--color-bg-app); flex-shrink: 0; }
.il-btn { padding: 0.154rem 0.615rem; font-size: 0.846rem; border: 1px solid var(--color-menu-border); background: var(--color-bg-surface); color: var(--color-text-primary); }
.il-btn:hover:not(:disabled) { background: var(--color-bg-hover); }
.il-btn:disabled { opacity: 0.5; }
</style>
