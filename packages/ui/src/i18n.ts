/** Nested message catalogue, e.g. { ft: { generate: { create: "Create {name}" } } }. */
export interface Messages {
  [key: string]: string | Messages
}

export type Translate = (key: string, vars?: Record<string, string | number>) => string

/**
 * Builds a translator over a catalogue. Keys are dotted paths ("ft.form.target");
 * `{name}` placeholders are filled from vars. A missing key returns the key
 * itself so gaps are visible rather than blank.
 */
export function createTranslator(messages: Messages): Translate {
  return (key, vars) => {
    let node: string | Messages | undefined = messages
    for (const part of key.split('.')) {
      if (typeof node !== 'object' || node === null) {
        node = undefined
        break
      }
      node = node[part]
    }
    if (typeof node !== 'string') return key
    if (!vars) return node
    return node.replace(/\{(\w+)\}/g, (whole, name: string) =>
      Object.prototype.hasOwnProperty.call(vars, name) ? String(vars[name]) : whole,
    )
  }
}
