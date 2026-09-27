<script lang="ts" generics="T">
  import type { Snippet } from 'svelte'
  import { SvelteSet } from 'svelte/reactivity'
  import { ChevronRight } from 'lucide-svelte'
  import { visibleNodes, type TreeNode } from '../lib/tree'

  interface Props {
    nodes: TreeNode<T>[]
    /** Row content after the chevron. */
    row: Snippet<[TreeNode<T>]>
    /** Path of the selected node. */
    selected?: string
    onselect?: (node: TreeNode<T>) => void
    /** Which nodes can be selected; all by default. */
    selectable?: (node: TreeNode<T>) => boolean
    collapsible?: boolean
    /** Accessible name of the tree. */
    label: string
  }

  let {
    nodes,
    row,
    selected = $bindable(),
    onselect,
    selectable,
    collapsible = true,
    label,
  }: Props = $props()

  const collapsed = new SvelteSet<string>()
  const none: ReadonlySet<string> = new Set()
  const visible = $derived(visibleNodes(nodes, collapsible ? collapsed : none))

  let focused = $state<string | undefined>()
  let list: HTMLDivElement | undefined = $state()

  // Roving tabindex: one row is in the tab order.
  const tabStop = $derived.by(() => {
    for (const candidate of [focused, selected]) {
      if (candidate !== undefined && visible.some((n) => n.path === candidate)) return candidate
    }
    return visible[0]?.path
  })

  function isOpen(node: TreeNode<T>): boolean {
    return !collapsible || !collapsed.has(node.path)
  }

  function toggle(node: TreeNode<T>, open = !isOpen(node)) {
    if (!collapsible || !node.isDir) return
    if (open) collapsed.delete(node.path)
    else collapsed.add(node.path)
  }

  function activate(node: TreeNode<T>) {
    if (selectable && !selectable(node)) return
    selected = node.path
    onselect?.(node)
  }

  function focusAt(index: number) {
    const target = visible[Math.max(0, Math.min(index, visible.length - 1))]
    if (!target) return
    focused = target.path
    const el = list?.querySelectorAll<HTMLElement>('[role="treeitem"]')[visible.indexOf(target)]
    el?.focus()
  }

  function click(e: MouseEvent, node: TreeNode<T>) {
    focused = node.path
    if (e.target instanceof Element && e.target.closest('[data-chevron]')) toggle(node)
    else activate(node)
  }

  function keydown(e: KeyboardEvent, node: TreeNode<T>, index: number) {
    // Keys typed into controls inside a row (switches, buttons) are theirs.
    if (e.target !== e.currentTarget) return
    let handled = true
    switch (e.key) {
      case 'Enter':
      case ' ':
        activate(node)
        break
      case 'ArrowDown':
        focusAt(index + 1)
        break
      case 'ArrowUp':
        focusAt(index - 1)
        break
      case 'Home':
        focusAt(0)
        break
      case 'End':
        focusAt(visible.length - 1)
        break
      case 'ArrowRight':
        if (node.isDir && !isOpen(node)) toggle(node, true)
        else if (node.isDir && node.children.length) focusAt(index + 1)
        break
      case 'ArrowLeft':
        if (node.isDir && collapsible && isOpen(node)) toggle(node, false)
        else {
          const parent = node.path.slice(0, Math.max(0, node.path.lastIndexOf('/')))
          const at = visible.findIndex((n) => n.path === parent)
          if (at >= 0) focusAt(at)
        }
        break
      default:
        handled = false
    }
    if (handled) e.preventDefault()
  }
</script>

<div class="ft-tree" role="tree" aria-label={label} bind:this={list}>
  {#each visible as node, i (node.path)}
    <div
      class="item"
      class:selected={node.path === selected}
      role="treeitem"
      aria-level={node.depth + 1}
      aria-expanded={node.isDir && collapsible ? isOpen(node) : undefined}
      aria-selected={node.path === selected}
      tabindex={node.path === tabStop ? 0 : -1}
      style:--depth={node.depth}
      onclick={(e) => click(e, node)}
      onkeydown={(e) => keydown(e, node, i)}
      onfocus={() => (focused = node.path)}
    >
      {#if collapsible && node.isDir}
        <span class="chevron" class:open={isOpen(node)} data-chevron aria-hidden="true">
          <ChevronRight size={14} />
        </span>
      {:else}
        <span class="chevron" aria-hidden="true"></span>
      {/if}
      {@render row(node)}
    </div>
  {/each}
</div>

<style>
  .ft-tree {
    display: flex;
    flex-direction: column;
    font-size: 13px;
  }

  .item {
    display: flex;
    align-items: center;
    gap: 6px;
    min-height: 28px;
    padding: 2px 8px 2px calc(var(--depth) * 16px + 4px);
    border-radius: var(--ft-radius);
    color: var(--ft-text);
    cursor: default;
    user-select: none;
    outline: none;
  }

  .item:hover {
    background: var(--ft-subtle);
  }

  .item:focus-visible {
    box-shadow: inset 0 0 0 2px var(--ft-accent);
  }

  .item.selected {
    background: var(--ft-accent-soft);
    color: var(--ft-text-strong);
  }

  .chevron {
    display: inline-flex;
    flex: none;
    width: 16px;
    justify-content: center;
    color: var(--ft-text-muted);
    cursor: pointer;
    transition: transform var(--ft-duration);
  }

  .chevron.open {
    transform: rotate(90deg);
  }
</style>
