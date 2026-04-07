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
  updateExclude: [index: number, data: IExcludeDetail]
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

function setExclude(field: string, value: string) {
  if (props.item.kind !== 'exclude') return
  emit('updateExclude', props.item.index, { ...props.item.data, [field]: value })
}

function updateExclElement(i: number, field: string, value: string) {
  if (props.item.kind !== 'exclude') return
  const elements = [...props.item.data.elements]
  elements[i] = { ...elements[i]!, [field]: value }
  emit('updateExclude', props.item.index, { ...props.item.data, elements })
}

function addExclElement() {
  if (props.item.kind !== 'exclude') return
  const elements = [...props.item.data.elements, { column: '', expression: '', opclass: '', with: '=' }]
  emit('updateExclude', props.item.index, { ...props.item.data, elements })
}

function removeExclElement(i: number) {
  if (props.item.kind !== 'exclude') return
  const elements = props.item.data.elements.filter((_, j) => j !== i)
  emit('updateExclude', props.item.index, { ...props.item.data, elements })
}

function toggleExclElementMode(i: number) {
  if (props.item.kind !== 'exclude') return
  const elements = [...props.item.data.elements]
  const el = elements[i]!
  if (el.expression) {
    elements[i] = { column: el.column, expression: '', opclass: el.opclass, with: el.with }
  } else {
    elements[i] = { column: '', expression: el.column || ' ', opclass: el.opclass, with: el.with }
  }
  emit('updateExclude', props.item.index, { ...props.item.data, elements })
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

    <!-- Exclude -->
    <template v-else-if="item.kind === 'exclude'">
      <div class="cp-title">Exclude Constraint</div>
      <div class="cp-row">
        <label class="cp-label">Name</label>
        <input class="cp-input" :value="item.data.name" @change="setExclude('name', ($event.target as HTMLInputElement).value)" />
      </div>
      <div class="cp-row">
        <label class="cp-label">Using</label>
        <select class="cp-input" :value="item.data.using || 'gist'" @change="setExclude('using', ($event.target as HTMLSelectElement).value)">
          <option value="gist">gist</option>
          <option value="spgist">spgist</option>
          <option value="btree">btree</option>
          <option value="hash">hash</option>
        </select>
      </div>

      <div class="cp-group-label">Elements</div>
      <div v-for="(el, i) in item.data.elements" :key="i" class="cp-elem">
        <div class="cp-elem-header">
          <label class="cp-check">
            <input type="radio" :name="`excl-mode-${i}`" :checked="!el.expression" @change="toggleExclElementMode(i)" /> Column
          </label>
          <label class="cp-check">
            <input type="radio" :name="`excl-mode-${i}`" :checked="!!el.expression" @change="toggleExclElementMode(i)" /> Expression
          </label>
          <button class="cp-btn-del" @click="removeExclElement(i)">×</button>
        </div>
        <div v-if="!el.expression" class="cp-row">
          <select class="cp-input" :value="el.column" @change="updateExclElement(i, 'column', ($event.target as HTMLSelectElement).value)">
            <option value="">(select)</option>
            <option v-for="c in columns" :key="c" :value="c">{{ c }}</option>
          </select>
        </div>
        <div v-else class="cp-row">
          <input class="cp-input cp-mono" :value="el.expression" placeholder="SQL expression" @change="updateExclElement(i, 'expression', ($event.target as HTMLInputElement).value)" />
        </div>
        <div class="cp-row">
          <label class="cp-label-sm">Opclass</label>
          <input class="cp-input" :value="el.opclass" placeholder="opclass" @change="updateExclElement(i, 'opclass', ($event.target as HTMLInputElement).value)" />
        </div>
        <div class="cp-row">
          <label class="cp-label-sm">WITH</label>
          <input class="cp-input cp-with" :value="el.with" placeholder="=" @change="updateExclElement(i, 'with', ($event.target as HTMLInputElement).value)" />
        </div>
      </div>
      <button class="cp-btn-add" @click="addExclElement">+ Add element</button>

      <div class="cp-group-label">Where</div>
      <div class="cp-row">
        <textarea class="cp-textarea cp-mono" :value="item.data.where || ''" rows="2" placeholder="partial constraint predicate" @change="setExclude('where', ($event.target as HTMLTextAreaElement).value)" />
      </div>
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
.cp-mono { font-family: monospace; font-size: 0.846rem; }
.cp-elem { margin-bottom: 0.462rem; padding: 0.308rem; border: 1px solid var(--color-border-subtle); border-radius: 2px; }
.cp-elem-header { display: flex; align-items: center; gap: 0.462rem; margin-bottom: 0.308rem; }
.cp-label-sm { width: 2.5rem; font-size: 0.846rem; color: var(--color-text-secondary); flex-shrink: 0; }
.cp-with { max-width: 4rem; }
.cp-btn-del {
  width: 1.385rem; height: 1.385rem; font-size: 0.846rem; margin-left: auto;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-secondary); cursor: pointer; display: flex; align-items: center; justify-content: center;
}
.cp-btn-del:hover { background: var(--color-bg-hover); color: #cc3333; }
.cp-btn-add { font-size: 0.769rem; color: var(--color-accent); background: none; border: none; cursor: pointer; padding: 0; margin-bottom: 0.462rem; }
.cp-btn-add:hover { text-decoration: underline; }
select.cp-input { cursor: pointer; }
</style>
