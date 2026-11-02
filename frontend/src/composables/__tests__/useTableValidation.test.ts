import { describe, it, expect } from 'vitest'
import { validateTable, type ValidationError } from '../useTableValidation'
import type { ITableDetail } from '@/api/factory'

function minTable(overrides?: Partial<ITableDetail>): ITableDetail {
  return {
    name: 'users',
    comment: '',
    unlogged: false,
    columns: [{ name: 'id', type: 'integer', nullable: false, default: '', comment: '', compression: '', storage: '', collation: '', length: 0, precision: 0, scale: 0, identity: null, generated: null }],
    pk: { name: 'pk_users', columns: ['id'] },
    fks: [],
    uniques: [],
    checks: [],
    excludes: [],
    indexes: [],
    partitionBy: null,
    partitions: [],
    ...overrides,
  } as ITableDetail
}

function codes(errs: ValidationError[]): string[] {
  return errs.map(e => e.code).filter(Boolean)
}


describe('validateTable', () => {
  it('valid table returns no errors', () => {
    expect(validateTable(minTable())).toEqual([])
  })

  // --- General ---

  it('E001: empty table name', () => {
    const errs = validateTable(minTable({ name: '' }))
    expect(codes(errs)).toContain('E001')
    expect(errs.some(e => e.tab === 'general' && e.field === 'name')).toBe(true)
  })

  it('E001: whitespace-only table name', () => {
    const errs = validateTable(minTable({ name: '   ' }))
    expect(codes(errs)).toContain('E001')
  })

  it('E002: table name > 63 chars', () => {
    const errs = validateTable(minTable({ name: 'a'.repeat(64) }))
    expect(codes(errs)).toContain('E002')
  })

  it('name exactly 63 chars is valid', () => {
    const errs = validateTable(minTable({ name: 'a'.repeat(63) }))
    expect(codes(errs)).not.toContain('E002')
  })

  // --- Columns ---

  it('E017: no columns', () => {
    const errs = validateTable(minTable({ columns: [] }))
    expect(codes(errs)).toContain('E017')
  })

  it('E001: empty column name', () => {
    const t = minTable()
    t.columns[0]!.name = ''
    const errs = validateTable(t)
    expect(errs.some(e => e.code === 'E001' && e.field === 'col.0.name')).toBe(true)
  })

  it('E002: column name > 63 chars', () => {
    const t = minTable()
    t.columns[0]!.name = 'x'.repeat(64)
    const errs = validateTable(t)
    expect(errs.some(e => e.code === 'E002' && e.field === 'col.0.name')).toBe(true)
  })

  it('empty column type', () => {
    const t = minTable()
    t.columns[0]!.type = ''
    const errs = validateTable(t)
    expect(errs.some(e => e.field === 'col.0.type')).toBe(true)
  })

  it('E004: duplicate column names', () => {
    const t = minTable()
    t.columns = [
      { ...t.columns[0]!, name: 'id' },
      { ...t.columns[0]!, name: 'id' },
    ]
    const errs = validateTable(t)
    expect(codes(errs)).toContain('E004')
    expect(errs.filter(e => e.code === 'E004')).toHaveLength(2)
  })

  it('E004: duplicate column names case-insensitive', () => {
    const t = minTable()
    t.columns = [
      { ...t.columns[0]!, name: 'Email' },
      { ...t.columns[0]!, name: 'email' },
    ]
    const errs = validateTable(t)
    expect(codes(errs)).toContain('E004')
  })

  it('E031: multiple identity columns', () => {
    const t = minTable()
    const idCol = { ...t.columns[0]!, identity: 'always' }
    t.columns = [
      { ...idCol, name: 'id1' },
      { ...idCol, name: 'id2' },
    ]
    const errs = validateTable(t)
    expect(codes(errs)).toContain('E031')
  })

  it('single identity column is valid', () => {
    const t = minTable()
    t.columns[0]!.identity = 'always'
    const errs = validateTable(t)
    expect(codes(errs)).not.toContain('E031')
  })

  // --- PK ---

  it('E001: PK without name', () => {
    const errs = validateTable(minTable({ pk: { name: '', columns: ['id'] } as any }))
    expect(errs.some(e => e.code === 'E001' && e.field === 'pk.name')).toBe(true)
  })

  it('E007: PK without columns', () => {
    const errs = validateTable(minTable({ pk: { name: 'pk_t', columns: [] } as any }))
    expect(codes(errs)).toContain('E007')
  })

  it('no PK is valid', () => {
    const errs = validateTable(minTable({ pk: null as any }))
    expect(errs.filter(e => e.field.startsWith('pk.'))).toHaveLength(0)
  })

  // --- Unique ---

  it('E001: UNIQUE without name', () => {
    const errs = validateTable(minTable({ uniques: [{ name: '', columns: ['email'] }] as any }))
    expect(errs.some(e => e.code === 'E001' && e.field === 'uq.0.name')).toBe(true)
  })

  it('E013: UNIQUE without columns', () => {
    const errs = validateTable(minTable({ uniques: [{ name: 'uq_x', columns: [] }] as any }))
    expect(codes(errs)).toContain('E013')
  })

  // --- Check ---

  it('E001: CHECK without name', () => {
    const errs = validateTable(minTable({ checks: [{ name: '', expression: 'x > 0' }] as any }))
    expect(errs.some(e => e.code === 'E001' && e.field === 'chk.0.name')).toBe(true)
  })

  it('CHECK without expression', () => {
    const errs = validateTable(minTable({ checks: [{ name: 'chk_x', expression: '' }] as any }))
    expect(errs.some(e => e.field === 'chk.0.expression')).toBe(true)
  })

  // --- Exclude ---

  it('E001: EXCLUDE without name', () => {
    const errs = validateTable(minTable({ excludes: [{ name: '', elements: [{ column: 'x', with: '=' }] }] as any }))
    expect(errs.some(e => e.code === 'E001' && e.field === 'excl.0.name')).toBe(true)
  })

  it('E026: EXCLUDE without elements', () => {
    const errs = validateTable(minTable({ excludes: [{ name: 'ex_x', elements: [] }] as any }))
    expect(codes(errs)).toContain('E026')
  })

  // --- Indexes ---

  it('E001: index without name', () => {
    const errs = validateTable(minTable({ indexes: [{ name: '', keys: [{ kind: 'column', name: 'id' }] }] as any }))
    expect(errs.some(e => e.code === 'E001' && e.field === 'idx.0.name')).toBe(true)
  })

  it('E011: index without columns or expressions', () => {
    const errs = validateTable(minTable({ indexes: [{ name: 'idx_x', keys: [] }] as any }))
    expect(codes(errs)).toContain('E011')
  })

  it('E011: empty index column or expression', () => {
    const empty = (key: object) => validateTable(minTable({ indexes: [{ name: 'idx_x', keys: [key] }] as any }))
    expect(codes(empty({ kind: 'column', name: '' }))).toContain('E011')
    expect(codes(empty({ kind: 'expression', expression: ' ' }))).toContain('E011')
    expect(codes(empty({ kind: 'expression', expression: 'lower(name)' }))).not.toContain('E011')
  })

  it('E005: duplicate index names', () => {
    const errs = validateTable(minTable({
      indexes: [
        { name: 'idx_x', keys: [{ kind: 'column', name: 'id' }] },
        { name: 'idx_x', keys: [{ kind: 'column', name: 'name' }] },
      ] as any,
    }))
    expect(codes(errs)).toContain('E005')
    expect(errs.filter(e => e.code === 'E005')).toHaveLength(2)
  })

  // --- FK ---

  it('E001: FK without name', () => {
    const errs = validateTable(minTable({
      fks: [{ name: '', toTable: 'orders', onDelete: 'restrict', onUpdate: 'restrict', columns: [{ name: 'id', references: 'id' }] }] as any,
    }))
    expect(errs.some(e => e.code === 'E001' && e.field === 'fk.0.name')).toBe(true)
  })

  it('E009: FK without toTable', () => {
    const errs = validateTable(minTable({
      fks: [{ name: 'fk_x', toTable: '', onDelete: 'restrict', onUpdate: 'restrict', columns: [{ name: 'id', references: 'id' }] }] as any,
    }))
    expect(codes(errs)).toContain('E009')
  })

  it('E021: FK without columns', () => {
    const errs = validateTable(minTable({
      fks: [{ name: 'fk_x', toTable: 'orders', onDelete: 'restrict', onUpdate: 'restrict', columns: [] }] as any,
    }))
    expect(codes(errs)).toContain('E021')
  })

  it('E008: FK column without name', () => {
    const errs = validateTable(minTable({
      fks: [{ name: 'fk_x', toTable: 'orders', onDelete: 'restrict', onUpdate: 'restrict', columns: [{ name: '', references: 'id' }] }] as any,
    }))
    expect(codes(errs)).toContain('E008')
  })

  it('E010: FK column without references', () => {
    const errs = validateTable(minTable({
      fks: [{ name: 'fk_x', toTable: 'orders', onDelete: 'restrict', onUpdate: 'restrict', columns: [{ name: 'id', references: '' }] }] as any,
    }))
    expect(codes(errs)).toContain('E010')
  })

  // --- Duplicate constraint names ---

  it('E006: duplicate constraint names across PK and UNIQUE', () => {
    const errs = validateTable(minTable({
      pk: { name: 'same_name', columns: ['id'] } as any,
      uniques: [{ name: 'same_name', columns: ['email'] }] as any,
    }))
    expect(codes(errs)).toContain('E006')
    expect(errs.filter(e => e.code === 'E006')).toHaveLength(2)
  })

  it('E006: duplicate constraint names across FK and CHECK', () => {
    const errs = validateTable(minTable({
      checks: [{ name: 'dup', expression: 'x > 0' }] as any,
      fks: [{ name: 'dup', toTable: 'orders', onDelete: 'restrict', onUpdate: 'restrict', columns: [{ name: 'id', references: 'id' }] }] as any,
    }))
    expect(codes(errs)).toContain('E006')
  })

  it('unique constraint names across all types is valid', () => {
    const errs = validateTable(minTable({
      pk: { name: 'pk_t', columns: ['id'] } as any,
      uniques: [{ name: 'uq_t', columns: ['email'] }] as any,
      checks: [{ name: 'chk_t', expression: 'x > 0' }] as any,
      fks: [{ name: 'fk_t', toTable: 'orders', onDelete: 'restrict', onUpdate: 'restrict', columns: [{ name: 'id', references: 'id' }] }] as any,
    }))
    expect(codes(errs)).not.toContain('E006')
  })
})
