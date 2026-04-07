/* Code generated from jsonrpc schema by rpcgen v2.5.x with typescript v1.0.0; DO NOT EDIT. */
/* eslint-disable */
// @ts-nocheck
export interface IAboutInfo {
  name: string,
  description: string,
  version: string,
  goVersion: string,
  target: string,
  author: string,
  license: string,
  website: string,
  github: string
}

export interface IAppDismissUpdateParams {
  version: string
}

export interface IAppImportDSNParams {
  dsn: string,
  schemas: Array<string>,
  tables: Array<string>,
  categories: Array<string>
}

export interface IAppIntrospectDSNParams {
  dsn: string
}

export interface IAppListDirectoryParams {
  path: string,
  showAll: boolean
}

export interface IAppOpenDemoParams {
  name: string
}

export interface IAppOpenFileParams {
  path: string
}

export interface IAppRegisterParams {
  email: string
}

export interface IAppRemoveRecentFileParams {
  path: string
}

export interface IAppRunDiffExampleParams {
  name: string
}

export interface ICheckDetail {
  name: string,
  expression: string
}

export interface ICheckInput {
  name: string,
  expression: string
}

export interface IColumnDetail {
  name: string,
  type: string,
  length: number,
  precision: number,
  scale: number,
  nullable: boolean,
  default: string,
  pk: boolean,
  fk: boolean,
  identity: string,
  identitySeqOpt?: IIdentitySeqOpt,
  generated: string,
  generatedStored: boolean,
  comment: string,
  compression: string,
  storage: string,
  collation: string
}

export interface IColumnInput {
  name: string,
  type: string,
  length: number,
  precision: number,
  scale: number,
  nullable: boolean,
  default: string,
  identity: string,
  identitySeqOpt?: IIdentitySeqOpt,
  generated: string,
  generatedStored: boolean,
  comment: string,
  compression: string,
  storage: string,
  collation: string
}

export interface IDSNObjectPreview {
  name: string,
  schema: string
}

export interface IDSNPreview {
  database: string,
  pgVersion: string,
  schemas: Array<IDSNSchemaPreview>,
  views: Array<IDSNObjectPreview>,
  matViews: Array<IDSNObjectPreview>,
  functions: Array<IDSNObjectPreview>,
  triggers: Array<IDSNObjectPreview>,
  enums: Array<IDSNObjectPreview>,
  domains: Array<IDSNObjectPreview>,
  sequences: Array<IDSNObjectPreview>,
  extensions: Array<IDSNObjectPreview>,
  roles: Array<IDSNRolePreview>,
  grants: number,
  defaultPrivileges: number
}

export interface IDSNRolePreview {
  name: string,
  login: boolean,
  members: number
}

export interface IDSNSchemaPreview {
  name: string,
  tables: Array<IDSNTablePreview>
}

export interface IDSNTablePreview {
  name: string,
  columns: number,
  indexes: number,
  fks: number,
  partitioned: boolean
}

export interface IDemoSchema {
  name: string,
  title: string,
  tables: number,
  fks: number
}

export interface IDiffChange {
  object: string, // table, column, index, fk, pk, unique, check, enum
  action: string, // add, drop, alter
  table: string, // parent table (for column/constraint changes)
  name: string, // object name
  sql: string, // generated ALTER/CREATE/DROP
  hazards: Array<IDiffHazard> // warnings
}

export interface IDiffExample {
  name: string,
  title: string,
  description: string
}

export interface IDiffHazard {
  level: string, // dangerous, warning, info
  code: string, // DELETES_DATA, TABLE_REWRITE, etc.
  message: string
}

export interface IDiffUnsavedResult {
  changes: Array<IDiffChange>,
  sql: string // full ALTER script
}

export interface IDirEntry {
  name: string,
  isDir: boolean,
  size: number,
  modTime: string,
  supported: boolean
}

export interface IDirectoryListing {
  path: string,
  entries: Array<IDirEntry>
}

export interface IERDColumn {
  name: string,
  type: string,
  pk: boolean,
  nn: boolean,
  fk: boolean,
  default: string
}

export interface IERDIndex {
  name: string
}

export interface IERDReference {
  name: string,
  from: string,
  fromCol: string,
  to: string,
  toCol: string
}

export interface IERDSchema {
  tables: Array<IERDTable>,
  references: Array<IERDReference>
}

export interface IERDTable {
  name: string,
  schema: string,
  x: number,
  y: number,
  columns: Array<IERDColumn>,
  indexes: Array<IERDIndex>,
  partitioned: boolean,
  partitionCount: number
}

export interface IExcludeDetail {
  name: string,
  using: string,
  elements: Array<IExcludeElementDetail>,
  where: string
}

export interface IExcludeElementDetail {
  column: string,
  expression: string,
  opclass: string,
  with: string
}

export interface IExcludeElementInput {
  column: string,
  expression: string,
  opclass: string,
  with: string
}

export interface IExcludeInput {
  name: string,
  using: string,
  elements: Array<IExcludeElementInput>,
  where: string
}

export interface IFKColDetail {
  name: string,
  references: string
}

export interface IFKColInput {
  name: string,
  references: string
}

export interface IFKDetail {
  name: string,
  toTable: string,
  onDelete: string,
  onUpdate: string,
  deferrable: boolean,
  initially: string,
  columns: Array<IFKColDetail>
}

export interface IFKInput {
  name: string,
  toTable: string,
  onDelete: string,
  onUpdate: string,
  deferrable: boolean,
  initially: string,
  columns: Array<IFKColInput>
}

export interface IFixLintResult {
  fixed: number,
  issues: Array<ILintIssue>
}

export interface IGeneralInput {
  name?: string,
  comment?: string,
  unlogged?: boolean,
  generate?: boolean
}

export interface IIdentitySeqOpt {
  start: number,
  increment: number,
  min: number,
  max: number,
  cache: number,
  cycle: boolean
}

export interface IIgnoredRule {
  code: string,
  title: string,
  scope: string // "project" or table name
}

export interface IIndexColDetail {
  name: string,
  order: string, // asc|desc
  nulls: string, // first|last
  opclass: string
}

export interface IIndexColInput {
  name: string,
  order: string,
  nulls: string,
  opclass: string
}

export interface IIndexDetail {
  name: string,
  unique: boolean,
  nullsDistinct: boolean,
  using: string,
  columns: Array<IIndexColDetail>,
  expressions: Array<string>,
  with: Array<IWithParamDetail>,
  where: string,
  include: Array<string>
}

export interface IIndexInput {
  name: string,
  table: string,
  unique: boolean,
  nullsDistinct: boolean,
  using: string,
  columns: Array<IIndexColInput>,
  expressions: Array<string>,
  with: Array<IWithParamInput>,
  where: string,
  include: Array<string>
}

export interface ILayoutPosition {
  name: string,
  schema: string,
  x: number,
  y: number
}

export interface ILintFixRequest {
  code: string,
  path: string
}

export interface ILintIssue {
  severity: string,
  code: string,
  title: string,
  path: string,
  message: string,
  fixable: boolean
}

export interface IObjectItem {
  name: string,
  kind: string, // table, column, index, fk, pk, unique, check, trigger, sequence, view, function, extension, domain, enum
  table: string // parent table name (for focusing on canvas)
}

export interface IPKDetail {
  name: string,
  columns: Array<string>
}

export interface IPKInput {
  name: string,
  columns: Array<string>
}

export interface IPartitionByRPC {
  type: string, // range | list | hash
  columns: Array<string>
}

export interface IPartitionRPC {
  name: string,
  bound: string
}

export interface IProjectCreateSchemaParams {
  name: string
}

export interface IProjectCreateTableParams {
  schemaName: string,
  tableName: string
}

export interface IProjectDeleteSchemaParams {
  name: string
}

export interface IProjectDeleteTableParams {
  name: string
}

export interface IProjectFixLintIssuesParams {
  issues: Array<ILintFixRequest>
}

export interface IProjectGenerateTestDataParams {
  seed: number,
  rows: number
}

export interface IProjectGetTableDDLParams {
  name: string
}

export interface IProjectGetTableParams {
  name: string
}

export interface IProjectIgnoreLintRulesParams {
  rules: Array<string>,
  table?: string
}

export interface IProjectInfo {
  name: string,
  pgVersion: string,
  tables: number,
  references: number,
  indexes: number,
  autoSave: boolean,
  schemas: Array<string>,
  defaultNullable: boolean,
  isDemo: boolean,
  isReadOnly: boolean,
  isRegistered: boolean,
  filePath: string,
  workDir: string
}

export interface IProjectLintTableParams {
  name: string
}

export interface IProjectMoveTableParams {
  name: string,
  toSchema: string
}

export interface IProjectPreviewDiffParams {
  name: string,
  general?: IGeneralInput,
  columns: Array<IColumnInput>,
  pk?: IPKInput,
  fks: Array<IFKInput>,
  uniques: Array<IUniqueInput>,
  checks: Array<ICheckInput>,
  excludes: Array<IExcludeInput>,
  indexes: Array<IIndexInput>
}

export interface IProjectSaveLayoutParams {
  positions: Array<ILayoutPosition>
}

export interface IProjectSaveProjectAsParams {
  path: string
}

export interface IProjectSaveTextFileParams {
  path: string,
  content: string
}

export interface IProjectSetAutoSaveParams {
  enabled: boolean
}

export interface IProjectSettings {
  name: string,
  description: string,
  pgVersion: string,
  defaultSchema: string,
  namingConvention: string, // Naming
  namingTables: string,
  defaultNullable: string, // Defaults
  defaultOnDelete: string,
  defaultOnUpdate: string,
  lintIgnoreRules: string, // Lint
  autoSaveDDL: string // Export
}

export interface IProjectSingularizeParams {
  word: string
}

export interface IProjectUnignoreLintRulesParams {
  rules: Array<string>,
  table?: string
}

export interface IProjectUpdateProjectSettingsParams {
  settings: IProjectSettings
}

export interface IProjectUpdateTableParams {
  name: string,
  general?: IGeneralInput,
  columns: Array<IColumnInput>,
  pk?: IPKInput,
  fks: Array<IFKInput>,
  uniques: Array<IUniqueInput>,
  checks: Array<ICheckInput>,
  excludes: Array<IExcludeInput>,
  indexes: Array<IIndexInput>,
  partitionBy?: IPartitionByRPC,
  partitions: Array<IPartitionRPC>
}

export interface IRecentFile {
  path: string,
  name: string,
  size: number,
  modTime: string,
  exists: boolean
}

export interface ITableDetail {
  name: string,
  schema: string,
  unlogged: boolean,
  tablespace: string,
  comment: string,
  columns: Array<IColumnDetail>,
  pk?: IPKDetail,
  uniques: Array<IUniqueDetail>,
  checks: Array<ICheckDetail>,
  excludes: Array<IExcludeDetail>,
  fks: Array<IFKDetail>,
  indexes: Array<IIndexDetail>,
  partitionBy?: IPartitionByRPC,
  partitions: Array<IPartitionRPC>,
  ddl: string
}

export interface ITypeInfo {
  name: string,
  category: string, // numeric, character, datetime, boolean, json, network, geometric, search, array, enum, composite, domain, system, other
  source: string // builtin, user
}

export interface IUniqueDetail {
  name: string,
  columns: Array<string>,
  nullsDistinct: boolean
}

export interface IUniqueInput {
  name: string,
  columns: Array<string>,
  nullsDistinct: boolean
}

export interface IUpdateInfo {
  currentVersion: string,
  latestVersion: string,
  updateAvailable: boolean,
  releaseURL: string,
  shouldNotify: boolean
}

export interface IWithParamDetail {
  name: string,
  value: string
}

export interface IWithParamInput {
  name: string,
  value: string
}

export const factory = (send: any) => ({
  app: {
    /**
     * About returns application metadata.
     */
    about(): Promise<IAboutInfo> {
      return send('app.About')
    },
    /**
     * CheckForUpdate checks GitHub Releases for a newer version of PgDesigner.
Results are cached for 24 hours. Safe to call in read-only mode.
     */
    checkForUpdate(): Promise<IUpdateInfo> {
      return send('app.CheckForUpdate')
    },
    /**
     * CloseProject replaces current project with empty one (returns to welcome screen).
     */
    closeProject(): Promise<boolean> {
      return send('app.CloseProject')
    },
    /**
     * DismissUpdate records that the user has dismissed the update notification for the given version.
     */
    dismissUpdate(params: IAppDismissUpdateParams): Promise<boolean> {
      return send('app.DismissUpdate', params)
    },
    /**
     * GetHomePath returns the user's home directory path.
     */
    getHomePath(): Promise<string> {
      return send('app.GetHomePath')
    },
    /**
     * GetRecentFiles returns the list of recently opened files.
     */
    getRecentFiles(): Promise<Array<string>> {
      return send('app.GetRecentFiles')
    },
    /**
     * GetRecentFilesInfo returns recent files with metadata (size, mod time, exists).
     */
    getRecentFilesInfo(): Promise<Array<IRecentFile>> {
      return send('app.GetRecentFilesInfo')
    },
    /**
     * ImportDSN imports schema from PostgreSQL with filtering options.
     */
    importDSN(params: IAppImportDSNParams): Promise<boolean> {
      return send('app.ImportDSN', params)
    },
    /**
     * IntrospectDSN connects to a PostgreSQL database and returns a preview of available objects.
     */
    introspectDSN(params: IAppIntrospectDSNParams): Promise<IDSNPreview> {
      return send('app.IntrospectDSN', params)
    },
    /**
     * ListDemoSchemas returns available embedded demo schemas.
     */
    listDemoSchemas(): Promise<Array<IDemoSchema>> {
      return send('app.ListDemoSchemas')
    },
    /**
     * ListDiffExamples returns available pre-built diff examples.
     */
    listDiffExamples(): Promise<Array<IDiffExample>> {
      return send('app.ListDiffExamples')
    },
    /**
     * ListDirectory lists files and subdirectories at the given path.
Returns entries sorted: directories first (alphabetical), then files (alphabetical).
Hidden files (starting with .) are excluded.
     */
    listDirectory(params: IAppListDirectoryParams): Promise<IDirectoryListing> {
      return send('app.ListDirectory', params)
    },
    /**
     * NewProject creates a new empty project, replacing the current one.
     */
    newProject(): Promise<boolean> {
      return send('app.NewProject')
    },
    /**
     * OpenDemo loads an embedded demo schema by name.
     */
    openDemo(params: IAppOpenDemoParams): Promise<boolean> {
      return send('app.OpenDemo', params)
    },
    /**
     * OpenFile opens a file by path, auto-converting if necessary.
     */
    openFile(params: IAppOpenFileParams): Promise<boolean> {
      return send('app.OpenFile', params)
    },
    /**
     * Ping cancels a pending shutdown (e.g. after page reload).

zenrpc
     */
    ping(): Promise<string> {
      return send('app.Ping')
    },
    /**
     * Quit starts a delayed shutdown. If Ping is not called within the grace period, the server exits.

zenrpc
     */
    quit(): Promise<void> {
      return send('app.Quit')
    },
    /**
     * Register sets the registered email (honor system, no validation).
     */
    register(params: IAppRegisterParams): Promise<boolean> {
      return send('app.Register', params)
    },
    /**
     * RemoveRecentFile removes a path from the recent files list.
     */
    removeRecentFile(params: IAppRemoveRecentFileParams): Promise<boolean> {
      return send('app.RemoveRecentFile', params)
    },
    /**
     * RunDiffExample loads a diff pair and returns the diff result.
     */
    runDiffExample(params: IAppRunDiffExampleParams): Promise<IDiffUnsavedResult> {
      return send('app.RunDiffExample', params)
    }
  },
  project: {
    /**
     * CreateSchema adds a new empty schema to the project.
     */
    createSchema(params: IProjectCreateSchemaParams): Promise<boolean> {
      return send('project.CreateSchema', params)
    },
    /**
     * CreateTable creates a new empty table in the specified schema.
     */
    createTable(params: IProjectCreateTableParams): Promise<boolean> {
      return send('project.CreateTable', params)
    },
    /**
     * DeleteSchema removes an empty schema from the project.
     */
    deleteSchema(params: IProjectDeleteSchemaParams): Promise<boolean> {
      return send('project.DeleteSchema', params)
    },
    /**
     * DeleteTable removes a table and its indexes from the project.
     */
    deleteTable(params: IProjectDeleteTableParams): Promise<boolean> {
      return send('project.DeleteTable', params)
    },
    /**
     * DiffUnsaved returns ALTER SQL for all unsaved changes (saved snapshot vs current state).
     */
    diffUnsaved(): Promise<IDiffUnsavedResult> {
      return send('project.DiffUnsaved')
    },
    /**
     * FixLintIssues applies auto-fixes for selected lint issues.
     */
    fixLintIssues(params: IProjectFixLintIssuesParams): Promise<IFixLintResult> {
      return send('project.FixLintIssues', params)
    },
    /**
     * GenerateTestData returns INSERT statements with fake test data.
     */
    generateTestData(params: IProjectGenerateTestDataParams): Promise<string> {
      return send('project.GenerateTestData', params)
    },
    /**
     * GetAutoSave reports whether auto-save is enabled.
     */
    getAutoSave(): Promise<boolean> {
      return send('project.GetAutoSave')
    },
    /**
     * GetDDL returns the full DDL for the project.
     */
    getDDL(): Promise<string> {
      return send('project.GetDDL')
    },
    /**
     * GetIgnoredRules returns all ignored lint rules from project and table settings.
     */
    getIgnoredRules(): Promise<Array<IIgnoredRule>> {
      return send('project.GetIgnoredRules')
    },
    /**
     * GetInfo returns project metadata.
     */
    getInfo(): Promise<IProjectInfo> {
      return send('project.GetInfo')
    },
    /**
     * GetProjectSettings returns editable project settings.
     */
    getProjectSettings(): Promise<IProjectSettings> {
      return send('project.GetProjectSettings')
    },
    /**
     * GetSchema returns the ERD schema for rendering in the frontend.
     */
    getSchema(): Promise<IERDSchema> {
      return send('project.GetSchema')
    },
    /**
     * GetTable returns full table data for the Table Editor.
     */
    getTable(params: IProjectGetTableParams): Promise<ITableDetail> {
      return send('project.GetTable', params)
    },
    /**
     * GetTableDDL returns the DDL for a single table (CREATE TABLE + indexes + FK + comments).
     */
    getTableDDL(params: IProjectGetTableDDLParams): Promise<string> {
      return send('project.GetTableDDL', params)
    },
    /**
     * IgnoreLintRules adds rules to project or table ignore list.
     */
    ignoreLintRules(params: IProjectIgnoreLintRulesParams): Promise<Array<ILintIssue>> {
      return send('project.IgnoreLintRules', params)
    },
    /**
     * IsDirty reports whether the project has unsaved changes.
     */
    isDirty(): Promise<boolean> {
      return send('project.IsDirty')
    },
    /**
     * Lint validates the project and returns lint issues.
     */
    lint(): Promise<Array<ILintIssue>> {
      return send('project.Lint')
    },
    /**
     * LintTable validates a single table and returns all lint issues.
     */
    lintTable(params: IProjectLintTableParams): Promise<Array<ILintIssue>> {
      return send('project.LintTable', params)
    },
    /**
     * ListObjects returns a flat list of all database objects for Go-To search.
     */
    listObjects(): Promise<Array<IObjectItem>> {
      return send('project.ListObjects')
    },
    /**
     * ListTypes returns available column types for autocomplete.
     */
    listTypes(): Promise<Array<ITypeInfo>> {
      return send('project.ListTypes')
    },
    /**
     * MoveTable transfers a table from its current schema to another.
     */
    moveTable(params: IProjectMoveTableParams): Promise<boolean> {
      return send('project.MoveTable', params)
    },
    /**
     * PreviewDiff returns ALTER SQL that would result from applying the given changes.
It does NOT modify the project — only computes the diff.
     */
    previewDiff(params: IProjectPreviewDiffParams): Promise<Array<IDiffChange>> {
      return send('project.PreviewDiff', params)
    },
    /**
     * SaveLayout updates table positions in the default layout.
     */
    saveLayout(params: IProjectSaveLayoutParams): Promise<boolean> {
      return send('project.SaveLayout', params)
    },
    /**
     * SaveProject writes the project to the .pgd file.
     */
    saveProject(): Promise<boolean> {
      return send('project.SaveProject')
    },
    /**
     * SaveProjectAs saves the project to a new file path.
     */
    saveProjectAs(params: IProjectSaveProjectAsParams): Promise<boolean> {
      return send('project.SaveProjectAs', params)
    },
    /**
     * SaveTextFile writes text content to the specified file path.
Used for saving DDL, diff patches, and other generated text.
     */
    saveTextFile(params: IProjectSaveTextFileParams): Promise<boolean> {
      return send('project.SaveTextFile', params)
    },
    /**
     * SetAutoSave enables or disables auto-save after each mutation.
     */
    setAutoSave(params: IProjectSetAutoSaveParams): Promise<boolean> {
      return send('project.SetAutoSave', params)
    },
    /**
     * Singularize returns the singular form of a word.
     */
    singularize(params: IProjectSingularizeParams): Promise<string> {
      return send('project.Singularize', params)
    },
    /**
     * UnignoreLintRules removes rules from project or table ignore list.
     */
    unignoreLintRules(params: IProjectUnignoreLintRulesParams): Promise<boolean> {
      return send('project.UnignoreLintRules', params)
    },
    /**
     * UpdateProjectSettings saves project-level settings.
     */
    updateProjectSettings(params: IProjectUpdateProjectSettingsParams): Promise<boolean> {
      return send('project.UpdateProjectSettings', params)
    },
    /**
     * UpdateTable applies changes to a table. Each section is optional (null = skip).
     */
    updateTable(params: IProjectUpdateTableParams): Promise<ITableDetail> {
      return send('project.UpdateTable', params)
    }
  }
})
