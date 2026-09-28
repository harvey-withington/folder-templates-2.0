import type { Plugin } from 'vite'

/**
 * Rewrites `import { Folder, Trash2 } from 'lucide-svelte'` into per-icon
 * imports so tests compile only the icons used, not the whole barrel (which
 * takes about a minute). Host builds tree-shake the barrel themselves.
 */
export function lucidePerIcon(): Plugin {
  const kebab = (name: string) =>
    name
      .replace(/([a-z0-9])([A-Z])/g, '$1-$2')
      .replace(/([a-zA-Z])(\d)/g, '$1-$2')
      .toLowerCase()
  return {
    name: 'lucide-per-icon',
    enforce: 'pre',
    transform(code, id) {
      if (!/\.(svelte|ts)$/.test(id) || !code.includes('lucide-svelte')) return null
      return code.replace(/import\s*\{([^}]+)\}\s*from\s*['"]lucide-svelte['"]/g, (_all, names: string) =>
        names
          .split(',')
          .map((n) => n.trim())
          .filter(Boolean)
          .map((n) => {
            const [icon, local = icon] = n.split(/\s+as\s+/)
            return `import ${local} from 'lucide-svelte/icons/${kebab(icon)}'`
          })
          .join('\n'),
      )
    },
  }
}
