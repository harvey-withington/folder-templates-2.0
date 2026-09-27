import type { Parameter, TemplateDescriptor, TokenInfo, Values } from '../types'

/** Parameters a person is asked for: named, with a non-empty prompt. */
export function promptedParameters(parameters: Parameter[]): Parameter[] {
  return parameters.filter((p) => p.name !== '' && !!p.prompt)
}

export function isInternal(p: Parameter): boolean {
  return !p.prompt
}

/**
 * Starting answers for a form: every prompted parameter's default (or "").
 * Answers already in `existing` win, so re-rendering keeps what was typed.
 */
export function initialValues(parameters: Parameter[], existing: Values = {}): Values {
  const out: Values = {}
  for (const p of promptedParameters(parameters)) {
    out[p.name] = existing[p.name] ?? p.defaultValue ?? ''
  }
  return out
}

/**
 * Answers for a live preview. A blank answer that goes into names would make
 * the engine refuse outright (a folder called "{name}" resolves to no name at
 * all), so for previewing, blanks show as a marker instead: the tree reads
 * "‹name›" until it is filled in. Not "{name}": that is often the template
 * folder's own name, and previewing into the folder that holds the template
 * would then report the template itself as "already exists". Only the preview
 * does this; a real run keeps the blank and the host blocks it.
 */
export function previewValues(parameters: Parameter[], values: Values): Values {
  const out: Values = { ...values }
  for (const p of promptedParameters(parameters)) {
    if (p.replaceInFileNames && !(values[p.name] ?? '').trim()) out[p.name] = `‹${p.name}›`
  }
  return out
}

/** The engine's default name pattern for a parameter: the literal {name}. */
export function defaultMatch(name: string): string {
  return '\\{' + name.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + '\\}'
}

export function blankParameter(): Parameter {
  return {
    name: '',
    type: 'text',
    prompt: '',
    placeholder: null,
    defaultValue: null,
    match: null,
    replaceInFileNames: true,
    replaceInFiles: true,
  }
}

export function blankDescriptor(name = ''): TemplateDescriptor {
  return { name, description: '', defaultTargetPath: '', parameters: [] }
}

/** "videoName" / "video_name" / "video-name" → "Video name". */
export function humanize(token: string): string {
  const words = token
    .replace(/([a-z0-9])([A-Z])/g, '$1 $2')
    .replace(/[_-]+/g, ' ')
    .trim()
    .toLowerCase()
  return words.charAt(0).toUpperCase() + words.slice(1)
}

/** A parameter pre-wired for a token the scanner found. */
export function parameterFromToken(token: TokenInfo): Parameter {
  return {
    ...blankParameter(),
    name: token.name,
    prompt: humanize(token.name),
    replaceInFileNames: token.inNames.length > 0 || token.inContent.length === 0,
    replaceInFiles: token.inContent.length > 0,
  }
}

/**
 * Editor rows carry a stable id so reordering and deleting never confuse
 * Svelte's keyed lists (names are editable, so they can't be the key).
 */
export interface ParameterRow {
  id: string
  param: Parameter
  /** Prompt remembered while the row is switched to internal. */
  savedPrompt: string
  /**
   * Never asked; saved with a null prompt. Kept apart from the prompt so a new
   * "asked" row with a still-empty prompt doesn't read as internal.
   */
  internal: boolean
}

let nextId = 0
export function newRowId(): string {
  nextId += 1
  return `p${nextId}`
}

export function toRows(parameters: Parameter[]): ParameterRow[] {
  return parameters.map((param) => ({
    id: newRowId(),
    param: { ...param },
    savedPrompt: param.prompt ?? '',
    internal: isInternal(param),
  }))
}

/** A fresh editor row for a new, asked parameter. */
export function blankRow(): ParameterRow {
  return { id: newRowId(), param: blankParameter(), savedPrompt: '', internal: false }
}

export function fromRows(rows: ParameterRow[]): Parameter[] {
  return rows.map((r) => normalizeParameter(r.internal ? { ...r.param, prompt: null } : r.param))
}

/** Empty optional strings are stored as null, the way the C# app wrote them. */
export function normalizeParameter(p: Parameter): Parameter {
  const nullIfEmpty = (s: string | null) => (s === null || s === '' ? null : s)
  return {
    name: p.name.trim(),
    type: p.type || 'text',
    prompt: nullIfEmpty(p.prompt),
    placeholder: nullIfEmpty(p.placeholder),
    defaultValue: p.defaultValue,
    match: nullIfEmpty(p.match),
    replaceInFileNames: p.replaceInFileNames,
    replaceInFiles: p.replaceInFiles,
  }
}

/** Moves the item at `from` to position `to` (both in the original array). */
export function move<T>(list: T[], from: number, to: number): T[] {
  if (from === to || from < 0 || from >= list.length) return list
  const copy = list.slice()
  const [item] = copy.splice(from, 1)
  copy.splice(Math.max(0, Math.min(to, copy.length)), 0, item)
  return copy
}

/** Descriptors compare equal when their saved JSON would be identical. */
export function sameDescriptor(a: TemplateDescriptor, b: TemplateDescriptor): boolean {
  const norm = (d: TemplateDescriptor) =>
    JSON.stringify({ ...d, parameters: d.parameters.map(normalizeParameter) })
  return norm(a) === norm(b)
}
