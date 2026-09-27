import { createTranslator } from '../i18n'
import en from '../locales/en.json'
import { getFolderTemplatesContext, type ResolvedHost } from '../context'

/** The non-backend host services. */
export type UiHost = Omit<ResolvedHost, 'backend'>

let fallback: UiHost | undefined

/**
 * Host services for presentational components: the context's when a host set
 * one, otherwise English strings, console toasts and auto-confirm, so those
 * components also work standalone. Call during component initialisation.
 */
export function useUiHost(): UiHost {
  try {
    const { t, toast, confirm } = getFolderTemplatesContext()
    return { t, toast, confirm }
  } catch {
    fallback ??= {
      t: createTranslator(en),
      toast: (message, kind) => (kind === 'error' ? console.error(message) : console.info(message)),
      confirm: () => Promise.resolve(true),
    }
    return fallback
  }
}

/** Picks `${key}.one` or `${key}.other` by count and fills {count}. */
export function plural(t: UiHost['t'], key: string, count: number): string {
  return t(`${key}.${count === 1 ? 'one' : 'other'}`, { count })
}
