/** A node in a nested tree built from flat slash-separated paths. */
export interface TreeNode<T> {
  /** Full slash-separated path; unique within the tree. */
  path: string
  /** Last path segment. */
  name: string
  isDir: boolean
  depth: number
  /** The flat item this node came from; undefined for folders only implied by a child's path. */
  item?: T
  children: TreeNode<T>[]
}

/**
 * Nests flat items by their slash-separated paths. Parents missing from the
 * input are created as folders. Children keep folders first, then names in
 * case-insensitive order — the order Explorer shows.
 */
export function buildTree<T>(items: T[], pathOf: (item: T) => string, isDirOf: (item: T) => boolean): TreeNode<T>[] {
  const roots: TreeNode<T>[] = []
  const byPath = new Map<string, TreeNode<T>>()

  const ensure = (path: string, isDir: boolean): TreeNode<T> => {
    const existing = byPath.get(path)
    if (existing) {
      if (isDir) existing.isDir = true
      return existing
    }
    const slash = path.lastIndexOf('/')
    const parentPath = slash >= 0 ? path.slice(0, slash) : ''
    const parent = parentPath ? ensure(parentPath, true) : undefined
    const node: TreeNode<T> = {
      path,
      name: slash >= 0 ? path.slice(slash + 1) : path,
      isDir,
      depth: parent ? parent.depth + 1 : 0,
      children: [],
    }
    byPath.set(path, node)
    ;(parent ? parent.children : roots).push(node)
    return node
  }

  for (const item of items) {
    const path = pathOf(item).replace(/\/+$/, '')
    if (!path) continue
    const node = ensure(path, isDirOf(item))
    node.item = item
  }

  const sort = (nodes: TreeNode<T>[]) => {
    nodes.sort((a, b) => {
      if (a.isDir !== b.isDir) return a.isDir ? -1 : 1
      return a.name.localeCompare(b.name, undefined, { sensitivity: 'base', numeric: true })
    })
    nodes.forEach((n) => sort(n.children))
  }
  sort(roots)
  return roots
}

/** Depth-first list of the nodes visible when `collapsed` folders hide their children. */
export function visibleNodes<T>(roots: TreeNode<T>[], collapsed: ReadonlySet<string>): TreeNode<T>[] {
  const out: TreeNode<T>[] = []
  const walk = (nodes: TreeNode<T>[]) => {
    for (const n of nodes) {
      out.push(n)
      if (n.isDir && !collapsed.has(n.path)) walk(n.children)
    }
  }
  walk(roots)
  return out
}

/** Last segment of a slash-separated path. */
export function baseName(path: string): string {
  const trimmed = path.replace(/\/+$/, '')
  return trimmed.slice(trimmed.lastIndexOf('/') + 1)
}

/** Suffix marking a file whose contents get their tokens filled in. */
export const FT_SUFFIX = '.ft$'

/** "notes.md.ft$" → "notes.md"; other names are unchanged. */
export function stripFtSuffix(name: string): string {
  return name.endsWith(FT_SUFFIX) ? name.slice(0, -FT_SUFFIX.length) : name
}
