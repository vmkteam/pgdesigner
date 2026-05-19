import api from '@/api/factory'
import { pkColumnName } from '@/utils/naming'

/**
 * Creates a table with a default PK column whose case matches the project's
 * naming convention (e.g. snake_case → {singular}_id, camelCase → {singular}Id).
 * Returns the full table name (with schema prefix if non-default).
 */
export async function createTableWithPK(
  schemaName: string,
  tableName: string,
  defaultSchema: string,
  naming: string = '',
): Promise<string> {
  await api.project.createTable({ schemaName, tableName })

  const fullName = schemaName !== defaultSchema ? `${schemaName}.${tableName}` : tableName
  const singular = await api.project.singularize({ word: tableName })
  const pkColName = pkColumnName(singular || tableName, naming)

  await api.project.updateTable({
    name: fullName,
    columns: [{
      name: pkColName, type: 'integer', length: 0, precision: 0, scale: 0,
      nullable: false, default: '', pk: true, fk: false,
      identity: 'by-default', generated: '', generatedStored: false,
      comment: '', compression: '', storage: '', collation: '',
    }],
    pk: { name: `pk_${tableName}`, columns: [pkColName] },
  } as any)

  return fullName
}
