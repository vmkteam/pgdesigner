<script setup lang="ts">
import { DialogRoot, DialogPortal, DialogOverlay, DialogContent, DialogTitle, DialogClose } from 'reka-ui'
import { shortcutsByContext, contextNames } from '@/shortcuts'

defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()

const groups = shortcutsByContext()
</script>

<template>
  <DialogRoot :open="open" @update:open="(v: boolean) => !v && emit('close')">
    <DialogPortal>
      <DialogOverlay class="kr-overlay" />
      <DialogContent class="kr-box" @escape-key-down="emit('close')">
        <div class="kr-header">
          <DialogTitle class="kr-title">Keyboard Reference</DialogTitle>
          <DialogClose class="kr-close" @click="emit('close')">✕</DialogClose>
        </div>
        <div class="kr-body">
          <div v-for="(shortcuts, ctx) in groups" :key="ctx" class="kr-group">
            <div class="kr-group-title">{{ contextNames[ctx] || ctx }}</div>
            <div v-for="s in shortcuts" :key="s.key + s.action" class="kr-row">
              <span class="kr-key">{{ s.key }}</span>
              <span class="kr-action">{{ s.action }}</span>
            </div>
          </div>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<style>
.kr-overlay { position: fixed; inset: 0; background: rgba(0,0,0,.2); z-index: 60; }
.kr-box {
  position: fixed; z-index: 70; top: 15%; left: 50%; transform: translateX(-50%);
  width: 48rem; max-height: 70vh;
  background: var(--color-bg-surface); border: 1px solid var(--color-menu-border);
  display: flex; flex-direction: column; box-shadow: 0 4px 12px rgba(0,0,0,.2);
}
.kr-header {
  height: 2.154rem; background: var(--color-bg-app); border-bottom: 1px solid var(--color-border);
  display: flex; align-items: center; padding: 0 0.923rem; flex-shrink: 0;
  color: var(--color-text-primary);
}
.kr-title { font-size: 0.923rem; font-weight: 600; flex: 1; }
.kr-close {
  width: 1.538rem; height: 1.538rem; display: flex; align-items: center; justify-content: center;
  color: var(--color-text-secondary); font-size: 1.077rem; cursor: default;
}
.kr-close:hover { background: var(--color-bg-hover); }
.kr-body { padding: 0.615rem 0.923rem; columns: 2; column-gap: 1.538rem; }
.kr-group { margin-bottom: 0.923rem; break-inside: avoid; }
.kr-group-title {
  font-size: 0.769rem; font-weight: 600; color: var(--color-text-muted);
  text-transform: uppercase; letter-spacing: 0.05em;
  margin-bottom: 0.308rem; padding-bottom: 0.231rem; border-bottom: 1px solid var(--color-border-subtle);
}
.kr-row { display: flex; align-items: center; padding: 0.154rem 0; font-size: 0.846rem; }
.kr-key {
  width: 9rem; flex-shrink: 0; font-family: monospace; font-size: 0.769rem;
  color: var(--color-text-primary); font-weight: 600;
}
.kr-action { color: var(--color-text-secondary); }
</style>
