// Typed access to the Go service (window.go.main.App) and the Wails JS
// runtime (window.runtime). Declared here by hand rather than imported from
// the generated wailsjs/ folder, so the frontend type-checks and tests
// without running Wails first. Shapes mirror app/app.go and the engine.

import type {
  Inspection,
  NameHit,
  PreviewResult,
  RenderedFile,
  ScanResult,
  TemplateDescriptor,
  TreeEntry,
  ValidationIssue,
  Values,
} from '@harvey-withington/folder-templates-ui'

export type LaunchMode = 'home' | 'open' | 'edit' | 'pick'

export interface LaunchIntent {
  mode: LaunchMode
  source?: string
  target?: string
  extra?: string[]
}

export type LibrarySource = 'library' | 'recent' | 'pinned' | 'sample'

export interface LibraryEntry {
  path: string
  name: string
  description: string
  folderName: string
  paramCount: number
  source: LibrarySource
  pinned: boolean
  lastUsed?: string
  libraryFolder?: string
  missing?: boolean
  error?: string
}

export type Theme = 'system' | 'dark' | 'light'
export type ConflictChoice = 'refuse' | 'merge' | 'overwrite'

export interface Settings {
  theme: Theme
  libraryFolders: string[]
  recents: { path: string; lastUsed: string }[]
  pinned: string[]
  recentTargets: string[]
  defaultConflict: ConflictChoice
  openFolderAfter: boolean
  closeAfter: boolean
  showSamples: boolean
  window?: { width: number; height: number; maximised: boolean }
}

export interface GenerateRequest {
  dir: string
  target: string
  values: Values
  conflict: ConflictChoice
}

export interface GenerateResult {
  rootPath: string
  filesWritten: number
  skipped?: string[]
  overwritten?: string[]
  warnings?: string[]
}

export interface GenerateProgress {
  done: number
  total: number
  path: string
}

export type ShellFeature = 'sendToProcess' | 'sendToEdit' | 'folderMenu' | 'backgroundMenu'
export type ShellStatus = Record<ShellFeature, boolean>

/** The bound Go methods. Each returns a promise that rejects with the Go error text. */
export interface AppBindings {
  GetLaunchIntent(): Promise<LaunchIntent>
  Version(): Promise<string>
  Quit(): Promise<void>
  Inspect(dir: string): Promise<Inspection>
  Validate(desc: TemplateDescriptor): Promise<ValidationIssue[]>
  SaveTemplate(dir: string, desc: TemplateDescriptor): Promise<void>
  Preview(dir: string, desc: TemplateDescriptor | null, values: Values, target: string): Promise<PreviewResult>
  RenderFile(dir: string, desc: TemplateDescriptor | null, sourceRel: string, values: Values): Promise<RenderedFile>
  Tree(dir: string): Promise<TreeEntry[] | null>
  Scan(dir: string, desc: TemplateDescriptor | null): Promise<ScanResult>
  TestMatch(dir: string, pattern: string, replacement: string): Promise<NameHit[] | null>
  SetContentProcessing(dir: string, rel: string, on: boolean): Promise<string>
  Generate(req: GenerateRequest): Promise<GenerateResult>
  CancelGenerate(): Promise<void>
  PickFolder(title: string, initial: string): Promise<string>
  OpenFolder(path: string): Promise<void>
  RevealPath(path: string): Promise<void>
  ListLibrary(): Promise<LibraryEntry[]>
  RecordRecent(dir: string): Promise<void>
  RemoveRecent(path: string): Promise<void>
  SetPinned(path: string, pinned: boolean): Promise<void>
  GetSettings(): Promise<Settings>
  SaveSettings(s: Settings): Promise<Settings>
  SamplesFolder(): Promise<string>
  ShellStatus(): Promise<ShellStatus>
  SetShellFeature(feature: ShellFeature, on: boolean): Promise<ShellStatus>
}

export interface WailsRuntime {
  EventsOn(event: string, callback: (...data: unknown[]) => void): () => void
  OnFileDrop(callback: (x: number, y: number, paths: string[]) => void, useDropTarget: boolean): void
  OnFileDropOff(): void
  WindowSetTitle(title: string): void
}

declare global {
  interface Window {
    go?: { main?: { App?: AppBindings } }
    runtime?: WailsRuntime
  }
}

/** The Go service; throws if the page isn't running inside the Wails app. */
export function app(): AppBindings {
  const bindings = window.go?.main?.App
  if (!bindings) throw new Error('Folder Templates backend is not available (not running inside the app)')
  return bindings
}

export function wailsRuntime(): WailsRuntime | undefined {
  return window.runtime
}

/** Go errors arrive as plain strings; everything else as Error objects. */
export function errorText(err: unknown): string {
  if (typeof err === 'string') return err
  if (err instanceof Error) return err.message
  return String(err)
}
