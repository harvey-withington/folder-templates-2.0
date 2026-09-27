// Shapes mirror the Go engine's JSON (github.com/harvey-withington/foldertemplate).
// Paths are slash-separated and relative unless a field says otherwise.

/** One template parameter, exactly as stored in .ft/template.json (camelCase). */
export interface Parameter {
  /** Empty for a nameless rename rule (match-only). */
  name: string
  type: string
  /** null/empty → internal: never asked, uses defaultValue. */
  prompt: string | null
  placeholder: string | null
  defaultValue: string | null
  /** null/empty → the default `\{name\}`. .NET regex syntax. */
  match: string | null
  replaceInFileNames: boolean
  replaceInFiles: boolean
}

/** The descriptor in .ft/template.json. */
export interface TemplateDescriptor {
  name: string
  description: string
  defaultTargetPath: string
  parameters: Parameter[]
}

/** Answers keyed by parameter name. */
export type Values = Record<string, string>

export type ConflictPolicy = 'refuse' | 'merge' | 'overwrite'

/** What a folder is: a template, or a plain folder that could become one. */
export interface Inspection {
  /** Absolute path. */
  dir: string
  folderName: string
  isTemplate: boolean
  template?: TemplateDescriptor
  /** .ft/template.json exists but could not be parsed. */
  loadError?: string
  issues: string[]
  /** Absolute; set for templates. */
  defaultTarget?: string
  files: number
  dirs: number
  sizeBytes: number
}

/** One would-be output item from a dry run. */
export interface PreviewEntry {
  /** Relative to the template folder; "" for the root. */
  sourceRel: string
  /** Relative to the target folder, starting with the root name. */
  outputRel: string
  isDir: boolean
  /** A .ft$ file: tokens filled in, suffix dropped. */
  processed: boolean
  /** Already on disk at the target. */
  exists?: boolean
}

export interface PreviewResult {
  rootName: string
  /** Absolute. */
  rootPath: string
  rootExists: boolean
  entries: PreviewEntry[]
  warnings: string[]
}

export interface RenderedFile {
  before: string
  after: string
}

/** One item of a template's source tree (.ft/ excluded). */
export interface TreeEntry {
  rel: string
  isDir: boolean
  size: number
  processed: boolean
}

export interface TokenInfo {
  name: string
  /** "." is the template folder's own name. */
  inNames: string[]
  inContent: string[]
  declared: boolean
}

export interface ParamUsage {
  name: string
  match: string
  nameHits: number
  contentHits: number
}

export type ScanIssueKind =
  | 'token-in-unprocessed-file'
  | 'undeclared-token'
  | 'unused-parameter'
  | 'name-replacement-off'
  | 'content-replacement-off'
  | 'unprocessable-content-file'

export interface ScanIssue {
  kind: ScanIssueKind
  param?: string
  token?: string
  path?: string
  message: string
}

export interface ScanResult {
  tokens: TokenInfo[]
  params: ParamUsage[]
  issues: ScanIssue[]
}

/** A descriptor problem; param is "" for problems not tied to one parameter. */
export interface ValidationIssue {
  param: string
  message: string
}

export interface NameHit {
  /** "." is the template folder itself. */
  sourceRel: string
  name: string
  result: string
  isDir: boolean
}

/**
 * Everything the components need from the host. The desktop app implements it
 * over Wails bindings, BRUV over its HTTP API. `dir` is always the template
 * folder (absolute, or whatever reference the host resolves).
 *
 * `descriptor`, where accepted, is an unsaved copy from the editor; omit it to
 * use the template as saved on disk.
 */
export interface TemplateBackend {
  inspect(dir: string): Promise<Inspection>
  /** The engine's checks (unique names, .NET regex patterns) on a descriptor. */
  validate(descriptor: TemplateDescriptor): Promise<ValidationIssue[]>
  save(dir: string, descriptor: TemplateDescriptor): Promise<void>
  preview(dir: string, values: Values, target: string, descriptor?: TemplateDescriptor): Promise<PreviewResult>
  renderFile(dir: string, sourceRel: string, values: Values, descriptor?: TemplateDescriptor): Promise<RenderedFile>
  tree(dir: string): Promise<TreeEntry[]>
  scan(dir: string, descriptor?: TemplateDescriptor): Promise<ScanResult>
  testMatch(dir: string, pattern: string, replacement: string): Promise<NameHit[]>
  setContentProcessing(dir: string, rel: string, on: boolean): Promise<string>
  /** Native folder picker; null when cancelled. */
  pickFolder(title: string, initial?: string): Promise<string | null>
}
