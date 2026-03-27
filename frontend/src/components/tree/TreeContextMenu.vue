<script setup lang="ts">
import { ref, nextTick, onBeforeUnmount } from 'vue'

export interface ContextMenuItem {
  label?: string
  shortcut?: string
  action?: () => void
  disabled?: boolean
  separator?: boolean
  children?: ContextMenuItem[]
}

const visible = ref(false)
const x = ref(0)
const y = ref(0)
const items = ref<ContextMenuItem[]>([])
const submenuIndex = ref(-1)

function show(event: MouseEvent, menuItems: ContextMenuItem[]) {
  if (visible.value) hide()
  items.value = menuItems
  x.value = event.clientX
  y.value = event.clientY
  submenuIndex.value = -1
  visible.value = true
  nextTick(() => {
    document.addEventListener('click', handleClickOutside, true)
    document.addEventListener('contextmenu', handleClickOutside, true)
    document.addEventListener('keydown', handleKeydown)
  })
}

function hide() {
  visible.value = false
  submenuIndex.value = -1
  document.removeEventListener('click', handleClickOutside, true)
  document.removeEventListener('contextmenu', handleClickOutside, true)
  document.removeEventListener('keydown', handleKeydown)
}

function handleClickOutside() {
  hide()
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.preventDefault()
    hide()
  }
}

function onItemClick(item: ContextMenuItem) {
  if (item.disabled || item.separator || item.children) return
  hide()
  item.action?.()
}

function onItemEnter(index: number, item: ContextMenuItem) {
  submenuIndex.value = item.children?.length ? index : -1
}

onBeforeUnmount(() => hide())

defineExpose({ show, hide })
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="tcm-backdrop">
      <div class="tcm-menu" :style="{ left: x + 'px', top: y + 'px' }">
        <template v-for="(item, i) in items" :key="i">
          <div v-if="item.separator" class="tcm-sep" />
          <div
            v-else
            class="tcm-item"
            :class="{ 'tcm-disabled': item.disabled, 'tcm-has-children': item.children?.length }"
            @click="onItemClick(item)"
            @mouseenter="onItemEnter(i, item)"
          >
            <span class="tcm-label">{{ item.label }}</span>
            <span v-if="item.shortcut" class="tcm-shortcut">{{ item.shortcut }}</span>
            <span v-if="item.children?.length" class="tcm-arrow">&#9656;</span>

            <!-- Submenu -->
            <div v-if="item.children?.length && submenuIndex === i" class="tcm-submenu">
              <div
                v-for="(child, j) in item.children" :key="j"
                class="tcm-item" :class="{ 'tcm-disabled': child.disabled }"
                @click.stop="onItemClick(child)"
              >
                <span class="tcm-label">{{ child.label }}</span>
              </div>
            </div>
          </div>
        </template>
      </div>
    </div>
  </Teleport>
</template>

<style>
.tcm-backdrop { position: fixed; inset: 0; z-index: 9999; }
.tcm-menu {
  position: fixed;
  min-width: 12rem;
  background: var(--color-bg-surface); border: 1px solid var(--color-menu-border);
  box-shadow: 0 2px 8px rgba(0,0,0,.2); padding: 0.231rem 0;
}
.tcm-item {
  display: flex; align-items: center; gap: 0.462rem;
  padding: 0.308rem 0.769rem; font-size: 0.846rem;
  color: var(--color-text-primary); cursor: default;
  position: relative;
}
.tcm-item:hover:not(.tcm-disabled) { background: var(--color-bg-hover); }
.tcm-disabled { opacity: 0.4; cursor: default; }
.tcm-label { flex: 1; }
.tcm-shortcut { font-size: 0.692rem; color: var(--color-text-muted); }
.tcm-arrow { font-size: 0.692rem; color: var(--color-text-muted); }
.tcm-sep { height: 1px; margin: 0.231rem 0.462rem; background: var(--color-border-subtle); }

.tcm-submenu {
  position: absolute; left: 100%; top: -0.231rem;
  min-width: 10rem;
  background: var(--color-bg-surface); border: 1px solid var(--color-menu-border);
  box-shadow: 0 2px 8px rgba(0,0,0,.2); padding: 0.231rem 0;
}
</style>
