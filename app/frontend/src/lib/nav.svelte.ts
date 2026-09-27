export type View = 'library' | 'generate' | 'editor' | 'settings'

export interface NavTarget {
  view: View
  /** Template (or folder to templatize) for generate/editor. */
  dir?: string | null
  /** Destination to preselect in the generate view. */
  target?: string | null
}

/** A view with unsaved work registers a guard; it resolves true to allow leaving. */
export type LeaveGuard = () => Promise<boolean>

class Navigation {
  view = $state<View>('library')
  dir = $state<string | null>(null)
  target = $state<string | null>(null)
  /** Set by the "New from template here…" menu: every generate goes here. */
  pickTarget = $state<string | null>(null)
  /** The folder the app was launched for (SendTo), if any. */
  launchedFor = $state<string | null>(null)

  #guard: LeaveGuard | null = null

  setGuard(guard: LeaveGuard | null): void {
    this.#guard = guard
  }

  async go(to: NavTarget): Promise<boolean> {
    if (this.#guard && !(await this.#guard())) return false
    this.#guard = null
    this.view = to.view
    if (to.dir !== undefined) this.dir = to.dir
    if (to.target !== undefined) this.target = to.target
    return true
  }
}

export const nav = new Navigation()
