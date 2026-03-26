import HttpRpcClient from './HttpRpcClient'
import { factory } from './factory.generated'

const client = new HttpRpcClient({ url: '/rpc/' })

const api = factory(client.call)

export default api

// Re-export types from generated file
export type {
  IProjectInfo,
  IERDSchema,
  IERDTable,
  IERDColumn,
  IERDIndex,
  IERDReference,
  ILintIssue,
  ITableDetail,
  IColumnDetail,
  IPKDetail,
  IUniqueDetail,
  ICheckDetail,
  IExcludeDetail,
  IFKDetail,
  IFKColDetail,
  IIndexDetail,
  IObjectItem,
  ITypeInfo,
  IExcludeElementDetail,
  IDiffChange,
  IDiffHazard,
  IDiffUnsavedResult,
  IIgnoredRule,
  IAboutInfo,
  IIdentitySeqOpt,
  IIndexColDetail,
  IPartitionByRPC,
  IPartitionRPC,
  IProjectSettings,
  IProjectUpdateTableParams,
  IProjectPreviewDiffParams,
  IProjectGenerateTestDataParams,
  IDemoSchema,
  IDiffExample,
  IDirEntry,
  IDirectoryListing,
  IRecentFile,
  IDSNPreview,
  IDSNSchemaPreview,
  IDSNTablePreview,
  IDSNObjectPreview,
  IDSNRolePreview,
} from './factory.generated'
