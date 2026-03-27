// shortcuts.ts — single source of truth for all keyboard shortcuts

export interface Shortcut {
  key: string       // display string: "Ctrl+S", "Enter", "↑/↓"
  action: string    // human-readable action
  context: string   // where it applies
}

export const shortcuts: Shortcut[] = [
  // Global (always active)
  { key: '?', action: 'Keyboard Reference', context: 'global' },
  { key: 'Ctrl+F', action: 'Go To (search)', context: 'global' },
  { key: 'Ctrl++', action: 'Zoom in', context: 'global' },
  { key: 'Ctrl+−', action: 'Zoom out', context: 'global' },
  { key: 'Ctrl+0', action: 'Reset zoom 100%', context: 'global' },
  { key: 'Ctrl+L', action: 'Check Diagram', context: 'global' },
  { key: 'Ctrl+G', action: 'Generate DDL', context: 'global' },
  { key: 'Ctrl+,', action: 'Project Settings', context: 'global' },
  { key: 'Ctrl+Shift+D', action: 'Toggle Dark Theme', context: 'global' },

  // Canvas tools
  { key: 'T', action: 'Create Table tool (toggle)', context: 'canvas' },
  { key: 'F', action: 'Create FK tool (toggle)', context: 'canvas' },
  { key: 'M', action: 'Create M:N tool (toggle)', context: 'canvas' },
  { key: 'Escape', action: 'Reset tool to pointer', context: 'canvas' },
  { key: 'Ctrl+A', action: 'Select all tables', context: 'canvas' },
  { key: 'Ctrl+C / Ctrl+V', action: 'Copy / paste tables', context: 'canvas' },
  { key: 'Delete', action: 'Delete selected tables', context: 'canvas' },

  // Table Editor dialog
  { key: 'Ctrl+S', action: 'Save project', context: 'editor' },
  { key: 'Ctrl+Enter', action: 'Apply changes', context: 'editor' },
  { key: 'Escape', action: 'Close editor', context: 'editor' },
  { key: 'Ctrl+1…6', action: 'Switch tab', context: 'editor' },

  // Column grid — navigation mode
  { key: '↑ / ↓', action: 'Select row', context: 'grid' },
  { key: '← / →', action: 'Move between cells', context: 'grid' },
  { key: 'Enter / F2', action: 'Edit cell', context: 'grid' },
  { key: 'Space', action: 'Toggle NN / PK', context: 'grid' },
  { key: 'Delete', action: 'Clear cell value', context: 'grid' },
  { key: '+', action: 'Add column', context: 'grid' },
  { key: '− / Del', action: 'Delete column', context: 'grid' },
  { key: 'Ctrl+↑ / Ctrl+↓', action: 'Move column up/down', context: 'grid' },

  // Constraints / Indexes / FK lists
  { key: '↑ / ↓', action: 'Select item', context: 'constraints' },
  { key: '+', action: 'Add', context: 'constraints' },
  { key: '−', action: 'Delete', context: 'constraints' },

  // Column grid — edit mode
  { key: 'Tab', action: 'Commit & next cell →', context: 'grid-edit' },
  { key: 'Shift+Tab', action: 'Commit & prev cell ←', context: 'grid-edit' },
  { key: 'Enter', action: 'Commit & next row ↓', context: 'grid-edit' },
  { key: 'Escape', action: 'Cancel edit', context: 'grid-edit' },
]

// Group shortcuts by context for display
export function shortcutsByContext(): Record<string, Shortcut[]> {
  const groups: Record<string, Shortcut[]> = {}
  for (const s of shortcuts) {
    ;(groups[s.context] ??= []).push(s)
  }
  return groups
}

// Context display names
export const contextNames: Record<string, string> = {
  global: 'Global',
  canvas: 'Canvas',
  editor: 'Table Editor',
  grid: 'Column Grid',
  'grid-edit': 'Column Grid (editing)',
  constraints: 'Constraints / Indexes / FK',
}

// Short hints for footer — only the most important per context
const contextHints: Record<string, string> = {
  grid: '↑↓←→: Navigate  │  Enter: Edit  │  Space: Toggle  │  +/−: Add/Del  │  ?: Help',
  'grid-edit': 'Enter: Commit ↓  │  Tab: Next →  │  Esc: Cancel',
  constraints: '↑↓: Select  │  +: Add  │  −: Delete  │  ?: Help',
  editor: 'Ctrl+S: Save  │  Ctrl+Enter: Apply  │  Esc: Close  │  ?: Help',
}

export function statusBarHints(context: string): string {
  return contextHints[context] || ''
}
