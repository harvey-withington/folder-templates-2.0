/**
 * Runs the latest call after `ms` of quiet. `cancel` drops a pending call;
 * `flush` runs it now.
 */
export function debounce<A extends unknown[]>(fn: (...args: A) => void, ms: number) {
  let timer: ReturnType<typeof setTimeout> | undefined
  let pending: A | undefined
  const run = () => {
    timer = undefined
    const args = pending
    pending = undefined
    if (args) fn(...args)
  }
  const debounced = (...args: A) => {
    pending = args
    if (timer !== undefined) clearTimeout(timer)
    timer = setTimeout(run, ms)
  }
  debounced.cancel = () => {
    if (timer !== undefined) clearTimeout(timer)
    timer = undefined
    pending = undefined
  }
  debounced.flush = () => {
    if (timer !== undefined) clearTimeout(timer)
    run()
  }
  return debounced
}

/**
 * Tracks async requests so only the newest result is applied — a slow
 * preview for "ab" must not overwrite the one for "abc".
 */
export function latestOnly() {
  let seq = 0
  return {
    next(): number {
      seq += 1
      return seq
    },
    isLatest(ticket: number): boolean {
      return ticket === seq
    },
  }
}
