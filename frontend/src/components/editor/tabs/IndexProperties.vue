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

function set(field: string, value: string | boolean | IIndexDetail['columns'] | IIndexDetail['expressions'] | IIndexDetail['include']) {
  emit('update', props.idx, { ...props.index, [field]: value })
}

function addCol() {
  set('columns', [...(props.index.columns || []), { name: '', order: '', nulls: '', opclass: '' }])
}
function removeCol(i: number) {
  set('columns', (props.index.columns || []).filter((_, j) => j !== i))
}
function updateCol(i: number, field: string, value: string) {
  const cols = [...(props.index.columns || [])]
  cols[i] = { ...cols[i]!, [field]: value }
  set('columns', cols)
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

    <div class="ip-section-label">Columns</div>
    <div v-for="(col, i) in (index.columns || [])" :key="i" class="ip-col-row">
      <select class="ip-col-select" :value="col.name" @change="updateCol(i, 'name', ($event.target as HTMLSelectElement).value)">
        <option value="">(select)</option>
        <option v-for="c in columns" :key="c" :value="c">{{ c }}</option>
      </select>
      <select class="ip-col-order" :value="col.order || ''" @change="updateCol(i, 'order', ($event.target as HTMLSelectElement).value)">
        <option value="">ASC</option>
        <option value="desc">DESC</option>
      </select>
      <select class="ip-col-nulls" :value="col.nulls || ''" @change="updateCol(i, 'nulls', ($event.target as HTMLSelectElement).value)">
        <option value="">(default)</option>
        <option value="first">NULLS FIRST</option>
        <option value="last">NULLS LAST</option>
      </select>
      <button class="ip-btn-del" @click="removeCol(i)">×</button>
    </div>
    <button class="ip-btn-add" @click="addCol">+ Add column</button>

    <div class="ip-row">
      <label class="ip-label">Expressions</label>
    </div>
    <div v-for="(expr, i) in (index.expressions || [])" :key="i" class="ip-expr-row">
      <input class="ip-input ip-mono" :value="expr" @change="set('expressions', [...(index.expressions || []).slice(0, i), ($event.target as HTMLInputElement).value, ...(index.expressions || []).slice(i + 1)])" />
      <button class="ip-btn-del" @click="set('expressions', (index.expressions || []).filter((_, j) => j !== i))">×</button>
    </div>
    <button class="ip-btn-add" @click="set('expressions', [...(index.expressions || []), ''])">+ Add expression</button>

    <DynamicColumnList label="Include" :model-value="index.include || []" :columns="columns" @update:model-value="set('include', $event)" />

    <template v-if="index.with?.length">
      <div class="ip-section-label">Storage Params</div>
      <div v-for="(p, i) in index.with" :key="i" class="ip-row">
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
.ip-col-row { display: flex; gap: 0.231rem; margin-bottom: 0.231rem; }
.ip-col-select {
  flex: 1; padding: 1px 0.308rem; font-size: 0.923rem; height: 1.538rem;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-primary); outline: none; cursor: pointer;
}
.ip-col-select:focus { border-color: var(--color-accent); }
.ip-col-order, .ip-col-nulls {
  width: 5rem; padding: 1px 0.231rem; font-size: 0.769rem; height: 1.538rem;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-secondary); outline: none; cursor: pointer;
}
.ip-col-order:focus, .ip-col-nulls:focus { border-color: var(--color-accent); }
.ip-expr-row { display: flex; gap: 0.231rem; margin-bottom: 0.231rem; }
.ip-btn-del {
  width: 1.538rem; height: 1.538rem; font-size: 0.923rem;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-secondary); cursor: pointer; display: flex; align-items: center; justify-content: center;
}
.ip-btn-del:hover { background: var(--color-bg-hover); color: #cc3333; }
.ip-btn-add { font-size: 0.769rem; color: var(--color-accent); background: none; border: none; cursor: pointer; padding: 0; margin-bottom: 0.462rem; }
.ip-btn-add:hover { text-decoration: underline; }
select.ip-input { cursor: pointer; }
</style>
