<script setup lang="ts">
import type { IPKDetail, IUniqueDetail, ICheckDetail, IExcludeDetail } from '@/api/factory'

type ConstraintItem =
  | { kind: 'pk'; data: IPKDetail }
  | { kind: 'unique'; data: IUniqueDetail; index: number }
  | { kind: 'check'; data: ICheckDetail; index: number }
  | { kind: 'exclude'; data: IExcludeDetail; index: number }

const props = defineProps<{
  item: ConstraintItem
  columns: string[] // table column names for checkboxes
}>()

const emit = defineEmits<{
  updatePK: [pk: IPKDetail]
  updateUnique: [index: number, data: IUniqueDetail]
  updateCheck: [index: number, data: ICheckDetail]
}>()

function togglePKColumn(colName: string) {
  if (props.item.kind !== 'pk') return
  const pk = props.item.data
  const cols = pk.columns.includes(colName)
    ? pk.columns.filter(c => c !== colName)
    : [...pk.columns, colName]
  emit('updatePK', { ...pk, columns: cols })
}

function setPKName(name: string) {
  if (props.item.kind !== 'pk') return
  emit('updatePK', { ...props.item.data, name })
}

function setUnique(field: string, value: string | boolean) {
  if (props.item.kind !== 'unique') return
  emit('updateUnique', props.item.index, { ...props.item.data, [field]: value })
}

function toggleUniqueColumn(colName: string) {
  if (props.item.kind !== 'unique') return
  const u = props.item.data
  const cols = u.columns.includes(colName)
    ? u.columns.filter(c => c !== colName)
    : [...u.columns, colName]
  emit('updateUnique', props.item.index, { ...u, columns: cols })
}

function setCheck(field: string, value: string) {
  if (props.item.kind !== 'check') return
  emit('updateCheck', props.item.index, { ...props.item.data, [field]: value })
}
</script>

<template>
  <div class="cp-panel">
    <!-- PK -->
    <template v-if="item.kind === 'pk'">
      <div class="cp-title">Primary Key</div>
      <div class="cp-row">
        <label class="cp-label">Name</label>
        <input class="cp-input" :value="item.data.name" @change="setPKName(($event.target as HTMLInputElement).value)" />
      </div>
      <div class="cp-group-label">Columns</div>
      <div v-for="col in columns" :key="col" class="cp-check-row">
        <label class="cp-check">
          <input type="checkbox" :checked="item.data.columns.includes(col)" @change="togglePKColumn(col)" />
          {{ col }}
        </label>
      </div>
    </template>

    <!-- Unique -->
    <template v-else-if="item.kind === 'unique'">
      <div class="cp-title">Unique Constraint</div>
      <div class="cp-row">
        <label class="cp-label">Name</label>
        <input class="cp-input" :value="item.data.name" @change="setUnique('name', ($event.target as HTMLInputElement).value)" />
      </div>
      <div class="cp-row">
        <label class="cp-label"></label>
        <label class="cp-check">
          <input type="checkbox" :checked="item.data.nullsDistinct" @change="setUnique('nullsDistinct', ($event.target as HTMLInputElement).checked)" />
          Nulls Distinct
        </label>
      </div>
      <div class="cp-group-label">Columns</div>
      <div v-for="col in columns" :key="col" class="cp-check-row">
        <label class="cp-check">
          <input type="checkbox" :checked="item.data.columns.includes(col)" @change="toggleUniqueColumn(col)" />
          {{ col }}
        </label>
      </div>
    </template>

    <!-- Check -->
    <template v-else-if="item.kind === 'check'">
      <div class="cp-title">Check Constraint</div>
      <div class="cp-row">
        <label class="cp-label">Name</label>
        <input class="cp-input" :value="item.data.name" @change="setCheck('name', ($event.target as HTMLInputElement).value)" />
      </div>
      <div class="cp-row">
        <label class="cp-label">Expression</label>
        <textarea class="cp-textarea" :value="item.data.expression" rows="3" @change="setCheck('expression', ($event.target as HTMLTextAreaElement).value)" />
      </div>
    </template>

    <!-- Exclude (read-only) -->
    <template v-else-if="item.kind === 'exclude'">
      <div class="cp-title">Exclude Constraint (read-only)</div>
      <div class="cp-row">
        <label class="cp-label">Name</label>
        <input class="cp-input" :value="item.data.name" disabled />
      </div>
      <div class="cp-row">
        <label class="cp-label">Using</label>
        <input class="cp-input" :value="item.data.using" disabled />
      </div>
      <div class="cp-group-label">Elements</div>
      <div v-for="(el, i) in item.data.elements" :key="i" class="cp-row">
        <span class="cp-mono">{{ el.expression || el.column }} WITH {{ el.with }}</span>
      </div>
      <template v-if="item.data.where">
        <div class="cp-group-label">Where</div>
        <div class="cp-row">
          <span class="cp-mono cp-where">{{ item.data.where }}</span>
        </div>
      </template>
    </template>
  </div>
</template>

<style scoped>
.cp-panel { padding: 0.615rem 0.923rem; font-size: 0.923rem; overflow-y: auto; height: 100%; color: var(--color-text-primary); }
.cp-title { font-weight: 600; font-size: 0.846rem; color: var(--color-text-secondary); margin-bottom: 0.615rem; padding-bottom: 0.308rem; border-bottom: 1px solid var(--color-border); }
.cp-row { display: flex; align-items: center; gap: 0.462rem; margin-bottom: 0.462rem; }
.cp-label { width: 5.385rem; font-size: 0.846rem; color: var(--color-text-secondary); flex-shrink: 0; }
.cp-input {
  flex: 1; padding: 1px 0.308rem; font-size: 0.923rem; height: 1.538rem;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-primary); outline: none;
}
.cp-input:focus { border-color: var(--color-accent); }
.cp-input:disabled { opacity: 0.5; }
.cp-textarea {
  flex: 1; padding: 0.231rem 0.308rem; font-size: 0.923rem; font-family: monospace;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-primary); outline: none; resize: vertical;
}
.cp-textarea:focus { border-color: var(--color-accent); }
.cp-group-label { font-size: 0.769rem; font-weight: 600; color: var(--color-text-muted); margin: 0.462rem 0 0.231rem; }
.cp-check-row { margin-bottom: 0.154rem; }
.cp-check { font-size: 0.923rem; display: flex; align-items: center; gap: 0.308rem; cursor: pointer; }
.cp-mono { font-family: monospace; font-size: 0.846rem; color: var(--color-text-secondary); }
.cp-where { word-break: break-all; }
</style>
