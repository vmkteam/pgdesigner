import { ref, shallowRef } from 'vue'

interface SaveDialogState {
  defaultDir: string
  defaultName: string
  resolve: (path: string | null) => void
}

export const saveDialogVisible = ref(false)
export const saveDialogState = shallowRef<SaveDialogState | null>(null)

/** Show Save As dialog. Returns chosen path or null if cancelled. */
export function appSaveAs(defaultDir: string, defaultName: string): Promise<string | null> {
  return new Promise((resolve) => {
    saveDialogState.value = { defaultDir, defaultName, resolve }
    saveDialogVisible.value = true
  })
}

export function closeSaveDialog(path: string | null) {
  if (saveDialogState.value) {
    saveDialogState.value.resolve(path)
  }
  saveDialogVisible.value = false
  saveDialogState.value = null
}
