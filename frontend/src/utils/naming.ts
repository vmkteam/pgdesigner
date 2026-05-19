export type NamingConvention = 'snake_case' | 'camelCase' | 'PascalCase' | ''

function splitWords(s: string): string[] {
  return s
    .replace(/([a-z\d])([A-Z])/g, '$1_$2')
    .replace(/([A-Z]+)([A-Z][a-z])/g, '$1_$2')
    .split(/[_\s-]+/)
    .filter(Boolean)
}

export function toSnakeCase(s: string): string {
  return splitWords(s).map(w => w.toLowerCase()).join('_')
}

export function toCamelCase(s: string): string {
  const parts = splitWords(s)
  if (parts.length === 0) return ''
  const [first, ...rest] = parts
  return first!.toLowerCase() + rest.map(p => p.charAt(0).toUpperCase() + p.slice(1).toLowerCase()).join('')
}

export function toPascalCase(s: string): string {
  return splitWords(s).map(p => p.charAt(0).toUpperCase() + p.slice(1).toLowerCase()).join('')
}

export function pkColumnName(singular: string, naming: NamingConvention | string): string {
  switch (naming) {
    case 'camelCase':
      return `${toCamelCase(singular)}Id`
    case 'PascalCase':
      return `${toPascalCase(singular)}Id`
    case 'snake_case':
    default:
      return `${toSnakeCase(singular)}_id`
  }
}
