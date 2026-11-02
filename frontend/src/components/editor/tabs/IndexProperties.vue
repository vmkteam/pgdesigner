<script setup lang="ts">
import type { IIndexDetail } from '@/api/factory'
import DynamicColumnList from './DynamicColumnList.vue'

const props = defineProps<{
  index: IIndexDetail
  idx: number
  columns: string[]
}>()

const emit = defineEmits<{
  update: [idx: number, data: IIndexDetail]
}>()

function set(field: string, value: string | boolean | IIndexDetail['keys'] | IIndexDetail['include']) {
  emit('update', props.idx, { ...props.index, [field]: value })
}

function addKey(kind: 'column' | 'expression') {
  set('keys', [...(props.index.keys || []), { kind, name: '', expression: '', order: '', nulls: '', opclass: '' }])
}
function removeKey(i: number) {
  set('keys', (props.index.keys || []).filter((_, j) => j !== i))
}
function updateKey(i: number, field: string, value: string) {
  const keys = [...(props.index.keys || [])]
  keys[i] = { ...keys[i]!, [field]: value }
  set('keys', keys)
}
// moveKey swaps key i with its neighbour: the key order is the index definition.
function moveKey(i: number, delta: number) {
  const keys = [...(props.index.keys || [])]
  const j = i + delta
  if (j < 0 || j >= keys.length) return
  ;[keys[i], keys[j]] = [keys[j]!, keys[i]!]
  set('keys', keys)
}
</script>

<template>
  <div class="ip-panel">
    <div class="ip-title">Index #{{ idx + 1 }}</div>

    <div class="ip-row">
      <label class="ip-label">Name</label>
      <input class="ip-input" :value="index.name" @change="set('name', ($event.target as HTMLInputElement).value)" />
    </div>
    <div class="ip-row">
      <label class="ip-label">Method</label>
      <select class="ip-input" :value="index.using || 'btree'" @change="set('using', ($event.target as HTMLSelectElement).value)">
        <option value="btree">btree</option>
        <option value="hash">hash</option>
        <option value="gin">gin</option>
        <option value="gist">gist</option>
        <option value="spgist">spgist</option>
        <option value="brin">brin</option>
      </select>
    </div>
    <div class="ip-row">
      <label class="ip-label"></label>
      <label class="ip-check"><input type="checkbox" :checked="index.unique" @change="set('unique', ($event.target as HTMLInputElement).checked)" /> Unique</label>
      <label v-if="index.unique" class="ip-check"><input type="checkbox" :checked="index.nullsDistinct" @change="set('nullsDistinct', ($event.target as HTMLInputElement).checked)" /> Nulls Distinct</label>
    </div>

    <div class="ip-section-label">Columns and expressions</div>
    <div v-for="(key, i) in (index.keys || [])" :key="i" class="ip-key">
      <input v-if="key.kind === 'expression'" class="ip-expr-input ip-key-col ip-mono" :value="key.expression" placeholder="expression" @change="updateKey(i, 'expression', ($event.target as HTMLInputElement).value)" />
      <select v-else class="ip-col-select ip-key-col" :value="key.name" @change="updateKey(i, 'name', ($event.target as HTMLSelectElement).value)">
        <option value="">(select)</option>
        <option v-for="c in columns" :key="c" :value="c">{{ c }}</option>
      </select>
      <select class="ip-col-order" :value="key.order || ''" @change="updateKey(i, 'order', ($event.target as HTMLSelectElement).value)">
        <option value="">ASC</option>
        <option value="desc">DESC</option>
      </select>
      <select class="ip-col-nulls" :value="key.nulls || ''" @change="updateKey(i, 'nulls', ($event.target as HTMLSelectElement).value)">
        <option value="">(default)</option>
        <option value="first">NULLS FIRST</option>
        <option value="last">NULLS LAST</option>
      </select>
      <input class="ip-col-opclass ip-mono" :value="key.opclass || ''" placeholder="opclass" title="operator class (e.g. gin_trgm_ops)" @change="updateKey(i, 'opclass', ($event.target as HTMLInputElement).value)" />
      <button class="ip-btn-icon" title="Move up" :disabled="i === 0" @click="moveKey(i, -1)">↑</button>
      <button class="ip-btn-icon" title="Move down" :disabled="i === (index.keys || []).length - 1" @click="moveKey(i, 1)">↓</button>
      <button class="ip-btn-icon ip-btn-del" title="Remove" @click="removeKey(i)">×</button>
    </div>
    <div class="ip-add-row">
      <button class="ip-btn-add" @click="addKey('column')">+ Add column</button>
      <button class="ip-btn-add" @click="addKey('expression')">+ Add expression</button>
    </div>

    <DynamicColumnList label="Include" :model-value="index.include || []" :columns="columns" @update:model-value="set('include', $event)" />

    <template v-if="index.with?.length">
      <div class="ip-section-label">Storage Params</div>
      <div v-for="p in index.with" :key="p.name" class="ip-row">
        <span class="ip-mono">{{ p.name }} = {{ p.value }}</span>
      </div>
    </template>

    <div class="ip-row">
      <label class="ip-label">WHERE</label>
      <input class="ip-input ip-mono" :value="index.where || ''" placeholder="partial index predicate" @change="set('where', ($event.target as HTMLInputElement).value)" />
    </div>
  </div>
</template>

<style scoped>
.ip-panel { padding: 0.615rem 0.923rem; font-size: 0.923rem; overflow-y: auto; height: 100%; color: var(--color-text-primary); }
.ip-title { font-weight: 600; font-size: 0.846rem; color: var(--color-text-secondary); margin-bottom: 0.615rem; padding-bottom: 0.308rem; border-bottom: 1px solid var(--color-border); }
.ip-row { display: flex; align-items: center; gap: 0.462rem; margin-bottom: 0.462rem; }
.ip-label { width: 5.385rem; font-size: 0.846rem; color: var(--color-text-secondary); flex-shrink: 0; }
.ip-input {
  flex: 1; padding: 1px 0.308rem; font-size: 0.923rem; height: 1.538rem;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-primary); outline: none;
}
.ip-input:focus { border-color: var(--color-accent); }
.ip-mono { font-family: monospace; }
.ip-check { font-size: 0.923rem; display: flex; align-items: center; gap: 0.308rem; cursor: pointer; }
.ip-section-label { font-size: 0.769rem; font-weight: 600; color: var(--color-text-muted); margin: 0.462rem 0 0.231rem; }
.ip-col-select, .ip-expr-input {
  flex: 1 1 8rem; min-width: 6rem; padding: 1px 0.308rem; font-size: 0.923rem; height: 1.538rem;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-primary); outline: none;
}
.ip-col-select { cursor: pointer; }
.ip-col-select:focus, .ip-expr-input:focus { border-color: var(--color-accent); }
.ip-col-order, .ip-col-nulls {
  width: 4.25rem; padding: 1px 0.231rem; font-size: 0.769rem; height: 1.538rem;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-secondary); outline: none; cursor: pointer;
}
.ip-col-order:focus, .ip-col-nulls:focus { border-color: var(--color-accent); }
.ip-col-opclass {
  flex: 1 1 6rem; min-width: 5rem; padding: 1px 0.308rem; font-size: 0.846rem; height: 1.538rem;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-primary); outline: none;
}
.ip-col-opclass:focus { border-color: var(--color-accent); }
.ip-key { display: flex; flex-wrap: wrap; gap: 0.231rem; margin-bottom: 0.462rem; }
.ip-key-col { flex-basis: 100%; }
.ip-key .ip-col-opclass { flex-basis: 3rem; min-width: 3rem; }
.ip-btn-icon {
  width: 1.538rem; height: 1.538rem; font-size: 0.923rem;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-secondary); cursor: pointer; display: flex; align-items: center; justify-content: center;
}
.ip-btn-icon:hover:not(:disabled) { background: var(--color-bg-hover); color: var(--color-text-primary); }
.ip-btn-icon:disabled { opacity: 0.4; cursor: default; }
.ip-btn-del:hover { color: #cc3333; }
.ip-add-row { display: flex; gap: 0.923rem; }
.ip-btn-add { font-size: 0.769rem; color: var(--color-accent); background: none; border: none; cursor: pointer; padding: 0; margin-bottom: 0.462rem; }
.ip-btn-add:hover { text-decoration: underline; }
select.ip-input { cursor: pointer; }
</style>
