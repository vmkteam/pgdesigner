<script setup lang="ts">
import { ref, useTemplateRef, watch } from 'vue'
import { DialogRoot, DialogPortal, DialogOverlay, DialogContent, DialogTitle } from 'reka-ui'
import { saveDialogVisible, saveDialogState, closeSaveDialog } from '@/composables/useSaveDialog'
import { appConfirm } from '@/composables/useAppDialog'
import FileBrowser from './FileBrowser.vue'

const browserRef = useTemplateRef<InstanceType<typeof FileBrowser>>('browserRef')
const saving = ref(false)

watch(saveDialogVisible, (open) => {
  if (open) {
    saving.value = false
    // FileBrowser will init via its watch on initialDir
  }
})

async function onSave(fullPath: string) {
  // Check if file exists in current listing
  if (browserRef.value?.fileExistsInDir) {
    const overwrite = await appConfirm(
      `"${browserRef.value.fileName}" already exists. Overwrite?`,
      'Confirm Overwrite',
    )
    if (!overwrite) return
  }
  closeSaveDialog(fullPath)
}

function onCancel() {
  closeSaveDialog(null)
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    e.preventDefault()
    onCancel()
    return
  }
  if (e.key === 'Enter' && document.activeElement?.classList.contains('fb-save-input')) {
    e.preventDefault()
    browserRef.value?.doSave()
    return
  }
  browserRef.value?.onKeydown(e)
}
</script>

<template>
  <DialogRoot :open="saveDialogVisible">
    <DialogPortal>
      <DialogOverlay class="od-overlay" @click="onCancel" />
      <DialogContent class="od-box" @escape-key-down.prevent @keydown="onKeydown">
        <DialogTitle class="od-title">Save As</DialogTitle>

        <div class="od-body" style="display: flex; flex-direction: column;">
          <FileBrowser
            ref="browserRef"
            mode="save"
            :initial-dir="saveDialogState?.defaultDir ?? ''"
            :initial-file-name="saveDialogState?.defaultName ?? 'untitled.pgd'"
            :show-filter="false"
            @save="onSave"
          />
        </div>

        <div class="od-footer">
          <button class="od-btn" @click="onCancel">Cancel</button>
          <button
            class="od-btn od-btn-primary"
            :disabled="saving || !browserRef?.fileName?.trim()"
            @click="browserRef?.doSave()"
          >{{ saving ? 'Saving...' : 'Save' }}</button>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
