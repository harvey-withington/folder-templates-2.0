/** One line of a before/after pair, compared by position. */
export interface DiffLine {
  /** 1-based line number. */
  number: number
  before: string
  after: string
  changed: boolean
}

/**
 * Splits text into lines on \n, \r\n or \r. A trailing line break doesn't add
 * an empty last line, and empty text has no lines.
 */
export function splitLines(text: string): string[] {
  if (text === '') return []
  const lines = text.split(/\r\n|\n|\r/)
  if (lines.length > 1 && lines[lines.length - 1] === '') lines.pop()
  return lines
}

/**
 * Pairs lines by index. Token replacement never adds or removes line breaks,
 * so position is enough; a length mismatch pads the shorter side with "".
 */
export function diffLines(before: string, after: string): DiffLine[] {
  const b = splitLines(before)
  const a = splitLines(after)
  const out: DiffLine[] = []
  for (let i = 0; i < Math.max(b.length, a.length); i++) {
    const bl = b[i] ?? ''
    const al = a[i] ?? ''
    out.push({ number: i + 1, before: bl, after: al, changed: bl !== al || (b[i] === undefined) !== (a[i] === undefined) })
  }
  return out
}

export function changedLines(lines: DiffLine[]): DiffLine[] {
  return lines.filter((l) => l.changed)
}
