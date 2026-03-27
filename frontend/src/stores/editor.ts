import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import equal from 'fast-deep-equal'
import api from '@/api/factory'
import type { ITableDetail, ILintIssue, IDiffChange, IProjectUpdateTableParams, IProjectPreviewDiffParams } from '@/api/factory'
import { validateTable, type ValidationError } from '@/composables/useTableValidation'

export type { ValidationError }

export const useEditorStore = defineStore('editor', () => {
  const tableName = ref<string | null>(null)
  const original = ref<ITableDetail | null>(null)
  const draft = ref<ITableDetail | null>(null)
  const saving = ref(false)
  const loading = ref(false)
  const serverErrors = ref<string[]>([])
  const lintIssues = ref<ILintIssue[]>([])
  const lintLoading = ref(false)
  const diffChanges = ref<IDiffChange[]>([])
  const diffLoading = ref(false)

  function clearServerErrors() { serverErrors.value = [] }

  const isOpen = computed(() => tableName.value !== null)
  const isDirty = computed(() => {
    if (!draft.value || !original.value) return false
    return !equal(draft.value, original.value)
  })

  const errors = computed<ValidationError[]>(() => {
    if (!draft.value) return []
    return validateTable(draft.value)
  })

  const hasErrors = computed(() => errors.value.length > 0)

  function tabErrors(tab: string): ValidationError[] {
    return errors.value.filter(e => e.tab === tab)
  }

  function fieldHasError(field: string): boolean {
    return errors.value.some(e => e.field === field)
  }

  async function openTable(name: string) {
    tableName.value = name
    loading.value = true
    try {
      const data = await api.project.getTable({ name })
      cleanupStaleRefs(data)
      original.value = JSON.parse(JSON.stringify(data))
      draft.value = JSON.parse(JSON.stringify(data))
    } catch (e) {
      console.error('Failed to load table:', e)
      original.value = null
      draft.value = null
    } finally {
      loading.value = false
    }
    loadLint()
  }

  /** Remove PK/UNIQUE/index/FK column references that don't exist in table columns */
  function cleanupStaleRefs(data: ITableDetail) {
    const colNames = new Set((data.columns || []).map(c => c.name))
    if (data.pk) {
      data.pk.columns = data.pk.columns.filter(c => colNames.has(c))
      if (data.pk.columns.length === 0) data.pk = undefined as any
    }
    if (data.uniques) {
      for (const u of data.uniques) u.columns = u.columns.filter(c => colNames.has(c))
    }
    if (data.indexes) {
      for (const ix of data.indexes) ix.columns = (ix.columns || []).filter(c => colNames.has(c.name))
    }
    if (data.fks) {
      for (const fk of data.fks) fk.columns = fk.columns.filter(c => colNames.has(c.name))
    }
  }

  async function apply() {
    if (!draft.value || !tableName.value || !isDirty.value) return
    saving.value = true
    serverErrors.value = []
    try {
      const params = buildChangeParams() as unknown as IProjectUpdateTableParams
      const updated = await api.project.updateTable(params)
      original.value = JSON.parse(JSON.stringify(updated))
      draft.value = JSON.parse(JSON.stringify(updated))
      // Update tableName after rename so subsequent saves use the new name
      if (updated.name !== tableName.value) {
        tableName.value = updated.name
      }
    } catch (e: unknown) {
      const err = e as { data?: { issues?: { code: string; message: string; path: string }[] }; message?: string }
      if (err?.data?.issues) {
        serverErrors.value = err.data.issues.map(
          (iss) => `[${iss.code}] ${iss.message} (${iss.path})`
        )
      } else {
        serverErrors.value = [err?.message || 'Unknown error']
      }
      throw e
    } finally {
      saving.value = false
    }
  }

  async function saveAndClose() {
    await apply()
  }

  async function loadLint() {
    if (!tableName.value) return
    lintLoading.value = true
    try {
      lintIssues.value = await api.project.lintTable({ name: tableName.value })
    } catch {
      lintIssues.value = []
    } finally {
      lintLoading.value = false
    }
  }

  async function loadDiff() {
    if (!tableName.value || !draft.value || !isDirty.value) {
      diffChanges.value = []
      return
    }
    diffLoading.value = true
    try {
      const params = buildChangeParams() as unknown as IProjectPreviewDiffParams
      diffChanges.value = await api.project.previewDiff(params)
    } catch {
      diffChanges.value = []
    } finally {
      diffLoading.value = false
    }
  }

  function revert() {
    if (!original.value) return
    draft.value = JSON.parse(JSON.stringify(original.value))
  }

  function close() {
    tableName.value = null
    original.value = null
    draft.value = null
    serverErrors.value = []
    lintIssues.value = []
    diffChanges.value = []
  }

  function sectionChanged(key: keyof ITableDetail): boolean {
    if (!draft.value || !original.value) return false
    return !equal(draft.value[key], original.value[key])
  }

  function generalChanged(): boolean {
    if (!draft.value || !original.value) return false
    return draft.value.name !== original.value.name ||
      draft.value.comment !== original.value.comment ||
      draft.value.unlogged !== original.value.unlogged
  }

  function buildChangeParams(): Record<string, unknown> {
    const params: Record<string, unknown> = { name: tableName.value }
    if (generalChanged()) {
      params.general = {
        name: draft.value!.name !== original.value!.name ? draft.value!.name : undefined,
        comment: draft.value!.comment !== original.value!.comment ? draft.value!.comment : undefined,
        unlogged: draft.value!.unlogged !== original.value!.unlogged ? draft.value!.unlogged : undefined,
      }
    }
    if (sectionChanged('columns')) params.columns = draft.value!.columns
    if (sectionChanged('pk')) params.pk = draft.value!.pk || { name: '', columns: [] }
    if (sectionChanged('fks')) params.fks = draft.value!.fks
    if (sectionChanged('uniques')) params.uniques = draft.value!.uniques
    if (sectionChanged('checks')) params.checks = draft.value!.checks
    if (sectionChanged('excludes')) params.excludes = draft.value!.excludes
    if (sectionChanged('indexes')) params.indexes = draft.value!.indexes
    if (sectionChanged('partitionBy') || sectionChanged('partitions')) {
      params.partitionBy = draft.value!.partitionBy || null
      params.partitions = draft.value!.partitions || []
    }
    return params
  }

  return {
    tableName, original, draft, isOpen, isDirty, saving, loading,
    errors, hasErrors, tabErrors, fieldHasError, serverErrors, clearServerErrors,
    lintIssues, lintLoading, loadLint,
    diffChanges, diffLoading, loadDiff,
    openTable, close, apply, save: saveAndClose, saveAndClose, revert,
  }
})
