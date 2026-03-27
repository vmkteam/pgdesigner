<script setup lang="ts">
import { computed, watch, ref, nextTick } from 'vue'
import { useDebounceFn, watchDebounced } from '@vueuse/core'
import { VueFlow, useVueFlow } from '@vue-flow/core'
import { MiniMap } from '@vue-flow/minimap'
import type { Node, Edge } from '@vue-flow/core'

import '@vue-flow/minimap/dist/style.css'
import api from '@/api/factory'
import { useProjectStore } from '@/stores/project'
import { useCanvasStore } from '@/stores/canvas'
import { useUiStore } from '@/stores/ui'
import TableNode from './TableNode.vue'
import BatchEdge from './BatchEdge.vue'
import { appPrompt, appConfirm } from '@/composables/useAppDialog'
import { showToast } from '@/composables/useToast'
import { createTableWithPK } from '@/composables/useCreateTable'
import {
  computeEdgePath,
  buildAdjacency,
  findClusters,
  fixOverlaps as fixOverlapsAlgo,
  gridPlaceCluster,
  layoutClusterSugiyama,
  placeClusterBoxes,
  type Rect,
} from './erd-engine'

import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'

const store = useProjectStore()
const canvasStore = useCanvasStore()
const ui = useUiStore()

const { fitView, zoomIn, zoomOut, zoomTo, setCenter, setViewport, setNodes, getNodes, addSelectedNodes, removeSelectedNodes, nodeLookup, viewport } = useVueFlow({
  id: 'erd',
  defaultEdgeOptions: { type: 'batch', animated: false },
  snapToGrid: true,
  snapGrid: [20, 20] as [number, number],
  fitViewOnInit: false,
  panOnDrag: true,
  selectNodesOnDrag: false,
  selectionKeyCode: 'Shift',
  multiSelectionKeyCode: 'Shift',
  panOnScroll: true,
  zoomOnScroll: false,
  zoomOnPinch: true,
  zoomOnDoubleClick: false,
  minZoom: 0.1,
  maxZoom: 2,
})

// Sync zoom % in toolbar
watch(() => viewport.value.zoom, (z) => {
  canvasStore.zoom = Math.round(z * 100)
})

const grayTargets = new Set(['statuses'])

// Convert schema → nodes (filtered by active schema)
const visibleTableNames = computed(() => {
  if (!store.schema) return new Set<string>()
  const filter = canvasStore.activeSchema
  return new Set(
    store.schema.tables
      .filter(t => !filter || t.schema === filter)
      .map(t => t.name)
  )
})

const nodes = computed<Node[]>(() => {
  if (!store.schema) return []
  return store.schema.tables
    .filter(t => visibleTableNames.value.has(t.name))
    .map(t => ({
      id: t.name,
      type: 'table',
      position: { x: t.x, y: t.y },
      data: { name: t.name, columns: t.columns, indexes: t.indexes, partitioned: t.partitioned, partitionCount: t.partitionCount },
    }))
})

// ── Helpers ──────────────────────────────────────────────────────────

function getNodeSize(id: string) {
  const node = nodeLookup.value.get(id)
  return {
    width: node?.dimensions?.width ?? 200,
    height: node?.dimensions?.height ?? 100,
  }
}

function getNodePos(id: string) {
  const node = nodeLookup.value.get(id)
  return node ? { x: node.position.x, y: node.position.y } : { x: 0, y: 0 }
}

function nodeSizes(): Map<string, { width: number; height: number }> {
  const m = new Map<string, { width: number; height: number }>()
  for (const n of getNodes.value) {
    m.set(n.id, { width: n.dimensions?.width ?? 200, height: n.dimensions?.height ?? 100 })
  }
  return m
}

// ── Edge path computation ────────────────────────────────────────────

const edges = ref<Edge[]>([])

function buildEdges() {
  if (!store.schema) { edges.value = []; return }

  const edgeItems: { name: string; from: string; to: string; path: string; labelX: number; labelY: number; gray: boolean }[] = []
  const visible = visibleTableNames.value

  for (const r of store.schema.references) {
    if (visible.size > 0 && !visible.has(r.from) && !visible.has(r.to)) continue
    // Skip edges to non-existent nodes (deleted tables with stale FK refs)
    if (!nodeLookup.value.has(r.from) || !nodeLookup.value.has(r.to)) continue
    const pathData = computeEdgePath(
      getNodePos(r.from), getNodeSize(r.from),
      getNodePos(r.to), getNodeSize(r.to),
      r.from === r.to,
    )
    if (!pathData) continue
    edgeItems.push({ name: r.name, from: r.from, to: r.to, path: pathData.path, labelX: pathData.labelX, labelY: pathData.labelY, gray: grayTargets.has(r.to) })
  }

  const firstTable = store.schema.tables[0]?.name || '_'
  edges.value = [{
    id: '__batch',
    source: firstTable,
    target: firstTable,
    type: 'batch',
    data: { edges: edgeItems },
    style: { strokeWidth: 1 },
  }]
}

// Build edges once nodes have dimensions (replaces polling)
let edgesBuilt = false

let initialFitDone = false

function onNodesInitialized() {
  if (!edgesBuilt) {
    edgesBuilt = true
    buildEdges()
  }
  if (savedViewport) {
    const vp = savedViewport
    savedViewport = null
    nextTick(() => setViewport(vp, { duration: 0 }))
    buildEdges()
  } else if (!initialFitDone) {
    initialFitDone = true
    nextTick(() => fitView({ padding: 0.1, maxZoom: 1 }))
  }
}

// Rebuild edges when schema changes (table/FK added/removed)
watchDebounced(() => store.schema, () => {
  if (edgesBuilt) buildEdges()
}, { debounce: 200 })
watchDebounced(() => canvasStore.activeSchema, () => {
  if (edgesBuilt) buildEdges()
}, { debounce: 150 })

// Toolbar actions
watch(() => canvasStore.pendingAction, (action) => {
  if (!action) return
  canvasStore.consumeAction()
  switch (action) {
    case 'zoomIn': zoomIn(); break
    case 'zoomOut': zoomOut(); break
    case 'fitToScreen': fitView({ padding: 0.1, maxZoom: 1 }); break
    case 'resetZoom': zoomTo(1); break
    case 'fixOverlaps': runFixOverlaps(); break
    case 'autoLayout': runAutoLayout(); break
    case 'clusterTables': runClusterTables(); break
    case 'focusNode': handleFocusNode(); break
  }
})

function onNodeDrag() {
  buildEdges()
}

const debouncedSaveLayout = useDebounceFn(saveLayout, 500)

function onNodeDragStop() {
  if (!store.schema) return
  for (const n of getNodes.value) {
    const table = store.schema.tables.find(t => t.name === n.id)
    if (table) {
      table.x = Math.round(n.position.x)
      table.y = Math.round(n.position.y)
    }
  }
  buildEdges()
  debouncedSaveLayout()
}

function saveLayout() {
  if (!store.schema) return
  const positions = store.schema.tables.map(t => ({
    name: t.name, schema: t.schema || '', x: t.x, y: t.y,
  }))
  api.project.saveLayout({ positions }).catch(() => {})
}

function onEdgeClick(name: string, from: string, _to: string) {
  ui.openTableEditor(from, 'fk', name)
}

function onNodeDoubleClick(event: { node: Node }) {
  if (canvasStore.activeTool !== 'pointer') return
  ui.openTableEditor(event.node.id)
}

let clipboard: string[] = []

function onCanvasKeydown(e: KeyboardEvent) {
  if (!(e.metaKey || e.ctrlKey)) return
  // Ctrl+A — select all nodes
  if (e.key === 'a') {
    e.preventDefault()
    addSelectedNodes(getNodes.value)
  }
  // Ctrl+C — copy selected tables
  if (e.key === 'c') {
    const selected = getNodes.value.filter(n => n.selected)
    if (selected.length) {
      e.preventDefault()
      clipboard = selected.map(n => n.id)
    }
  }
  // Ctrl+V — paste copied tables
  if (e.key === 'v' && clipboard.length) {
    e.preventDefault()
    pasteClipboard()
  }
}

async function pasteClipboard() {
  if (!clipboard.length) return
  const schemaName = canvasStore.activeSchema || store.info?.schemas?.[0] || 'public'
  try {
    for (const srcName of clipboard) {
      const srcData = await api.project.getTable({ name: srcName })
      const shortSrc = srcName.replace(/.*\./, '')
      let copyName = shortSrc + '_copy'
      // Avoid collision
      const existing = new Set((store.schema?.tables || []).map(t => t.name))
      let n = 1
      while (existing.has(copyName)) { copyName = `${shortSrc}_copy${++n}` }

      await api.project.createTable({ schemaName, tableName: copyName })
      const fullCopy = schemaName !== (store.info?.schemas?.[0] || 'public') ? `${schemaName}.${copyName}` : copyName

      // Copy columns (strip identity), PK, indexes, constraints — but NOT FKs (would reference wrong tables)
      const cols = (srcData.columns || []).map(c => ({ ...c, pk: false, fk: false, identity: '' }))
      const params: Record<string, unknown> = { name: fullCopy, columns: cols }
      if (srcData.pk) {
        params.pk = { name: `pk_${copyName}`, columns: srcData.pk.columns }
        // restore pk flags
        const pkSet = new Set(srcData.pk.columns)
        for (const c of cols) c.pk = pkSet.has(c.name)
      }
      if (srcData.uniques?.length) params.uniques = srcData.uniques
      if (srcData.checks?.length) params.checks = srcData.checks
      if (srcData.indexes?.length) {
        params.indexes = srcData.indexes.map(ix => ({ ...ix, name: ix.name.replace(shortSrc, copyName) }))
      }
      await api.project.updateTable(params as any)

      // Position offset from source
      const srcNode = getNodes.value.find(n => n.id === srcName)
      if (srcNode) {
        const allPos = (store.schema?.tables || []).map(t => ({ name: t.name, schema: t.schema || '', x: t.x, y: t.y }))
        allPos.push({ name: fullCopy, schema: schemaName, x: srcNode.position.x + 40, y: srcNode.position.y + 40 })
        await api.project.saveLayout({ positions: allPos })
      }
    }
    await reloadKeepViewport()
  } catch (e: unknown) {
    showToast('Paste failed: ' + (e instanceof Error ? e.message : e))
  }
}

async function onDeleteSelected() {
  const selected = getNodes.value.filter(n => n.selected)
  if (!selected.length) return
  const names = selected.map(n => n.id)
  const label = names.length === 1 ? `"${names[0]}"` : `${names.length} tables`
  if (!await appConfirm(`Delete ${label}?`, 'Delete')) return
  try {
    for (const name of names) {
      await api.project.deleteTable({ name })
    }
    await reloadKeepViewport()
  } catch (e: unknown) {
    showToast('Delete failed: ' + (e instanceof Error ? e.message : e))
  }
}

function onNodeClick(event: { node: Node }) {
  const tool = canvasStore.activeTool
  if (tool === 'pointer') return

  const nodeName = event.node.id

  if (tool === 'createFK' || tool === 'createM2M') {
    if (!canvasStore.toolSourceNode) {
      // First click — select source, highlight it
      canvasStore.toolSourceNode = nodeName
      removeSelectedNodes(getNodes.value)
      const graphNode = getNodes.value.find(n => n.id === nodeName)
      if (graphNode) addSelectedNodes([graphNode])
    } else {
      // Second click — execute
      const source = canvasStore.toolSourceNode
      if (source === nodeName) { canvasStore.toolSourceNode = null; return } // same node, reset
      if (tool === 'createFK') executeCreateFK(source, nodeName)
      else executeCreateM2M(source, nodeName)
    }
  }
}

async function onPaneClick(event: MouseEvent) {
  if (canvasStore.activeTool === 'createTable') {
    await executeCreateTable(event)
  }
}

/** Generate FK column name based on target PK column and naming convention */
function fkColName(targetTable: string, pkCol: string, singularTarget: string, naming: string): string {
  const isSnake = naming === 'snake_case' || naming === ''
  // If PK is generic "id" or "{table}id" pattern → prefix with singular table name
  const lowerPk = pkCol.toLowerCase()
  const lowerTarget = targetTable.toLowerCase().replace(/.*\./, '')
  if (lowerPk === 'id' || lowerPk === `${lowerTarget}id` || lowerPk === `${lowerTarget}_id`) {
    if (isSnake) return `${singularTarget}_id`
    return `${singularTarget}Id`
  }
  // PK has a specific name (like user_code) → use as-is
  return pkCol
}

let savedViewport: { x: number; y: number; zoom: number } | null = null

/** Reload schema without resetting viewport */
async function reloadKeepViewport() {
  const vp = { x: viewport.value.x, y: viewport.value.y, zoom: viewport.value.zoom }
  savedViewport = vp
  // Sync current canvas positions to server before reload to prevent node jumps
  syncPositionsToSchema()
  const positions = (store.schema?.tables || []).map(t => ({
    name: t.name, schema: t.schema || '', x: t.x, y: t.y,
  }))
  await api.project.saveLayout({ positions }).catch(() => {})
  await store.loadAll()
  // Belt-and-suspenders: also restore via setTimeout in case onNodesInitialized doesn't fire
  setTimeout(() => { savedViewport = null; setViewport(vp, { duration: 0 }) }, 300)
}

// ── Canvas tool actions ──────────────────────────────────────────────

async function executeCreateTable(event: MouseEvent) {
  const schemaName = canvasStore.activeSchema || store.info?.schemas?.[0] || 'public'
  const name = await appPrompt('New table name:', 'Create Table')
  if (!name) { canvasStore.resetTool(); return }

  try {
    const defaultSchema = store.info?.schemas?.[0] || 'public'
    const fullName = await createTableWithPK(schemaName, name, defaultSchema)

    // Calculate click position in canvas coordinates
    let posX = 0, posY = 0
    const curVp = viewport.value
    const container = document.querySelector('.vue-flow') as HTMLElement
    if (container) {
      const rect = container.getBoundingClientRect()
      posX = Math.round((event.clientX - rect.left - curVp.x) / curVp.zoom / 20) * 20
      posY = Math.round((event.clientY - rect.top - curVp.y) / curVp.zoom / 20) * 20
    }

    // Save ALL positions (current canvas + new table) to avoid overwriting layout
    syncPositionsToSchema()
    const allPositions = (store.schema?.tables || []).map(t => ({ name: t.name, schema: t.schema || '', x: t.x, y: t.y }))
    allPositions.push({ name: fullName, schema: schemaName, x: posX, y: posY })
    await api.project.saveLayout({ positions: allPositions })
    savedViewport = { x: curVp.x, y: curVp.y, zoom: curVp.zoom }
    await store.loadAll()
    setTimeout(() => { savedViewport = null; setViewport({ x: curVp.x, y: curVp.y, zoom: curVp.zoom }, { duration: 0 }) }, 300)
    canvasStore.resetTool()
    ui.openTableEditor(fullName)
  } catch (e: unknown) {
    showToast('Create table failed: ' + (e instanceof Error ? e.message : e))
    canvasStore.resetTool()
  }
}

async function executeCreateFK(fromTable: string, toTable: string) {
  canvasStore.resetTool()
  try {
    // Get current table to read existing FKs
    const tableData = await api.project.getTable({ name: fromTable })
    const existingFks = tableData.fks || []
    const existingCols = tableData.columns || []

    // Get target table PK columns
    const targetData = await api.project.getTable({ name: toTable })
    const targetPkCols = targetData.pk?.columns || []
    if (!targetPkCols.length) {
      showToast(`Table "${toTable}" has no primary key`)
      return
    }

    // Check for existing FK to same target
    if (existingFks.some(fk => fk.toTable === toTable)) {
      showToast(`FK to "${toTable}" already exists`)
      return
    }

    // Create FK columns in source
    const shortTarget = toTable.replace(/.*\./, '')
    const shortFrom = fromTable.replace(/.*\./, '')
    const singularTarget = await api.project.singularize({ word: shortTarget })
    const naming = store.settings?.namingConvention || 'snake_case'
    const mappings: { name: string; references: string }[] = []
    const newCols = [...existingCols]
    const fkColNames: string[] = []

    for (const pkCol of targetPkCols) {
      const colName = fkColName(shortTarget, pkCol, singularTarget || shortTarget, naming)
      fkColNames.push(colName)
      // Add column if not exists
      if (!newCols.find(c => c.name === colName)) {
        const targetCol = targetData.columns.find(c => c.name === pkCol)
        newCols.push({
          name: colName,
          type: targetCol?.type || 'integer',
          length: targetCol?.length || 0,
          precision: targetCol?.precision || 0,
          scale: targetCol?.scale || 0,
          nullable: true,
          default: '',
          pk: false, fk: true,
          identity: '', generated: '', generatedStored: false,
          comment: '', compression: '', storage: '', collation: '',
        })
      }
      mappings.push({ name: colName, references: pkCol })
    }

    const defaultOnDelete = store.settings?.defaultOnDelete || 'no action'
    const defaultOnUpdate = store.settings?.defaultOnUpdate || 'no action'

    const fkName = `fk_${shortFrom}_${fkColNames.join('_')}`
    const newFk = {
      name: fkName,
      toTable,
      onDelete: defaultOnDelete,
      onUpdate: defaultOnUpdate,
      deferrable: false,
      initially: '',
      columns: mappings,
    }

    // Create index for each FK column
    const existingIndexes = tableData.indexes || []
    const newIndexes = fkColNames
      .filter(col => !existingIndexes.some(ix => ix.columns?.length === 1 && ix.columns[0]?.name === col))
      .map(col => ({ name: `ix_${shortFrom}_${col}`, unique: false, nullsDistinct: false, using: 'btree', columns: [{ name: col }], expressions: [], where: '', include: [] }))

    await api.project.updateTable({
      name: fromTable,
      columns: newCols,
      fks: [...existingFks, newFk],
      indexes: [...existingIndexes, ...newIndexes],
    } as any)
    await reloadKeepViewport()
  } catch (e: unknown) {
    showToast('Create FK failed: ' + (e instanceof Error ? e.message : e))
  }
}

async function executeCreateM2M(tableA: string, tableB: string) {
  canvasStore.resetTool()
  try {
    const shortA = tableA.replace(/.*\./, '')
    const shortB = tableB.replace(/.*\./, '')
    const schemaName = canvasStore.activeSchema || store.info?.schemas?.[0] || 'public'
    const naming = store.settings?.namingConvention || 'snake_case'
    const isSnake = naming === 'snake_case' || naming === ''
    const junctionName = isSnake ? `${shortA}_${shortB}` : `${shortA}${shortB.charAt(0).toUpperCase()}${shortB.slice(1)}`

    // Get PKs of both tables
    const [dataA, dataB, singA, singB] = await Promise.all([
      api.project.getTable({ name: tableA }),
      api.project.getTable({ name: tableB }),
      api.project.singularize({ word: shortA }),
      api.project.singularize({ word: shortB }),
    ])
    const pkA = dataA.pk?.columns || []
    const pkB = dataB.pk?.columns || []
    if (!pkA.length || !pkB.length) {
      showToast('Both tables must have primary keys')
      return
    }

    // Create junction table
    await api.project.createTable({ schemaName, tableName: junctionName })
    const defaultSchema = store.info?.schemas?.[0] || 'public'
    const fullJunction = schemaName !== defaultSchema ? `${schemaName}.${junctionName}` : junctionName

    // Position between the two tables + save ALL positions
    const nodeA = getNodes.value.find(n => n.id === tableA)
    const nodeB = getNodes.value.find(n => n.id === tableB)
    const allPositions = (store.schema?.tables || []).map(t => ({ name: t.name, schema: t.schema || '', x: t.x, y: t.y }))
    if (nodeA && nodeB) {
      const x = Math.round(((nodeA.position.x + nodeB.position.x) / 2) / 20) * 20
      const y = Math.round(((nodeA.position.y + nodeB.position.y) / 2) / 20) * 20
      allPositions.push({ name: fullJunction, schema: schemaName, x, y })
    }
    await api.project.saveLayout({ positions: allPositions })

    // Build columns, PK, and 2 FKs
    const columns: any[] = []
    const pkCols: string[] = []
    const fks: any[] = []
    const defaultOnDelete = store.settings?.defaultOnDelete || 'no action'
    const defaultOnUpdate = store.settings?.defaultOnUpdate || 'no action'

    for (const pkCol of pkA) {
      const colName = fkColName(shortA, pkCol, singA || shortA, naming)
      const srcCol = dataA.columns.find(c => c.name === pkCol)
      columns.push({ name: colName, type: srcCol?.type || 'integer', length: srcCol?.length || 0, precision: srcCol?.precision || 0, scale: srcCol?.scale || 0, nullable: false, default: '', pk: true, fk: true, identity: '', generated: '', generatedStored: false, comment: '', compression: '', storage: '', collation: '' })
      pkCols.push(colName)
    }
    for (const pkCol of pkB) {
      const colName = fkColName(shortB, pkCol, singB || shortB, naming)
      const srcCol = dataB.columns.find(c => c.name === pkCol)
      columns.push({ name: colName, type: srcCol?.type || 'integer', length: srcCol?.length || 0, precision: srcCol?.precision || 0, scale: srcCol?.scale || 0, nullable: false, default: '', pk: true, fk: true, identity: '', generated: '', generatedStored: false, comment: '', compression: '', storage: '', collation: '' })
      pkCols.push(colName)
    }

    const fkColNamesA = pkA.map(pk => fkColName(shortA, pk, singA || shortA, naming))
    const fkColNamesB = pkB.map(pk => fkColName(shortB, pk, singB || shortB, naming))

    fks.push({
      name: `fk_${junctionName}_${fkColNamesA.join('_')}`, toTable: tableA,
      onDelete: defaultOnDelete, onUpdate: defaultOnUpdate, deferrable: false, initially: '',
      columns: pkA.map((pk, i) => ({ name: fkColNamesA[i]!, references: pk })),
    })
    fks.push({
      name: `fk_${junctionName}_${fkColNamesB.join('_')}`, toTable: tableB,
      onDelete: defaultOnDelete, onUpdate: defaultOnUpdate, deferrable: false, initially: '',
      columns: pkB.map((pk, i) => ({ name: fkColNamesB[i]!, references: pk })),
    })

    // Create indexes for each FK column
    const indexes = [...fkColNamesA, ...fkColNamesB].map(col => ({
      name: `ix_${junctionName}_${col}`, unique: false, nullsDistinct: false,
      using: 'btree', columns: [{ name: col }], expressions: [], where: '', include: [],
    }))

    await api.project.updateTable({
      name: fullJunction,
      columns,
      pk: { name: `pk_${junctionName}`, columns: pkCols },
      fks,
      indexes,
    } as any)
    await reloadKeepViewport()
  } catch (e: unknown) {
    showToast('Create M:N failed: ' + (e instanceof Error ? e.message : e))
  }
}

function handleFocusNode() {
  const name = canvasStore.focusNodeName
  canvasStore.focusNodeName = null
  if (!name) return

  const node = getNodes.value.find(n => n.id === name)
  if (!node) return

  const w = node.dimensions?.width ?? 200
  const h = node.dimensions?.height ?? 100
  removeSelectedNodes(getNodes.value)
  addSelectedNodes([node])
  setCenter(node.position.x + w / 2, node.position.y + h / 2, { zoom: 1, duration: 300 })
}

// ── Layout actions ───────────────────────────────────────────────────

function syncPositionsToSchema() {
  if (!store.schema) return
  for (const n of getNodes.value) {
    const t = store.schema.tables.find(t => t.name === n.id)
    if (t) { t.x = Math.round(n.position.x); t.y = Math.round(n.position.y) }
  }
}

function nodeRects(): Rect[] {
  return getNodes.value.map(n => ({
    id: n.id,
    x: n.position.x,
    y: n.position.y,
    w: n.dimensions?.width ?? 200,
    h: n.dimensions?.height ?? 100,
  }))
}

function applyPositions(targets: Record<string, { x: number; y: number }>) {
  setNodes(getNodes.value.map(n => {
    const t = targets[n.id]
    return t ? { ...n, position: t } : n
  }))
  syncPositionsToSchema()
}

function runFixOverlaps() {
  const rects = nodeRects()
  if (!rects.length) return

  const fixed = fixOverlapsAlgo(rects)
  const targets: Record<string, { x: number; y: number }> = {}
  for (const r of fixed) targets[r.id] = { x: r.x, y: r.y }

  applyPositions(targets)
  setTimeout(() => { fitView({ padding: 0.1, maxZoom: 1 }); buildEdges() }, 100)
}

const HUB_TABLES = new Set(['statuses'])

function runClusterTables() {
  if (!store.schema) return
  const padX = 240, padY = 180, clusterGap = 400, gridSize = 20
  const flowNodes = getNodes.value
  const nodeIds = flowNodes.map(n => n.id)
  const sizes = nodeSizes()

  const adj = buildAdjacency(nodeIds, store.schema.references, HUB_TABLES)
  const clusters = findClusters(nodeIds, adj, HUB_TABLES)

  const targets: Record<string, { x: number; y: number }> = {}
  const boxes: { w: number; h: number; names: string[] }[] = []

  for (const cluster of clusters) {
    const result = gridPlaceCluster(cluster, adj, sizes, padX, padY)
    Object.assign(targets, result.targets)
    boxes.push({ w: result.w, h: result.h, names: cluster })
  }

  placeClusterBoxes(boxes, targets, clusterGap, gridSize)
  applyPositions(targets)
  setTimeout(() => { runFixOverlaps(); buildEdges() }, 100)
}

function runAutoLayout() {
  if (!store.schema) return
  const padX = 60, padY = 50, clusterGap = 150, gridSize = 20
  const flowNodes = getNodes.value
  const nodeIds = flowNodes.map(n => n.id)
  const sizes = nodeSizes()

  const adj = buildAdjacency(nodeIds, store.schema.references, HUB_TABLES)
  const clusters = findClusters(nodeIds, adj, HUB_TABLES)

  const targets: Record<string, { x: number; y: number }> = {}
  const boxes: { w: number; h: number; names: string[] }[] = []

  for (const cluster of clusters) {
    const clusterTargets = layoutClusterSugiyama(cluster, store.schema.references, HUB_TABLES, sizes, padX, padY)
    let maxW = 0, maxH = 0
    for (const name of cluster) {
      const t = clusterTargets[name]; if (!t) continue
      const s = sizes.get(name) ?? { width: 200, height: 100 }
      maxW = Math.max(maxW, t.x + s.width)
      maxH = Math.max(maxH, t.y + s.height)
    }
    boxes.push({ w: maxW, h: maxH, names: cluster })
    for (const name of cluster) if (clusterTargets[name]) targets[name] = clusterTargets[name]!
  }

  placeClusterBoxes(boxes, targets, clusterGap, gridSize)
  applyPositions(targets)
  setTimeout(() => { runFixOverlaps(); buildEdges() }, 100)
}
</script>

<template>
  <VueFlow
    :nodes="nodes"
    :edges="edges"
    class="h-full"
    :class="{ 'cursor-crosshair': canvasStore.activeTool !== 'pointer' }"
    :default-viewport="{ x: 0, y: 0, zoom: 1 }"
    :delete-key-code="null"
    @keydown.delete="onDeleteSelected"
    @keydown.backspace="onDeleteSelected"
    @keydown="onCanvasKeydown"
    @node-drag="onNodeDrag"
    @node-drag-stop="onNodeDragStop"
    @node-click="onNodeClick"
    @node-double-click="onNodeDoubleClick"
    @pane-click="onPaneClick"
    @nodes-initialized="onNodesInitialized"
  >
    <template #node-table="nodeProps">
      <TableNode :data="nodeProps.data" />
    </template>

    <template #edge-batch="edgeProps">
      <BatchEdge v-bind="edgeProps" @edge-click="onEdgeClick" />
    </template>

    <MiniMap
      pannable
      zoomable
      :width="150"
      :height="100"
      :node-stroke-color="() => 'var(--color-minimap-table-stroke)'"
      :node-color="() => 'var(--color-minimap-table-fill)'"
      :mask-color="'rgba(0,0,0,0.05)'"
      position="bottom-right"
    />
  </VueFlow>
</template>

<style>
.vue-flow {
  background-image: radial-gradient(circle, var(--color-border-subtle) 1px, transparent 1px) !important;
  background-size: 20px 20px !important;
  background-color: var(--color-bg-panel) !important;
}
.vue-flow__node { border-radius: 0 !important; padding: 0 !important; border: none !important; background: none !important; box-shadow: none !important; }
.vue-flow__node.selected { box-shadow: 0 0 0 2px var(--color-accent) !important; }
.vue-flow__edge-path { stroke-width: 1; }
.vue-flow__selection { border-radius: 0 !important; }
.vue-flow__minimap { background: var(--color-bg-surface) !important; }
.vue-flow__minimap-mask { fill: var(--color-bg-app) !important; opacity: 0.6 !important; }
.cursor-crosshair .vue-flow__pane, .cursor-crosshair .vue-flow__node { cursor: crosshair !important; }
</style>
