<script setup lang="ts">
import { computed } from 'vue'
import { useProjectStore } from '@/stores/project'
import { useUiStore } from '@/stores/ui'
import { useCanvasStore } from '@/stores/canvas'
import {
  MenubarRoot,
  MenubarMenu,
  MenubarTrigger,
  MenubarPortal,
  MenubarContent,
  MenubarItem,
  MenubarCheckboxItem,
  MenubarItemIndicator,
  MenubarSeparator,
} from 'reka-ui'

import { useFileActions } from '@/composables/useFileActions'

const store = useProjectStore()
const ui = useUiStore()
const canvas = useCanvasStore()
const { fileNew, fileOpen, fileSaveAs, fileClose } = useFileActions()

const isMac = navigator.platform.includes('Mac')
const mod = isMac ? '⌘' : 'Ctrl+'

interface MenuItem {
  label: string
  action?: () => void
  shortcut?: string
  separator?: boolean
  disabled?: boolean
  checkbox?: boolean
  checked?: boolean
}

async function exportCanvas(format: 'png' | 'svg') {
  const container = document.querySelector('.vue-flow') as HTMLElement
  if (!container) return

  const viewport = container.querySelector('.vue-flow__viewport') as HTMLElement
  if (!viewport) return

  ui.exporting = true
  await new Promise(r => setTimeout(r, 50))

  // Compute bounding box of all nodes
  const nodeEls = container.querySelectorAll('.vue-flow__node')
  if (!nodeEls.length) { ui.exporting = false; return }

  let minX = Infinity, minY = Infinity, maxX = -Infinity, maxY = -Infinity
  for (const node of nodeEls) {
    const el = node as HTMLElement
    const x = parseFloat(el.style.transform?.match(/translate\(([^,]+)px/)?.[1] || el.style.left || '0')
    const y = parseFloat(el.style.transform?.match(/,\s*([^)]+)px/)?.[1] || el.style.top || '0')
    const w = el.offsetWidth
    const h = el.offsetHeight
    minX = Math.min(minX, x)
    minY = Math.min(minY, y)
    maxX = Math.max(maxX, x + w)
    maxY = Math.max(maxY, y + h)
  }

  const pad = 40
  const width = maxX - minX + pad * 2
  const height = maxY - minY + pad * 2

  // Save original transform and container size
  const origTransform = viewport.style.transform
  const origWidth = container.style.width
  const origHeight = container.style.height
  const origOverflow = container.style.overflow

  try {
    // Set viewport to identity transform offset by bbox origin
    viewport.style.transform = `translate(${-minX + pad}px, ${-minY + pad}px) scale(1)`
    container.style.width = `${width}px`
    container.style.height = `${height}px`
    container.style.overflow = 'visible'

    // Wait for layout
    await new Promise(r => setTimeout(r, 100))

    const { toPng, toSvg } = await import('html-to-image')
    const fn = format === 'png' ? toPng : toSvg
    const bg = getComputedStyle(container).backgroundColor || '#ffffff'

    const dataUrl = await fn(container, {
      width, height, backgroundColor: bg,
      filter: (node: HTMLElement) => {
        if (node.classList?.contains('vue-flow__minimap')) return false
        if (node.classList?.contains('vue-flow__controls')) return false
        if (node.classList?.contains('vue-flow__panel')) return false
        return true
      },
    })

    const link = document.createElement('a')
    link.download = `diagram.${format}`
    link.href = dataUrl
    link.click()
  } finally {
    // Restore
    viewport.style.transform = origTransform
    container.style.width = origWidth
    container.style.height = origHeight
    container.style.overflow = origOverflow
    ui.exporting = false
  }
}

interface Menu {
  label: string
  items: MenuItem[]
}

const menus = computed<Menu[]>(() => [
  {
    label: 'File',
    items: [
      { label: 'New', shortcut: mod + 'N', action: fileNew },
      { label: 'Open...', shortcut: mod + 'O', action: fileOpen },
      { label: 'Save', shortcut: mod + 'S', action: () => store.saveProject(), disabled: store.autoSave || !store.info?.filePath },
      { label: 'Save As...', shortcut: mod + '⇧S', action: fileSaveAs, disabled: ui.isWelcome },
      { label: '', separator: true },
      { label: 'Auto Save', checkbox: true, checked: store.autoSave, action: () => store.toggleAutoSave() },
      { label: '', separator: true },
      { label: ui.exporting ? 'Exporting...' : 'Export PNG...', action: () => exportCanvas('png'), disabled: ui.exporting },
      { label: ui.exporting ? 'Exporting...' : 'Export SVG...', action: () => exportCanvas('svg'), disabled: ui.exporting },
      { label: '', separator: true },
      { label: 'Close', shortcut: mod + 'W', action: fileClose },
    ],
  },
  {
    label: 'Edit',
    items: [
      { label: 'Undo', disabled: true },
      { label: 'Redo', disabled: true },
      { label: '', separator: true },
      { label: 'Select All', disabled: ui.isWelcome },
      { label: 'Go To...', shortcut: mod + 'F', action: () => { ui.goToOpen = true }, disabled: ui.isWelcome },
    ],
  },
  {
    label: 'View',
    items: [
      { label: 'Zoom In', shortcut: mod + '+', action: () => canvas.zoomIn(), disabled: ui.isWelcome },
      { label: 'Zoom Out', shortcut: mod + '−', action: () => canvas.zoomOut(), disabled: ui.isWelcome },
      { label: 'Fit to Screen', action: () => canvas.fitToScreen(), disabled: ui.isWelcome },
      { label: 'Actual Size', shortcut: mod + '0', action: () => canvas.resetZoom(), disabled: ui.isWelcome },
      { label: '', separator: true },
      { label: 'Dark Theme', shortcut: mod + '⇧D', checkbox: true, checked: ui.theme === 'dark', action: () => ui.toggleTheme() },
    ],
  },
  {
    label: 'Diagram',
    items: [
      {
        label: 'Check Diagram',
        shortcut: mod + 'L',
        disabled: ui.isWelcome,
        action: () => {
          store.loadLint()
          ui.openLint()
        },
      },
      { label: '', separator: true },
      { label: 'Auto Layout', action: () => canvas.autoLayout(), disabled: ui.isWelcome },
      { label: 'Cluster Tables', action: () => canvas.clusterTables(), disabled: ui.isWelcome },
      { label: 'Fix Overlaps', action: () => canvas.fixOverlaps(), disabled: ui.isWelcome },
    ],
  },
  {
    label: 'Database',
    items: [
      {
        label: 'Generate DDL...',
        shortcut: mod + 'G',
        disabled: ui.isWelcome,
        action: () => {
          store.loadDDL()
          ui.openDDL()
        },
      },
      {
        label: 'Generate Test Data...',
        disabled: ui.isWelcome,
        action: () => ui.openTestData(),
      },
      {
        label: 'Unsaved Changes (Diff)...',
        shortcut: mod + 'D',
        disabled: ui.isWelcome,
        action: () => ui.openDiff(),
      },
      { label: '', separator: true },
      { label: 'Project Settings...', shortcut: mod + ',', action: () => { ui.settingsOpen = true } },
    ],
  },
  {
    label: 'Help',
    items: [
      { label: 'Keyboard Reference', shortcut: '?', action: () => { ui.keyboardRefOpen = true } },
      { label: '', separator: true },
      { label: 'About PgDesigner', action: () => { ui.aboutOpen = true } },
    ],
  },
])
</script>

<template>
  <MenubarRoot class="menubar">
    <MenubarMenu v-for="menu in menus" :key="menu.label">
      <MenubarTrigger class="menubar-trigger">
        {{ menu.label }}
      </MenubarTrigger>
      <MenubarPortal>
        <MenubarContent class="menubar-content" :side-offset="2" :align="'start'">
          <template v-for="(item, idx) in menu.items" :key="idx">
            <MenubarSeparator v-if="item.separator" class="menubar-sep" />
            <MenubarCheckboxItem
              v-else-if="item.checkbox"
              class="menubar-item"
              :model-value="item.checked"
              @select="item.action?.()"
            >
              <MenubarItemIndicator class="menubar-indicator">✓</MenubarItemIndicator>
              <span>{{ item.label }}</span>
              <span v-if="item.shortcut" class="menubar-shortcut">{{ item.shortcut }}</span>
            </MenubarCheckboxItem>
            <MenubarItem
              v-else
              class="menubar-item"
              :class="item.disabled ? 'disabled' : ''"
              :disabled="item.disabled"
              @select="item.action?.()"
            >
              <span>{{ item.label }}</span>
              <span v-if="item.shortcut" class="menubar-shortcut">{{ item.shortcut }}</span>
            </MenubarItem>
          </template>
        </MenubarContent>
      </MenubarPortal>
    </MenubarMenu>
  </MenubarRoot>
</template>

<style>
/* Not scoped — MenubarContent renders via Portal (Teleport) outside this component */
.menubar {
  height: 1.846rem; background: var(--color-bg-app); border-bottom: 1px solid var(--color-border);
  display: flex; align-items: center; padding: 0 0.308rem; flex-shrink: 0; user-select: none;
}
.menubar-trigger {
  padding: 0 0.615rem; height: 100%; font-size: 0.923rem; cursor: default;
}
.menubar-trigger:hover, .menubar-trigger[data-state="open"] {
  background: var(--color-bg-hover);
}
.menubar-content {
  min-width: 12.308rem; background: var(--color-menu-bg); border: 1px solid var(--color-menu-border);
  padding: 2px 0; z-index: 50; box-shadow: 0 2px 4px rgba(0,0,0,.15);
}
.menubar-sep { height: 1px; background: var(--color-border); margin: 0.154rem 0; }
.menubar-item {
  padding: 0.154rem 0.923rem; font-size: 0.923rem; cursor: default; user-select: none; outline: none;
  color: var(--color-text-primary);
  display: flex; align-items: center; justify-content: space-between; gap: 2rem;
}
.menubar-shortcut {
  font-size: 0.769rem; color: var(--color-text-muted); flex-shrink: 0;
}
.menubar-item[data-highlighted] { background: var(--color-bg-hover); }
.menubar-item.disabled { color: var(--color-text-disabled); }
.menubar-item[role="menuitemcheckbox"] { padding-left: 1.846rem; position: relative; }
.menubar-indicator { position: absolute; left: 0.615rem; }
</style>
