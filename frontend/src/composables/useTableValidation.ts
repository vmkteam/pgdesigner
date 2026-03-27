import type { ITableDetail } from '@/api/factory'
import { validateIdentifier } from './useIdentifierValidation'

export interface ValidationError {
  tab: 'general' | 'columns' | 'constraints' | 'indexes' | 'fk'
  field: string
  code: string
  message: string
}

function identErrors(name: string, tab: ValidationError['tab'], field: string, label: string): ValidationError[] {
  return validateIdentifier(name)
    .filter(i => i.level === 'error')
    .map(i => ({ tab, field, code: i.message.includes('required') ? 'E001' : 'E002', message: `${label}: ${i.message}` }))
}

export function validateTable(draft: ITableDetail): ValidationError[] {
  return [
    ...validateGeneral(draft),
    ...validateColumns(draft),
    ...validateConstraints(draft),
    ...validateIndexes(draft),
    ...validateFKs(draft),
    ...validateDuplicateConstraintNames(draft),
  ]
}

function validateGeneral(draft: ITableDetail): ValidationError[] {
  return identErrors(draft.name, 'general', 'name', 'Table name')
}

function validateColumns(draft: ITableDetail): ValidationError[] {
  const errs: ValidationError[] = []

  if (!draft.columns || draft.columns.length === 0) {
    errs.push({ tab: 'columns', field: '', code: 'E017', message: 'Table must have at least one column' })
    return errs
  }

  const colNames = new Map<string, number[]>()
  let identityCount = 0

  for (let i = 0; i < draft.columns.length; i++) {
    const c = draft.columns[i]!
    errs.push(...identErrors(c.name, 'columns', `col.${i}.name`, 'Column name'))
    if (!c.type.trim())
      errs.push({ tab: 'columns', field: `col.${i}.type`, code: '', message: 'Column type is required' })
    if (c.identity) identityCount++
    if (c.name.trim()) {
      const key = c.name.toLowerCase()
      if (!colNames.has(key)) colNames.set(key, [])
      colNames.get(key)!.push(i)
    }
  }

  for (const [, indices] of colNames) {
    if (indices.length > 1) {
      for (const i of indices)
        errs.push({ tab: 'columns', field: `col.${i}.name`, code: 'E004', message: 'Duplicate column name' })
    }
  }

  if (identityCount > 1) {
    let seen = 0
    for (let i = 0; i < draft.columns.length; i++) {
      if (draft.columns[i]!.identity) {
        seen++
        if (seen > 1) errs.push({ tab: 'columns', field: `col.${i}.name`, code: 'E031', message: 'Only one identity column allowed' })
      }
    }
  }

  return errs
}

function validateConstraints(draft: ITableDetail): ValidationError[] {
  const errs: ValidationError[] = []

  if (draft.pk) {
    errs.push(...identErrors(draft.pk.name, 'constraints', 'pk.name', 'PK name'))
    if (!draft.pk.columns || draft.pk.columns.length === 0)
      errs.push({ tab: 'constraints', field: 'pk.columns', code: 'E007', message: 'PK must have at least one column' })
  }

  if (draft.uniques) {
    for (let i = 0; i < draft.uniques.length; i++) {
      const u = draft.uniques[i]!
      errs.push(...identErrors(u.name, 'constraints', `uq.${i}.name`, 'UNIQUE name'))
      if (!u.columns || u.columns.length === 0)
        errs.push({ tab: 'constraints', field: `uq.${i}.columns`, code: 'E013', message: 'UNIQUE must have at least one column' })
    }
  }

  if (draft.checks) {
    for (let i = 0; i < draft.checks.length; i++) {
      const c = draft.checks[i]!
      errs.push(...identErrors(c.name, 'constraints', `chk.${i}.name`, 'CHECK name'))
      if (!c.expression.trim())
        errs.push({ tab: 'constraints', field: `chk.${i}.expression`, code: '', message: 'CHECK expression is required' })
    }
  }

  if (draft.excludes) {
    for (let i = 0; i < draft.excludes.length; i++) {
      const ex = draft.excludes[i]!
      errs.push(...identErrors(ex.name, 'constraints', `excl.${i}.name`, 'EXCLUDE name'))
      if (!ex.elements || ex.elements.length === 0)
        errs.push({ tab: 'constraints', field: `excl.${i}.elements`, code: 'E026', message: 'EXCLUDE must have at least one element' })
    }
  }

  return errs
}

function validateIndexes(draft: ITableDetail): ValidationError[] {
  const errs: ValidationError[] = []
  if (!draft.indexes) return errs

  const iNames = new Map<string, number[]>()
  for (let i = 0; i < draft.indexes.length; i++) {
    const idx = draft.indexes[i]!
    errs.push(...identErrors(idx.name, 'indexes', `idx.${i}.name`, 'Index name'))
    if ((!idx.columns || idx.columns.length === 0) && (!idx.expressions || idx.expressions.length === 0))
      errs.push({ tab: 'indexes', field: `idx.${i}.columns`, code: 'E011', message: 'Index must have at least one column or expression' })
    if (idx.name.trim()) {
      if (!iNames.has(idx.name)) iNames.set(idx.name, [])
      iNames.get(idx.name)!.push(i)
    }
  }
  for (const [name, indices] of iNames) {
    if (indices.length > 1) {
      for (const i of indices)
        errs.push({ tab: 'indexes', field: `idx.${i}.name`, code: 'E005', message: `Duplicate index name: ${name}` })
    }
  }

  return errs
}

function validateFKs(draft: ITableDetail): ValidationError[] {
  const errs: ValidationError[] = []
  if (!draft.fks) return errs

  for (let i = 0; i < draft.fks.length; i++) {
    const fk = draft.fks[i]!
    errs.push(...identErrors(fk.name, 'fk', `fk.${i}.name`, 'FK name'))
    if (!fk.toTable)
      errs.push({ tab: 'fk', field: `fk.${i}.toTable`, code: 'E009', message: 'FK target table is required' })
    if (!fk.columns || fk.columns.length === 0)
      errs.push({ tab: 'fk', field: `fk.${i}.columns`, code: 'E021', message: 'FK must have at least one column' })
    if (fk.columns) {
      for (let j = 0; j < fk.columns.length; j++) {
        const fc = fk.columns[j]!
        if (!fc.name.trim())
          errs.push({ tab: 'fk', field: `fk.${i}.col.${j}.name`, code: 'E008', message: 'FK column name is required' })
        if (!fc.references.trim())
          errs.push({ tab: 'fk', field: `fk.${i}.col.${j}.ref`, code: 'E010', message: 'FK referenced column is required' })
      }
    }
  }

  return errs
}

function validateDuplicateConstraintNames(draft: ITableDetail): ValidationError[] {
  const errs: ValidationError[] = []
  const all: { name: string; tab: 'constraints' | 'fk'; field: string }[] = []

  if (draft.pk?.name.trim())
    all.push({ name: draft.pk.name, tab: 'constraints', field: 'pk.name' })
  if (draft.uniques) {
    for (let i = 0; i < draft.uniques.length; i++) {
      if (draft.uniques[i]!.name.trim()) all.push({ name: draft.uniques[i]!.name, tab: 'constraints', field: `uq.${i}.name` })
    }
  }
  if (draft.checks) {
    for (let i = 0; i < draft.checks.length; i++) {
      if (draft.checks[i]!.name.trim()) all.push({ name: draft.checks[i]!.name, tab: 'constraints', field: `chk.${i}.name` })
    }
  }
  if (draft.excludes) {
    for (let i = 0; i < draft.excludes.length; i++) {
      if (draft.excludes[i]!.name.trim()) all.push({ name: draft.excludes[i]!.name, tab: 'constraints', field: `excl.${i}.name` })
    }
  }
  if (draft.fks) {
    for (let i = 0; i < draft.fks.length; i++) {
      if (draft.fks[i]!.name.trim()) all.push({ name: draft.fks[i]!.name, tab: 'fk', field: `fk.${i}.name` })
    }
  }

  const counts = new Map<string, typeof all>()
  for (const entry of all) {
    if (!counts.has(entry.name)) counts.set(entry.name, [])
    counts.get(entry.name)!.push(entry)
  }
  for (const [, entries] of counts) {
    if (entries.length > 1) {
      for (const e of entries)
        errs.push({ tab: e.tab, field: e.field, code: 'E006', message: `Duplicate constraint name: ${entries[0]!.name}` })
    }
  }

  return errs
}
