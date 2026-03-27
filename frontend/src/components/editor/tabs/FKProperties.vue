<script setup lang="ts">
import { ref, watch } from 'vue'
import type { IFKDetail } from '@/api/factory'
import api from '@/api/factory'

const props = defineProps<{
  fk: IFKDetail
  idx: number
  columns: string[]       // local table column names
  tables: string[]        // all table names in project
}>()

const emit = defineEmits<{
  update: [idx: number, data: IFKDetail]
}>()

// Target table columns for referenced column dropdown
const targetColumns = ref<string[]>([])

watch(() => props.fk.toTable, async (toTable) => {
  if (!toTable) { targetColumns.value = []; return }
  try {
    const data = await api.project.getTable({ name: toTable })
    targetColumns.value = (data.columns || []).map(c => c.name)
  } catch {
    targetColumns.value = []
  }
}, { immediate: true })

function set(field: string, value: string | boolean | IFKDetail['columns']) {
  emit('update', props.idx, { ...props.fk, [field]: value })
}

function addColPair() {
  const cols = [...(props.fk.columns || []), { name: '', references: '' }]
  set('columns', cols)
}

function removeColPair(i: number) {
  set('columns', props.fk.columns.filter((_, j) => j !== i))
}

function updateColPair(i: number, field: 'name' | 'references', value: string) {
  const cols = [...props.fk.columns]
  cols[i] = { ...cols[i]!, [field]: value }
  set('columns', cols)
}

const REF_ACTIONS = ['no action', 'restrict', 'cascade', 'set null', 'set default']
</script>

<template>
  <div class="fp-panel">
    <div class="fp-title">Foreign Key #{{ idx + 1 }}</div>

    <div class="fp-row">
      <label class="fp-label">Name</label>
      <input class="fp-input" :value="fk.name" @change="set('name', ($event.target as HTMLInputElement).value)" />
    </div>
    <div class="fp-row">
      <label class="fp-label">Target</label>
      <select class="fp-input" :value="fk.toTable" @change="set('toTable', ($event.target as HTMLSelectElement).value)">
        <option value="">(select table)</option>
        <option v-for="t in tables" :key="t" :value="t">{{ t }}</option>
      </select>
    </div>

    <div class="fp-group-label">Column Mapping</div>
    <div v-for="(cp, i) in fk.columns" :key="i" class="fp-pair-row">
      <select class="fp-pair-select" :value="cp.name" @change="updateColPair(i, 'name', ($event.target as HTMLSelectElement).value)">
        <option value="">(local)</option>
        <option v-for="c in columns" :key="c" :value="c">{{ c }}</option>
      </select>
      <span class="fp-arrow">→</span>
      <select class="fp-pair-select" :value="cp.references" @change="updateColPair(i, 'references', ($event.target as HTMLSelectElement).value)">
        <option value="">(referenced)</option>
        <option v-for="c in targetColumns" :key="c" :value="c">{{ c }}</option>
      </select>
      <button class="fp-btn-del" @click="removeColPair(i)">×</button>
    </div>
    <button class="fp-btn-add" @click="addColPair">+ Add pair</button>

    <div class="fp-row">
      <label class="fp-label">On Delete</label>
      <select class="fp-input" :value="fk.onDelete || 'no action'" @change="set('onDelete', ($event.target as HTMLSelectElement).value)">
        <option v-for="a in REF_ACTIONS" :key="a" :value="a">{{ a }}</option>
      </select>
    </div>
    <div class="fp-row">
      <label class="fp-label">On Update</label>
      <select class="fp-input" :value="fk.onUpdate || 'no action'" @change="set('onUpdate', ($event.target as HTMLSelectElement).value)">
        <option v-for="a in REF_ACTIONS" :key="a" :value="a">{{ a }}</option>
      </select>
    </div>
    <div class="fp-row">
      <label class="fp-label"></label>
      <label class="fp-check"><input type="checkbox" :checked="fk.deferrable" @change="set('deferrable', ($event.target as HTMLInputElement).checked)" /> Deferrable</label>
    </div>
    <div v-if="fk.deferrable" class="fp-row">
      <label class="fp-label">Initially</label>
      <select class="fp-input" :value="fk.initially || 'IMMEDIATE'" @change="set('initially', ($event.target as HTMLSelectElement).value)">
        <option value="IMMEDIATE">IMMEDIATE</option>
        <option value="DEFERRED">DEFERRED</option>
      </select>
    </div>
  </div>
</template>

<style scoped>
.fp-panel { padding: 0.615rem 0.923rem; font-size: 0.923rem; overflow-y: auto; height: 100%; color: var(--color-text-primary); }
.fp-title { font-weight: 600; font-size: 0.846rem; color: var(--color-text-secondary); margin-bottom: 0.615rem; padding-bottom: 0.308rem; border-bottom: 1px solid var(--color-border); }
.fp-row { display: flex; align-items: center; gap: 0.462rem; margin-bottom: 0.462rem; }
.fp-label { width: 5.385rem; font-size: 0.846rem; color: var(--color-text-secondary); flex-shrink: 0; }
.fp-input {
  flex: 1; padding: 1px 0.308rem; font-size: 0.923rem; height: 1.538rem;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-primary); outline: none;
}
.fp-input:focus { border-color: var(--color-accent); }
select.fp-input { cursor: pointer; }
.fp-group-label { font-size: 0.769rem; font-weight: 600; color: var(--color-text-muted); margin: 0.462rem 0 0.231rem; }
.fp-pair-row { display: flex; gap: 0.231rem; margin-bottom: 0.231rem; align-items: center; }
.fp-pair-select, .fp-pair-input {
  flex: 1; padding: 1px 0.308rem; font-size: 0.923rem; height: 1.538rem;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-primary); outline: none;
}
.fp-pair-select { cursor: pointer; }
.fp-pair-select:focus, .fp-pair-input:focus { border-color: var(--color-accent); }
.fp-arrow { font-size: 0.846rem; color: var(--color-text-muted); flex-shrink: 0; }
.fp-btn-del {
  width: 1.538rem; height: 1.538rem; font-size: 0.923rem;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-secondary); cursor: pointer; display: flex; align-items: center; justify-content: center;
}
.fp-btn-del:hover { background: var(--color-bg-hover); color: #cc3333; }
.fp-btn-add { font-size: 0.769rem; color: var(--color-accent); background: none; border: none; cursor: pointer; padding: 0; margin-bottom: 0.462rem; }
.fp-btn-add:hover { text-decoration: underline; }
.fp-check { font-size: 0.923rem; display: flex; align-items: center; gap: 0.308rem; cursor: pointer; }
</style>
