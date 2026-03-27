import api from '@/api/factory'
import { useProjectStore } from '@/stores/project'
import { useUiStore } from '@/stores/ui'
import { confirmUnsaved } from './useConfirmUnsaved'
import { appSaveAs } from './useSaveDialog'
import { showToast } from './useToast'

export function useFileActions() {
  const store = useProjectStore()
  const ui = useUiStore()

  async function fileNew() {
    if (!await confirmUnsaved()) return
    try {
      await api.app.newProject()
      await store.loadAll()
      ui.isWelcome = false
      ui.settingsOpen = true
    } catch (e: unknown) {
      showToast('New failed: ' + (e instanceof Error ? e.message : e))
    }
  }

  async function fileOpen() {
    if (!await confirmUnsaved()) return
    ui.openDialogOpen = true
  }

  async function fileSaveAs() {
    if (ui.isWelcome) return
    const name = store.info?.name || 'untitled'
    const fp = store.info?.filePath || ''
    const defaultName = fp ? fp.substring(fp.lastIndexOf('/') + 1) : `${name}.pgd`
    const path = await appSaveAs(undefined, defaultName)
    if (!path) return
    try {
      await api.project.saveProjectAs({ path })
      await store.loadAll()
    } catch (e: unknown) {
      showToast('Save As failed: ' + (e instanceof Error ? e.message : e))
    }
  }

  async function fileClose() {
    if (!await confirmUnsaved()) return
    try {
      await api.app.closeProject()
      await store.loadAll()
      ui.isWelcome = true
    } catch (e: unknown) {
      showToast('Close failed: ' + (e instanceof Error ? e.message : e))
    }
  }

  return { fileNew, fileOpen, fileSaveAs, fileClose }
}
