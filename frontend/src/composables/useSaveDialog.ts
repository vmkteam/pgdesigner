import { ref, shallowRef } from 'vue'
import { useProjectStore } from '@/stores/project'

interface SaveDialogState {
  defaultDir: string
  defaultName: string
  defaultExtension: string
  resolve: (path: string | null) => void
}

export const saveDialogVisible = ref(false)
export const saveDialogState = shallowRef<SaveDialogState | null>(null)

/** Get the directory of the current project file, or workDir, or empty for FileBrowser home fallback. */
function projectDir(): string {
  const info = useProjectStore().info
  const fp = info?.filePath || ''
  if (fp && !fp.startsWith('postgres')) {
    const i = fp.lastIndexOf('/')
    if (i > 0) return fp.substring(0, i)
  }
  return info?.workDir || ''
}

/** Show Save As dialog. Returns chosen path or null if cancelled.
 *  If defaultDir is omitted, uses the directory of the current project file. */
export function appSaveAs(defaultDir: string | undefined, defaultName: string, defaultExtension = '.pgd'): Promise<string | null> {
  const dir = defaultDir ?? projectDir()
  return new Promise((resolve) => {
    saveDialogState.value = { defaultDir: dir, defaultName, defaultExtension, resolve }
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
