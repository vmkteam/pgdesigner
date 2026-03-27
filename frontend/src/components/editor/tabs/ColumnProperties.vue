<script setup lang="ts">
import { computed } from 'vue'
import type { IColumnDetail, IIndexDetail } from '@/api/factory'

const props = defineProps<{
  column: IColumnDetail
  index: number
  indexes: IIndexDetail[]
  tableName: string
}>()

const emit = defineEmits<{
  update: [index: number, field: string, value: string | number | boolean | null | object]
  togglePK: [columnName: string]
  goToIndex: [indexName: string]
  createIndex: [columnName: string]
}>()

function set(field: string, value: string | number | boolean | null | object) {
  emit('update', props.index, field, value)
}

// Strip [] suffix to get base type for modifier checks
const baseType = computed(() => {
  let t = (props.column.type || '').toLowerCase()
  if (t.endsWith('[]')) t = t.slice(0, -2)
  return t
})

const LENGTH_TYPES = ['varchar', 'character varying', 'char', 'character', 'bit', 'varbit', 'bit varying']
const PRECISION_SCALE_TYPES = ['numeric', 'decimal']
const PRECISION_TYPES = ['float', 'time', 'timetz', 'timestamp', 'timestamptz', 'interval',
  'time with time zone', 'timestamp without time zone',
  'time without time zone', 'timestamp with time zone']
const IDENTITY_TYPES = ['integer', 'bigint', 'smallint', 'serial', 'bigserial', 'smallserial']
const COLLATION_TYPES = ['varchar', 'character varying', 'char', 'character', 'text', 'name', 'citext']
const TOAST_TYPES = ['varchar', 'character varying', 'char', 'character', 'text', 'bytea',
  'jsonb', 'json', 'xml', 'tsvector']

const showLength = computed(() => LENGTH_TYPES.includes(baseType.value))
const showPrecisionScale = computed(() => PRECISION_SCALE_TYPES.includes(baseType.value))
const showPrecision = computed(() => PRECISION_TYPES.includes(baseType.value))
const showScale = computed(() => showPrecisionScale.value && props.column.precision > 0)
const showIdentity = computed(() => IDENTITY_TYPES.includes(baseType.value))
const showCollation = computed(() => COLLATION_TYPES.includes(baseType.value))
const isToastable = computed(() => TOAST_TYPES.includes(baseType.value) || props.column.type.endsWith('[]'))

function setSeqOpt(field: string, value: number | boolean) {
  const opt = { ...(props.column.identitySeqOpt || { start: 0, increment: 0, min: 0, max: 0, cache: 0, cycle: false }), [field]: value }
  emit('update', props.index, 'identitySeqOpt', opt)
}

const hasGenerated = computed(() => !!props.column.generated)
const hasIdentity = computed(() => !!props.column.identity)

const columnIndexes = computed(() =>
  (props.indexes || []).filter(ix => ix.columns?.some(c => c.name === props.column.name))
)
</script>

<template>
  <div class="cp-panel">
    <div class="cp-title">Column #{{ index + 1 }}</div>

    <!-- Type parameters -->
    <div v-if="showLength" class="cp-row">
      <label class="cp-label">Length</label>
      <input class="cp-input cp-num" type="number" :value="column.length || ''" @change="set('length', parseInt(($event.target as HTMLInputElement).value) || 0)" />
    </div>
    <div v-if="showPrecisionScale || showPrecision" class="cp-row">
      <label class="cp-label">Precision</label>
      <input class="cp-input cp-num" type="number" :value="column.precision || ''" @change="set('precision', parseInt(($event.target as HTMLInputElement).value) || 0)" />
    </div>
    <div v-if="showScale" class="cp-row">
      <label class="cp-label">Scale</label>
      <input class="cp-input cp-num" type="number" :value="column.scale || ''" @change="set('scale', parseInt(($event.target as HTMLInputElement).value) || 0)" />
    </div>

    <!-- Identity (only for integer types) -->
    <div v-if="showIdentity" class="cp-row">
      <label class="cp-label">Identity</label>
      <select class="cp-input" :value="column.identity || ''" :disabled="hasGenerated" @change="set('identity', ($event.target as HTMLSelectElement).value)">
        <option value="">(none)</option>
        <option value="by-default">by-default</option>
        <option value="always">always</option>
      </select>
    </div>

    <!-- Identity Sequence Options -->
    <template v-if="hasIdentity">
      <div class="cp-row">
        <label class="cp-label">Start</label>
        <input class="cp-input cp-num" type="number" :value="column.identitySeqOpt?.start || ''" @change="setSeqOpt('start', parseInt(($event.target as HTMLInputElement).value) || 0)" />
      </div>
      <div class="cp-row">
        <label class="cp-label">Increment</label>
        <input class="cp-input cp-num" type="number" :value="column.identitySeqOpt?.increment || ''" @change="setSeqOpt('increment', parseInt(($event.target as HTMLInputElement).value) || 0)" />
      </div>
      <div class="cp-row">
        <label class="cp-label">Cache</label>
        <input class="cp-input cp-num" type="number" :value="column.identitySeqOpt?.cache || ''" @change="setSeqOpt('cache', parseInt(($event.target as HTMLInputElement).value) || 0)" />
      </div>
      <div class="cp-row">
        <label class="cp-label"></label>
        <label class="cp-check"><input type="checkbox" :checked="column.identitySeqOpt?.cycle || false" @change="setSeqOpt('cycle', ($event.target as HTMLInputElement).checked)" /> Cycle</label>
      </div>
    </template>

    <!-- Default (hidden when identity or generated is set) -->
    <div v-if="!hasIdentity && !hasGenerated" class="cp-row">
      <label class="cp-label">Default</label>
      <input class="cp-input" :value="column.default || ''" @change="set('default', ($event.target as HTMLInputElement).value)" />
    </div>

    <!-- Collation (only for text types) -->
    <div v-if="showCollation" class="cp-row">
      <label class="cp-label">Collation</label>
      <input class="cp-input" :value="column.collation || ''" @change="set('collation', ($event.target as HTMLInputElement).value)" />
    </div>

    <!-- Generated -->
    <div class="cp-row">
      <label class="cp-label">Generated</label>
      <input class="cp-input" placeholder="expression" :value="column.generated || ''" :disabled="hasIdentity" @change="set('generated', ($event.target as HTMLInputElement).value)" />
    </div>
    <div v-if="column.generated" class="cp-row">
      <label class="cp-label"></label>
      <label class="cp-check"><input type="radio" :name="`gen-stored-${index}`" :checked="column.generatedStored !== false" @change="set('generatedStored', true)" /> Stored</label>
      <label class="cp-check"><input type="radio" :name="`gen-stored-${index}`" :checked="column.generatedStored === false" @change="set('generatedStored', false)" /> Virtual</label>
    </div>

    <!-- Storage (only for TOASTable types) -->
    <template v-if="isToastable">
      <div class="cp-row">
        <label class="cp-label">Compression</label>
        <select class="cp-input" :value="column.compression || ''" @change="set('compression', ($event.target as HTMLSelectElement).value)">
          <option value="">(default)</option>
          <option value="lz4">lz4</option>
          <option value="pglz">pglz</option>
        </select>
      </div>
      <div class="cp-row">
        <label class="cp-label">Storage</label>
        <select class="cp-input" :value="column.storage || ''" @change="set('storage', ($event.target as HTMLSelectElement).value)">
          <option value="">(auto)</option>
          <option value="plain">plain</option>
          <option value="external">external</option>
          <option value="extended">extended</option>
          <option value="main">main</option>
        </select>
      </div>
    </template>

    <!-- Comment (always) -->
    <div class="cp-row">
      <label class="cp-label">Comment</label>
      <input class="cp-input" :value="column.comment || ''" @change="set('comment', ($event.target as HTMLInputElement).value)" placeholder="column comment" />
    </div>

    <!-- Indexes -->
    <div class="cp-section">Indexes</div>
    <div v-for="ix in columnIndexes" :key="ix.name" class="cp-ix-link" @click="emit('goToIndex', ix.name)">
      {{ ix.name }} <span class="cp-ix-hint">{{ ix.using || 'btree' }}{{ ix.unique ? ', unique' : '' }}</span>
    </div>
    <div v-if="!column.pk" class="cp-ix-link cp-ix-add" @click="emit('createIndex', column.name)">+ Create Index</div>
    <div v-if="columnIndexes.length === 0 && column.pk" class="cp-ix-none">PK (implicit index)</div>
  </div>
</template>

<style scoped>
.cp-panel {
  padding: 0.615rem 0.923rem; font-size: 0.923rem; overflow-y: auto; height: 100%;
  color: var(--color-text-primary);
}
.cp-title {
  font-weight: 600; font-size: 0.846rem; color: var(--color-text-secondary);
  margin-bottom: 0.615rem; padding-bottom: 0.308rem; border-bottom: 1px solid var(--color-border);
}
.cp-row { display: flex; align-items: center; gap: 0.462rem; margin-bottom: 0.462rem; }
.cp-label { width: 5.385rem; font-size: 0.846rem; color: var(--color-text-secondary); flex-shrink: 0; }
.cp-input {
  flex: 1; padding: 1px 0.308rem; font-size: 0.923rem; height: 1.538rem;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-primary); outline: none;
}
.cp-input:focus { border-color: var(--color-accent); }
.cp-input:disabled { opacity: 0.5; }
.cp-num { width: 5.385rem; flex: none; }
.cp-check {
  font-size: 0.923rem; display: flex; align-items: center; gap: 0.308rem;
  cursor: pointer; color: var(--color-text-primary);
}
select.cp-input { cursor: pointer; }
.cp-section {
  font-weight: 600; font-size: 0.846rem; color: var(--color-text-secondary);
  margin-top: 0.769rem; margin-bottom: 0.385rem; padding-top: 0.462rem;
  border-top: 1px solid var(--color-border);
}
.cp-ix-link {
  font-size: 0.846rem; color: var(--color-accent); cursor: pointer;
  padding: 0.077rem 0; line-height: 1.308rem;
}
.cp-ix-link:hover { text-decoration: underline; }
.cp-ix-hint { color: var(--color-text-muted); font-size: 0.769rem; margin-left: 0.231rem; }
.cp-ix-add { margin-top: 0.231rem; }
.cp-ix-none { font-size: 0.769rem; color: var(--color-text-muted); font-style: italic; }
</style>
