<script lang="ts">
  import { untrack } from 'svelte'
  import { SvelteSet } from 'svelte/reactivity'
  import { FileText, Folder, LoaderCircle } from 'lucide-svelte'
  import type { TreeEntry } from '../types'
  import { getFolderTemplatesContext } from '../context'
  import { latestOnly } from '../lib/debounce'
  import { errorMessage } from '../lib/errors'
  import { buildTree, FT_SUFFIX, stripFtSuffix, type TreeNode } from '../lib/tree'
  import TreeView from './TreeView.svelte'

  interface Props {
    /** Template folder. */
    dir: string
    /** After a file was switched and the tree reloaded. */
    onchanged?: () => void
  }

  let { dir, onchanged }: Props = $props()

  const { backend, t, toast } = getFolderTemplatesContext()

  let entries = $state<TreeEntry[] | null>(null)
  let error = $state<string | null>(null)
  let loading = $state(false)
  const pending = new SvelteSet<string>()

  const nodes = $derived(
    buildTree(
      entries ?? [],
      (e) => e.rel,
      (e) => e.isDir,
    ),
  )

  const latest = latestOnly()
  async function load(forDir: string) {
    const ticket = latest.next()
    loading = true
    try {
      const next = await backend.tree(forDir)
      if (!latest.isLatest(ticket)) return
      entries = next
      error = null
    } catch (e) {
      if (!latest.isLatest(ticket)) return
      error = errorMessage(e)
    }
    loading = false
  }

  $effect(() => {
    const d = dir
    untrack(() => void load(d))
  })

  async function toggle(entry: TreeEntry) {
    if (pending.has(entry.rel)) return
    pending.add(entry.rel)
    try {
      await backend.setContentProcessing(dir, entry.rel, !entry.processed)
      await load(dir)
      onchanged?.()
    } catch (e) {
      toast(errorMessage(e), 'error')
    } finally {
      pending.delete(entry.rel)
    }
  }

  const displayName = (node: TreeNode<TreeEntry>) => (node.item?.processed ? stripFtSuffix(node.name) : node.name)
</script>

{#snippet row(node: TreeNode<TreeEntry>)}
  <span class="icon" class:dir={node.isDir}>
    {#if node.isDir}<Folder size={15} />{:else}<FileText size={15} />{/if}
  </span>
  <span class="name">{displayName(node)}</span>
  {#if node.item?.processed}
    <span class="badge ft-mono">{FT_SUFFIX}</span>
  {/if}
  {#if node.item && !node.isDir}
    {@const entry = node.item}
    <button
      type="button"
      class="switch"
      role="switch"
      aria-checked={entry.processed}
      aria-label={t('ft.contentToggle.switch', { name: displayName(node) })}
      title={t('ft.contentToggle.column')}
      disabled={pending.has(entry.rel)}
      onclick={() => toggle(entry)}
    >
      <span class="knob"></span>
    </button>
  {/if}
{/snippet}

<section class="ft-toggle-tree" aria-busy={loading}>
  <header>
    <p class="hint ft-muted">{t('ft.contentToggle.hint')}</p>
    <span class="column">{t('ft.contentToggle.column')}</span>
  </header>
  {#if error}
    <p class="ft-error-text" role="alert">{error}</p>
  {:else if entries === null}
    <p class="ft-muted status"><LoaderCircle size={14} aria-hidden="true" />{t('ft.common.loading')}</p>
  {:else if entries.length === 0}
    <p class="ft-muted status">{t('ft.contentToggle.empty')}</p>
  {:else}
    <TreeView {nodes} {row} label={t('ft.contentToggle.label')} selectable={() => false} />
  {/if}
</section>

<style>
  .ft-toggle-tree {
    display: flex;
    flex-direction: column;
    gap: 8px;
    font-size: 13px;
    color: var(--ft-text);
  }

  header {
    display: flex;
    align-items: flex-end;
    gap: 12px;
  }

  p {
    margin: 0;
  }

  .hint {
    flex: 1;
    font-size: 12px;
  }

  .column {
    color: var(--ft-text-secondary);
    font-size: 12px;
    white-space: nowrap;
  }

  .status {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 12px;
  }

  .icon {
    display: inline-flex;
    color: var(--ft-text-muted);
  }

  .icon.dir {
    color: var(--ft-accent);
  }

  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .badge {
    padding: 0 5px;
    border-radius: 4px;
    background: var(--ft-subtle);
    color: var(--ft-template);
    font-size: 11px;
  }

  .switch {
    position: relative;
    flex: none;
    width: 30px;
    height: 17px;
    margin-left: auto;
    padding: 0;
    border: 1px solid var(--ft-border);
    border-radius: 999px;
    background: var(--ft-elevated);
    cursor: pointer;
    transition: background var(--ft-duration), border-color var(--ft-duration);
  }

  .switch[aria-checked='true'] {
    border-color: var(--ft-template);
    background: var(--ft-template);
  }

  .switch:disabled {
    opacity: 0.5;
    cursor: progress;
  }

  .switch:focus-visible {
    outline: 2px solid var(--ft-accent);
    outline-offset: 2px;
  }

  .knob {
    position: absolute;
    top: 2px;
    left: 2px;
    width: 11px;
    height: 11px;
    border-radius: 50%;
    background: var(--ft-text-secondary);
    transition: transform var(--ft-duration), background var(--ft-duration);
  }

  .switch[aria-checked='true'] .knob {
    transform: translateX(13px);
    background: var(--ft-on-accent);
  }
</style>
