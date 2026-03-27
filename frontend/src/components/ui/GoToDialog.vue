<script setup lang="ts">
import { ref, computed, watch, nextTick, useTemplateRef } from 'vue'
import { whenever } from '@vueuse/core'
import { useCanvasStore } from '@/stores/canvas'
import { useUiStore } from '@/stores/ui'
import api from '@/api/factory'
import type { IObjectItem } from '@/api/factory'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()

const canvas = useCanvasStore()
const ui = useUiStore()

const kindTabMap: Record<string, string> = {
  column: 'columns',
  pk: 'constraints', unique: 'constraints', check: 'constraints',
  index: 'indexes',
  fk: 'fk',
}

const query = ref('')
const selectedIndex = ref(0)
const inputRef = useTemplateRef<HTMLInputElement>('inputRef')
const allItems = ref<IObjectItem[]>([])

whenever(
  () => props.open,
  async () => {
    query.value = ''
    selectedIndex.value = 0
    allItems.value = (await api.project.listObjects()) || []
    nextTick(() => inputRef.value?.focus())
  },
)

watch(query, () => {
  selectedIndex.value = 0
})

const filtered = computed(() => {
  const q = query.value.toLowerCase()
  if (!q) return allItems.value.filter((i) => i.kind === 'table')
  return allItems.value.filter((i) => i.name.toLowerCase().includes(q)).slice(0, 50)
})

function select(item: IObjectItem) {
  const tab = kindTabMap[item.kind]
  if (tab && item.table) {
    // item.name is "table.object" — extract just the object name
    const dot = item.name.indexOf('.')
    const objectName = dot >= 0 ? item.name.substring(dot + 1) : item.name
    canvas.focusNode(item.table)
    if (item.kind === 'fk') {
      ui.openTableEditor(item.table, tab, objectName)
    } else {
      ui.openTableEditor(item.table, tab, undefined, objectName)
    }
  } else {
    canvas.focusNode(item.table || item.name)
  }
  emit('close')
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    selectedIndex.value = Math.min(selectedIndex.value + 1, filtered.value.length - 1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
  } else if (e.key === 'Enter') {
    e.preventDefault()
    const item = filtered.value[selectedIndex.value]
    if (item) select(item)
  } else if (e.key === 'Escape') {
    emit('close')
  }
}

const kindIcons: Record<string, string> = {
  table: 'T', column: 'C', index: 'I', fk: 'F', pk: 'P', unique: 'U',
  check: 'K', trigger: 'G', sequence: 'S', view: 'V', matview: 'M',
  function: 'f', extension: 'X', domain: 'D', enum: 'E', composite: 'R',
}

const kindColors: Record<string, string> = {
  table: 'text-blue-600 bg-blue-100',
  column: 'text-green-700 bg-green-100',
  index: 'text-amber-700 bg-amber-100',
  fk: 'text-purple-700 bg-purple-100',
  pk: 'text-red-600 bg-red-100',
  unique: 'text-teal-700 bg-teal-100',
  check: 'text-orange-700 bg-orange-100',
  trigger: 'text-pink-700 bg-pink-100',
  sequence: 'text-cyan-700 bg-cyan-100',
  view: 'text-indigo-700 bg-indigo-100',
  matview: 'text-indigo-700 bg-indigo-100',
  function: 'text-violet-700 bg-violet-100',
  extension: 'text-gray-700 bg-gray-200',
  domain: 'text-lime-700 bg-lime-100',
  enum: 'text-emerald-700 bg-emerald-100',
  composite: 'text-sky-700 bg-sky-100',
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50" @click.self="$emit('close')">
      <div
        class="goto-box"
      >
        <div class="goto-input-wrap">
          <input
            ref="inputRef"
            v-model="query"
            class="goto-input"
            placeholder="Go to table, column, index, FK, trigger, sequence..."
            @keydown="onKeydown"
          />
        </div>
        <div class="flex-1 overflow-auto min-h-0">
          <div
            v-for="(item, idx) in filtered"
            :key="item.name + item.kind"
            class="flex items-center gap-2 px-2 py-0.5 text-xs cursor-pointer"
            :class="idx === selectedIndex ? 'goto-selected' : 'goto-hover'"
            @click="select(item)"
            @mouseenter="selectedIndex = idx"
          >
            <span
              class="w-4 h-4 flex items-center justify-center text-[10px] font-bold shrink-0"
              :class="kindColors[item.kind] || 'goto-kind-default'"
            >
              {{ kindIcons[item.kind] || '?' }}
            </span>
            <span class="truncate">{{ item.name }}</span>
            <span class="goto-kind-label">{{ item.kind }}</span>
          </div>
          <div v-if="!filtered.length" class="goto-empty">No results</div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.goto-box {
  position: fixed; top: 15%; left: 50%; transform: translateX(-50%); width: 36.923rem;
  background: var(--color-bg-surface); border: 1px solid var(--color-menu-border);
  box-shadow: 0 4px 12px rgba(0,0,0,.2); display: flex; flex-direction: column; max-height: 60vh;
  color: var(--color-text-primary);
}
.goto-input-wrap { border-bottom: 1px solid var(--color-border); padding: 0.308rem; }
.goto-input {
  width: 100%; padding: 0.308rem 0.615rem; font-size: 0.923rem;
  border: 1px solid var(--color-border); outline: none;
  background: var(--color-bg-surface); color: var(--color-text-primary);
}
.goto-input:focus { border-color: var(--color-accent); }
.goto-selected { background: var(--color-bg-hover); }
.goto-hover:hover { background: var(--color-bg-app); }
.goto-kind-default { color: var(--color-text-secondary); background: var(--color-bg-app); }
.goto-kind-label { margin-left: auto; font-size: 0.769rem; color: var(--color-text-muted); }
.goto-empty { padding: 0.615rem; font-size: 0.923rem; color: var(--color-text-muted); }
</style>
