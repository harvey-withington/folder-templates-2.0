// Components
export { default as ParameterForm } from './components/ParameterForm.svelte'
export { default as TreeView } from './components/TreeView.svelte'
export { default as PreviewTree } from './components/PreviewTree.svelte'
export { default as FileDiffView } from './components/FileDiffView.svelte'
export { default as ParameterRow } from './components/ParameterRow.svelte'
export { default as ParameterDetails } from './components/ParameterDetails.svelte'
export { default as TemplateEditor } from './components/TemplateEditor.svelte'
export { default as LivePreview } from './components/LivePreview.svelte'
export { default as TokenScanPanel } from './components/TokenScanPanel.svelte'
export { default as ContentToggleTree } from './components/ContentToggleTree.svelte'
export { default as RegexTester } from './components/RegexTester.svelte'

// Data shapes and the backend contract
export type * from './types'

// Host services
export { setFolderTemplatesContext, getFolderTemplatesContext } from './context'
export type { FolderTemplatesHost, ResolvedHost, ToastKind } from './context'
export { createTranslator } from './i18n'
export type { Messages, Translate } from './i18n'
export { default as en } from './locales/en.json'

// Helpers
export { buildTree, visibleNodes, baseName, stripFtSuffix, FT_SUFFIX } from './lib/tree'
export type { TreeNode } from './lib/tree'
export {
  promptedParameters,
  isInternal,
  initialValues,
  previewValues,
  defaultMatch,
  blankParameter,
  blankDescriptor,
  blankRow,
  humanize,
  parameterFromToken,
  newRowId,
  toRows,
  fromRows,
  normalizeParameter,
  move,
  sameDescriptor,
} from './lib/params'
export type { ParameterRow as ParameterRowData } from './lib/params'
export { debounce, latestOnly } from './lib/debounce'
export { diffLines, changedLines, splitLines } from './lib/diff'
export type { DiffLine } from './lib/diff'
export { errorMessage, cloneJson } from './lib/errors'
export { useUiHost, plural } from './lib/host'
export type { UiHost } from './lib/host'
