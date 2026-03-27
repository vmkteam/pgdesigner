<script setup lang="ts">
import { nextTick, watch, ref, computed, useTemplateRef } from 'vue'
import { DialogRoot, DialogPortal, DialogOverlay, DialogContent, DialogTitle } from 'reka-ui'
import { useAppDialogState } from '@/composables/useAppDialog'
import { identifierError } from '@/composables/useIdentifierValidation'

const { visible, state, inputValue, close } = useAppDialogState()
const inputRef = useTemplateRef<HTMLInputElement>('inputRef')
const dirty = ref(false)

watch(visible, (v) => {
  if (v && state.value?.mode === 'prompt') {
    dirty.value = false
    nextTick(() => {
      inputRef.value?.focus()
      inputRef.value?.select()
    })
  }
})

// Validate identifier for prompt dialogs (only after user starts typing, skip if flagged)
const promptError = computed(() => {
  if (state.value?.mode !== 'prompt') return null
  if (state.value?.skipValidation) return null
  if (!dirty.value) return null
  return identifierError(inputValue.value)
})

function onOk() {
  if (!state.value) return
  if (state.value.mode === 'prompt' && !state.value.skipValidation) {
    dirty.value = true
    if (identifierError(inputValue.value)) return
  }
  if (state.value.mode === 'confirm') close(true)
  else if (state.value.mode === 'confirmSave') close('save')
  else if (state.value.mode === 'prompt') close(inputValue.value.trim() || null)
  else close(undefined)
}

function onDiscard() { close('discard') }

function onCancel() {
  if (!state.value) return
  if (state.value.mode === 'confirm') close(false)
  else if (state.value.mode === 'confirmSave') close('cancel')
  else if (state.value.mode === 'prompt') close(null)
  else close(undefined)
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') { e.preventDefault(); onOk() }
}
</script>

<template>
  <DialogRoot :open="visible" @update:open="(v: boolean) => !v && onCancel()">
    <DialogPortal>
      <DialogOverlay class="ad-overlay" />
      <DialogContent class="ad-box" @escape-key-down.prevent="onCancel" @keydown="onKeydown">
        <div class="ad-header">
          <DialogTitle class="ad-title">{{ state?.title }}</DialogTitle>
        </div>

        <div class="ad-body">
          <p class="ad-message">{{ state?.message }}</p>
          <input
            v-if="state?.mode === 'prompt'"
            ref="inputRef"
            v-model="inputValue"
            class="ad-input"
            :class="{ 'ad-input-error': promptError }"
            :maxlength="state?.skipValidation ? undefined : 63"
            :placeholder="state?.placeholder || ''"
            @input="dirty = true"
          />
          <div v-if="promptError" class="ad-error">{{ promptError }}</div>
        </div>

        <div class="ad-footer">
          <template v-if="state?.mode === 'confirmSave'">
            <button class="ad-btn" @click="onDiscard">Don't Save</button>
            <span class="ad-spacer" />
            <button class="ad-btn" @click="onCancel">Cancel</button>
            <button class="ad-btn primary" @click="onOk">Save</button>
          </template>
          <template v-else>
            <button class="ad-btn primary" :disabled="!!promptError" @click="onOk">OK</button>
            <button v-if="state?.mode !== 'alert'" class="ad-btn" @click="onCancel">Cancel</button>
          </template>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<style>
.ad-overlay { position: fixed; inset: 0; background: rgba(0,0,0,.3); z-index: 60; }
.ad-box {
  position: fixed; z-index: 70;
  top: 50%; left: 50%; transform: translate(-50%, -50%);
  min-width: 20rem; max-width: 30rem;
  background: var(--color-bg-surface); border: 1px solid var(--color-menu-border);
  box-shadow: 0 4px 16px rgba(0,0,0,.25);
  display: flex; flex-direction: column;
}
.ad-header {
  padding: 0.615rem 0.923rem;
  background: var(--color-bg-app); border-bottom: 1px solid var(--color-border);
  user-select: none;
}
.ad-title { font-size: 0.923rem; font-weight: 600; color: var(--color-text-primary); }
.ad-body { padding: 0.923rem; }
.ad-message { font-size: 0.923rem; color: var(--color-text-primary); margin: 0 0 0.615rem; white-space: pre-wrap; }
.ad-input {
  width: 100%; padding: 0.308rem 0.462rem; font-size: 0.923rem;
  border: 1px solid var(--color-border); background: var(--color-bg-surface);
  color: var(--color-text-primary); outline: none; box-sizing: border-box;
}
.ad-input:focus { border-color: var(--color-accent); }
.ad-footer {
  padding: 0.462rem 0.923rem;
  background: var(--color-bg-app); border-top: 1px solid var(--color-border);
  display: flex; justify-content: flex-end; gap: 0.308rem;
}
.ad-btn {
  padding: 0.231rem 0.923rem; font-size: 0.923rem;
  border: 1px solid var(--color-menu-border); background: var(--color-bg-surface);
  color: var(--color-text-primary); cursor: default;
}
.ad-btn:hover:not(:disabled) { background: var(--color-bg-hover); }
.ad-btn:disabled { opacity: 0.5; }
.ad-btn.primary { font-weight: 600; }
.ad-spacer { flex: 1; }
.ad-input-error { border-color: #cc3333 !important; }
.ad-error { font-size: 0.769rem; color: #cc3333; margin-top: 0.231rem; }
</style>
