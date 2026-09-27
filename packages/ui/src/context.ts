import { getContext, setContext } from 'svelte'
import type { TemplateBackend } from './types'
import { createTranslator, type Translate } from './i18n'
import en from './locales/en.json'

export type ToastKind = 'info' | 'success' | 'error'

/** Host services the components call instead of importing them. */
export interface FolderTemplatesHost {
  backend: TemplateBackend
  /** Translator for `ft.*` keys. Defaults to the bundled English strings. */
  t?: Translate
  /** Transient notification. Defaults to console output. */
  toast?: (message: string, kind?: ToastKind) => void
  /** In-app confirmation (never window.confirm). Resolves true to proceed. */
  confirm?: (message: string, options?: { confirmLabel?: string; danger?: boolean }) => Promise<boolean>
}

export interface ResolvedHost {
  backend: TemplateBackend
  t: Translate
  toast: (message: string, kind?: ToastKind) => void
  confirm: (message: string, options?: { confirmLabel?: string; danger?: boolean }) => Promise<boolean>
}

const KEY = Symbol('folder-templates-host')

/** Call once in a host component above any folder-templates component. */
export function setFolderTemplatesContext(host: FolderTemplatesHost): ResolvedHost {
  const resolved: ResolvedHost = {
    backend: host.backend,
    t: host.t ?? createTranslator(en),
    toast: host.toast ?? ((message, kind) => (kind === 'error' ? console.error(message) : console.info(message))),
    confirm: host.confirm ?? (() => Promise.resolve(true)),
  }
  setContext(KEY, resolved)
  return resolved
}

export function getFolderTemplatesContext(): ResolvedHost {
  const host = getContext<ResolvedHost | undefined>(KEY)
  if (!host) {
    throw new Error('folder-templates-ui: call setFolderTemplatesContext() in a parent component first')
  }
  return host
}
