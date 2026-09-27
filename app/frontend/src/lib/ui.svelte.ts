import type { Theme } from './bridge'

// --- theme ---------------------------------------------------------------------

let mediaListener: ((e: MediaQueryListEvent) => void) | null = null

/** Applies the theme to <html data-theme>, following the OS when "system". */
export function applyTheme(theme: Theme): void {
  const root = document.documentElement
  const media = window.matchMedia?.('(prefers-color-scheme: light)')
  if (media && mediaListener) media.removeEventListener('change', mediaListener)
  mediaListener = null

  const set = (light: boolean) => {
    if (light) root.dataset.theme = 'light'
    else delete root.dataset.theme
  }
  if (theme === 'system') {
    set(!!media?.matches)
    if (media) {
      mediaListener = (e) => set(e.matches)
      media.addEventListener('change', mediaListener)
    }
  } else {
    set(theme === 'light')
  }
}

// --- toasts ----------------------------------------------------------------------

export type ToastKind = 'info' | 'success' | 'error'

export interface Toast {
  id: number
  message: string
  kind: ToastKind
}

class Toasts {
  items = $state<Toast[]>([])
  #next = 0

  show(message: string, kind: ToastKind = 'info'): void {
    this.#next += 1
    const id = this.#next
    this.items.push({ id, message, kind })
    setTimeout(() => this.dismiss(id), kind === 'error' ? 8000 : 4000)
  }

  dismiss(id: number): void {
    this.items = this.items.filter((t) => t.id !== id)
  }
}

export const toasts = new Toasts()

// --- confirm dialog ----------------------------------------------------------------

export interface ConfirmOptions {
  confirmLabel?: string
  danger?: boolean
}

interface ConfirmRequest extends ConfirmOptions {
  message: string
  resolve: (ok: boolean) => void
}

class Confirmer {
  current = $state<ConfirmRequest | null>(null)

  ask(message: string, options: ConfirmOptions = {}): Promise<boolean> {
    // A new question answers any open one with "no".
    this.current?.resolve(false)
    return new Promise((resolve) => {
      this.current = { message, ...options, resolve }
    })
  }

  answer(ok: boolean): void {
    const req = this.current
    this.current = null
    req?.resolve(ok)
  }
}

export const confirmer = new Confirmer()
