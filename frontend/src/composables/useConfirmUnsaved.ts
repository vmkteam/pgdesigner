import api from '@/api/factory'
import { useProjectStore } from '@/stores/project'
import { appConfirmSave } from './useAppDialog'
import { appSaveAs } from './useSaveDialog'
import { showToast } from './useToast'

/**
 * Check for unsaved changes and prompt user.
 * Returns true if safe to proceed, false if cancelled.
 */
export async function confirmUnsaved(): Promise<boolean> {
  const store = useProjectStore()
  let dirty = false
  try {
    dirty = await api.project.isDirty()
  } catch {
    // If RPC fails, assume not dirty
  }
  if (!dirty) return true

  const name = store.info?.name || 'Untitled'
  const result = await appConfirmSave(
    `Save changes to "${name}"?\nYour changes will be lost if you don't save them.`,
    'Unsaved Changes',
  )

  if (result === 'cancel') return false
  if (result === 'save') {
    try {
      if (!store.info?.filePath) {
        // No file path — need Save As
        const defaultName = `${name}.pgd`
        const path = await appSaveAs(undefined, defaultName)
        if (!path) return false
        await api.project.saveProjectAs({ path })
      } else {
        await api.project.saveProject()
      }
    } catch (e: unknown) {
      showToast('Save failed: ' + (e instanceof Error ? e.message : e))
      return false
    }
  }
  // 'discard' — proceed without saving
  return true
}
