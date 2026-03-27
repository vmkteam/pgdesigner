<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick, useTemplateRef } from 'vue'
import { useElementBounding } from '@vueuse/core'
import api from '@/api/factory'
import type { ITypeInfo } from '@/api/factory'

const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{
  'update:modelValue': [value: string]
  commit: []
  cancel: []
}>()

const query = ref(props.modelValue)
const allTypes = ref<ITypeInfo[]>([])
const showDropdown = ref(true)
const selectedIdx = ref(0)
const inputRef = useTemplateRef<HTMLInputElement>('inputRef')
const preventBlur = ref(false)
const { bottom: dropdownTop, left: dropdownLeft, width: inputWidth } = useElementBounding(inputRef)
const dropdownWidth = computed(() => Math.max(inputWidth.value, 220))

const filtered = computed(() => {
  const q = query.value.toLowerCase()
  if (!q) return allTypes.value.slice(0, 20)
  return allTypes.value.filter(t => t.name.toLowerCase().includes(q)).slice(0, 20)
})

watch(query, () => { selectedIdx.value = 0; showDropdown.value = true })

onMounted(async () => {
  try { allTypes.value = await api.project.listTypes() } catch { /* ignore */ }
  await nextTick()
  inputRef.value?.focus()
  inputRef.value?.select()
})

function select(name: string) {
  query.value = name
  emit('update:modelValue', name)
  showDropdown.value = false
  emit('commit')
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    selectedIdx.value = Math.min(selectedIdx.value + 1, filtered.value.length - 1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    selectedIdx.value = Math.max(selectedIdx.value - 1, 0)
  } else if (e.key === 'Enter' || e.key === 'Tab') {
    e.preventDefault()
    e.stopPropagation() // prevent grid from re-opening edit on same key
    if (filtered.value.length > 0 && showDropdown.value) {
      select(filtered.value[selectedIdx.value]!.name)
    } else {
      emit('update:modelValue', query.value)
      emit('commit')
    }
  } else if (e.key === 'Escape') {
    e.stopPropagation()
    emit('cancel')
  }
}

function onBlur() {
  if (preventBlur.value) {
    preventBlur.value = false
    nextTick(() => inputRef.value?.focus())
    return
  }
  emit('update:modelValue', query.value)
  emit('commit')
}

function onItemMousedown() {
  preventBlur.value = true
}

function onItemClick(name: string) {
  preventBlur.value = false
  select(name)
}

function categoryBadge(cat: string) {
  const map: Record<string, string> = {
    numeric: 'N', character: 'T', datetime: 'D', boolean: 'B',
    json: 'J', network: 'Net', geometric: 'G', search: 'S',
    array: '[]', enum: 'E', composite: 'C', domain: 'Dom',
  }
  return map[cat] || '?'
}
</script>

<template>
  <div>
    <input
      ref="inputRef"
      v-model="query"
      class="ta-input"
      @keydown="onKeydown"
      @blur="onBlur"
      @focus="showDropdown = true"
    />
    <div
      v-if="showDropdown && filtered.length"
      class="ta-dropdown"
      :style="{ top: dropdownTop + 'px', left: dropdownLeft + 'px', width: dropdownWidth + 'px' }"
    >
      <div
        v-for="(t, idx) in filtered" :key="t.name"
        class="ta-item" :class="{ selected: idx === selectedIdx }"
        @mousedown="onItemMousedown"
        @click="onItemClick(t.name)"
        @mouseenter="selectedIdx = idx"
      >
        <span class="ta-badge">{{ categoryBadge(t.category) }}</span>
        <span class="ta-name">{{ t.name }}</span>
        <span class="ta-cat">{{ t.category }}</span>
      </div>
    </div>
  </div>
</template>

<style>
.ta-input {
  width: 100%; padding: 0 0.231rem; font-size: 0.923rem; line-height: 1.231rem; height: 1.385rem;
  border: 1px solid var(--color-accent); box-sizing: border-box;
  background: var(--color-bg-surface); color: var(--color-text-primary); outline: none;
}
.ta-dropdown {
  position: fixed; z-index: 200;
  background: var(--color-menu-bg); border: 1px solid var(--color-menu-border);
  box-shadow: 0 2px 8px rgba(0,0,0,.15); max-height: 15.385rem; overflow-y: auto;
}
.ta-item {
  display: flex; align-items: center; gap: 0.462rem;
  padding: 0.154rem 0.462rem; font-size: 0.923rem; cursor: default;
  color: var(--color-text-primary);
}
.ta-item.selected { background: var(--color-bg-hover); }
.ta-badge {
  width: 1.692rem; text-align: center; font-size: 0.769rem; font-weight: 600;
  color: var(--color-text-secondary); background: var(--color-bg-app);
  padding: 0 0.154rem;
}
.ta-name { flex: 1; }
.ta-cat { font-size: 0.769rem; color: var(--color-text-muted); }
</style>
