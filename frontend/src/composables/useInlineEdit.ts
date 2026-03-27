import { ref, nextTick, type Ref, type ShallowRef } from 'vue'
import { identifierError } from './useIdentifierValidation'

/** Inline name editing for list rows (dblclick / Enter / F2). */
export function useInlineEdit(opts: {
  getName: (idx: number) => string
  onCommit: (idx: number, name: string) => void
  listRef: Ref<HTMLElement | undefined> | Readonly<ShallowRef<HTMLElement | null>>
  inputClass: string
  validate?: boolean
}) {
  const editingIdx = ref<number | null>(null)
  const editName = ref('')
  const editError = ref<string | null>(null)

  function startEdit(idx: number) {
    editingIdx.value = idx
    editName.value = opts.getName(idx)
    editError.value = null
    nextTick(() => {
      const el = opts.listRef.value?.querySelector(`.${opts.inputClass}`) as HTMLInputElement
      if (el) { el.focus(); el.select() }
    })
  }

  function commit() {
    if (editingIdx.value == null) return
    if (opts.validate !== false) {
      const err = identifierError(editName.value)
      if (err) { editError.value = err; return }
    }
    editError.value = null
    opts.onCommit(editingIdx.value, editName.value)
    editingIdx.value = null
    nextTick(() => opts.listRef.value?.focus())
  }

  function cancel() {
    editingIdx.value = null
    editError.value = null
    nextTick(() => opts.listRef.value?.focus())
  }

  function onEditKeydown(e: KeyboardEvent) {
    e.stopPropagation()
    if (e.key === 'Enter') { e.preventDefault(); commit() }
    else if (e.key === 'Escape') { cancel() }
  }

  return { editingIdx, editName, editError, startEdit, commit, cancel, onEditKeydown }
}
