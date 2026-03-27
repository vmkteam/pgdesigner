import { ref, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import { useIntervalFn, useTitle } from '@vueuse/core'
import equal from 'fast-deep-equal'
import api from '@/api/factory'
import type { IProjectInfo, IERDSchema, ILintIssue, IIgnoredRule, IProjectSettings } from '@/api/factory'
import { showToast } from '@/composables/useToast'

export const useProjectStore = defineStore('project', () => {
  const info = shallowRef<IProjectInfo | null>(null)
  const schema = shallowRef<IERDSchema | null>(null)
  const ddl = ref<string>('')
  const lintIssues = ref<ILintIssue[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  const autoSave = ref(false)
  const ignoredRules = ref<IIgnoredRule[]>([])
  const settings = shallowRef<IProjectSettings | null>(null)
  const dirty = ref(false)
  const lastSaved = ref<Date | null>(null)
  const saveStatus = ref<'idle' | 'saving' | 'saved' | 'error'>('idle')

  const docTitle = useTitle('PgDesigner')

  function updateTitle() {
    if (!info.value) { docTitle.value = 'PgDesigner'; return }
    const suffix = info.value.isRegistered ? 'PgDesigner' : 'PgDesigner [unregistered]'
    if (info.value.isDemo && !info.value.filePath) { docTitle.value = suffix; return }
    const name = info.value.name || 'Untitled'
    const dirtyMark = dirty.value ? ' *' : ''
    docTitle.value = `${name}${dirtyMark} — ${suffix}`
  }

  async function rpc<T>(fn: () => Promise<T>): Promise<T | undefined> {
    try {
      return await fn()
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      // Show toast for non-fatal errors (e.g. read-only mode), don't replace the whole screen
      if (!info.value) {
        error.value = msg // fatal: no project loaded yet
      } else {
        showToast(msg, 'error')
      }
    }
  }

  async function loadAll() {
    // Only show loading spinner on initial load (prevents DiagramCanvas remount)
    if (!schema.value) loading.value = true
    error.value = null
    try {
      const [infoResult, schemaResult, settingsResult] = await Promise.all([
        api.project.getInfo(),
        api.project.getSchema(),
        api.project.getProjectSettings(),
      ])
      info.value = infoResult
      settings.value = settingsResult
      autoSave.value = infoResult.autoSave ?? false
      const normalized = normalizeSchema(schemaResult)
      if (!equal(normalized, schema.value)) {
        schema.value = normalized
      }
      updateTitle()
      startDirtyPolling()
      pollDirty()
    } catch (e) {
      error.value = e instanceof Error ? e.message : String(e)
    } finally {
      loading.value = false
    }
  }

  const testData = ref<string>('')
  const testDataLoading = ref(false)

  function clearTestData() { testData.value = '' }

  async function loadDDL() {
    const result = await rpc(() => api.project.getDDL())
    if (result !== undefined) ddl.value = result
  }

  async function loadTestData(seed: number, rows: number) {
    testDataLoading.value = true
    const result = await rpc(() => api.project.generateTestData({ seed, rows }))
    if (result !== undefined) testData.value = result
    testDataLoading.value = false
  }

  async function toggleAutoSave() {
    const newVal = !autoSave.value
    const result = await rpc(() => api.project.setAutoSave({ enabled: newVal }))
    if (result !== undefined) autoSave.value = newVal
  }

  async function saveProject() {
    if (!info.value?.filePath) {
      showToast('Use Save As (⌘⇧S) to save a new project', 'error')
      return
    }
    saveStatus.value = 'saving'
    try {
      await api.project.saveProject()
      saveStatus.value = 'saved'
      lastSaved.value = new Date()
      dirty.value = false
    } catch (e) {
      saveStatus.value = 'error'
      showToast(e instanceof Error ? e.message : String(e), 'error')
    }
  }

  async function pollDirty() {
    try {
      const wasDirty = dirty.value
      dirty.value = await api.project.isDirty()
      if (dirty.value !== wasDirty) updateTitle()
    } catch { /* ignore */ }
  }

  const { resume: startDirtyPolling } = useIntervalFn(pollDirty, 5000, { immediate: false })

  async function loadLint() {
    const result = await rpc(() => api.project.lint())
    if (result !== undefined) lintIssues.value = result || []
  }

  async function fixLintIssues(issues: { code: string; path: string }[]) {
    const result = await rpc(() => api.project.fixLintIssues({ issues }))
    if (result) {
      lintIssues.value = result.issues || []
      return result.fixed
    }
    return 0
  }

  async function ignoreLintRules(rules: string[], table?: string) {
    const result = await rpc(() => api.project.ignoreLintRules({ rules, table }))
    if (result !== undefined) {
      lintIssues.value = result || []
      await loadIgnoredRules()
    }
  }

  async function loadIgnoredRules() {
    const result = await rpc(() => api.project.getIgnoredRules())
    if (result !== undefined) ignoredRules.value = result || []
  }

  async function unignoreLintRule(code: string, scope: string) {
    const table = scope !== 'project' ? scope : undefined
    const result = await rpc(() => api.project.unignoreLintRules({ rules: [code], table }))
    if (result !== undefined) {
      await Promise.all([loadIgnoredRules(), loadLint()])
    }
  }

  return { info, schema, ddl, testData, testDataLoading, lintIssues, ignoredRules, settings, loading, error, autoSave, dirty, lastSaved, saveStatus, loadAll, loadDDL, loadTestData, clearTestData, loadLint, toggleAutoSave, saveProject, pollDirty, fixLintIssues, ignoreLintRules, loadIgnoredRules, unignoreLintRule }
})

// Go JSON marshals nil slices as null — ensure arrays
function normalizeSchema(s: IERDSchema): IERDSchema {
  for (const t of s.tables || []) {
    t.columns = t.columns || []
    t.indexes = t.indexes || []
  }
  s.tables = s.tables || []
  s.references = s.references || []
  return s
}
