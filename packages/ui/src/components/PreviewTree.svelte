<script lang="ts">
  import { FileText, Folder, LoaderCircle, TriangleAlert } from 'lucide-svelte'
  import type { PreviewEntry, PreviewResult } from '../types'
  import { baseName, buildTree, stripFtSuffix, type TreeNode } from '../lib/tree'
  import { plural, useUiHost } from '../lib/host'
  import TreeView from './TreeView.svelte'

  interface Props {
    result: PreviewResult | null
    loading?: boolean
    error?: string | null
    /** sourceRel of the selected filled-in file. */
    selected?: string
    onselect?: (entry: PreviewEntry) => void
  }

  let { result, loading = false, error = null, selected = $bindable(), onselect }: Props = $props()

  const { t } = useUiHost()

  const nodes = $derived(
    result
      ? buildTree(
          result.entries,
          (e) => e.outputRel,
          (e) => e.isDir,
        )
      : [],
  )

  const counts = $derived.by(() => {
    const entries = result?.entries ?? []
    return {
      folders: entries.filter((e) => e.isDir).length,
      files: entries.filter((e) => !e.isDir).length,
      existing: entries.filter((e) => e.exists).length,
    }
  })

  const treeSelected = $derived(result?.entries.find((e) => e.processed && e.sourceRel === selected)?.outputRel)

  /** The template-side name to show when the output was renamed; undefined when not renamed. */
  function renamedFrom(node: TreeNode<PreviewEntry>): string | undefined {
    const e = node.item
    if (!e || e.sourceRel === '') return undefined
    const source = baseName(e.sourceRel)
    return (e.processed ? stripFtSuffix(source) : source) === node.name ? undefined : source
  }

  function select(node: TreeNode<PreviewEntry>) {
    if (!node.item?.processed) return
    selected = node.item.sourceRel
    onselect?.(node.item)
  }
</script>

{#snippet row(node: TreeNode<PreviewEntry>)}
  {@const from = renamedFrom(node)}
  <span class="icon" class:dir={node.isDir}>
    {#if node.isDir}<Folder size={15} />{:else}<FileText size={15} />{/if}
  </span>
  <span class="name" class:renamed={from !== undefined} title={from ? t('ft.preview.renamedFrom', { name: from }) : undefined}>
    {node.name}
  </span>
  {#if from}
    <span class="from ft-muted">{t('ft.preview.from', { name: from })}</span>
  {/if}
  {#if node.item?.processed}
    <span class="badge" title={t('ft.preview.filledInTitle')}>{t('ft.preview.filledIn')}</span>
  {/if}
  {#if node.item?.exists}
    <span class="exists" title={t('ft.preview.exists')}>
      <TriangleAlert size={14} aria-hidden="true" />
      <span class="sr-only">{t('ft.preview.exists')}</span>
    </span>
  {/if}
{/snippet}

<section class="ft-preview" aria-busy={loading}>
  {#if result}
    <header>
      <span>{plural(t, 'ft.preview.folders', counts.folders)}</span>
      <span class="dot" aria-hidden="true">·</span>
      <span>{plural(t, 'ft.preview.files', counts.files)}</span>
      {#if counts.existing > 0}
        <span class="dot" aria-hidden="true">·</span>
        <span class="warn"><TriangleAlert size={14} aria-hidden="true" />{t('ft.preview.existing', { count: counts.existing })}</span>
      {/if}
      {#if loading}
        <span class="busy ft-muted"><span class="spin"><LoaderCircle size={14} aria-hidden="true" /></span>{t('ft.preview.updating')}</span>
      {/if}
    </header>
  {/if}

  {#if error}
    <p class="message ft-error-text" role="alert">{error}</p>
  {:else if result}
    <div class="tree" class:stale={loading}>
      <TreeView
        {nodes}
        {row}
        label={t('ft.preview.label')}
        selected={treeSelected}
        selectable={(n) => !!n.item?.processed}
        onselect={select}
      />
    </div>
    {#if result.warnings.length}
      <div class="warnings" role="note">
        <h4>{t('ft.preview.warnings')}</h4>
        <ul>
          {#each result.warnings as w}
            <li><TriangleAlert size={14} aria-hidden="true" />{w}</li>
          {/each}
        </ul>
      </div>
    {/if}
  {:else if loading}
    <p class="message ft-muted"><span class="spin"><LoaderCircle size={14} aria-hidden="true" /></span>{t('ft.preview.updating')}</p>
  {:else}
    <p class="message ft-muted">{t('ft.preview.empty')}</p>
  {/if}
</section>

<style>
  .ft-preview {
    display: flex;
    flex-direction: column;
    gap: 8px;
    font-size: 13px;
    color: var(--ft-text);
  }

  header {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 6px;
    color: var(--ft-text-secondary);
  }

  .dot {
    color: var(--ft-text-muted);
  }

  .warn,
  .exists {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    color: var(--ft-warning);
  }

  .busy {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    margin-left: auto;
  }

  .spin {
    display: inline-flex;
    animation: ft-spin 1s linear infinite;
  }

  @keyframes ft-spin {
    to {
      transform: rotate(360deg);
    }
  }

  .tree {
    transition: opacity var(--ft-duration);
  }

  .tree.stale {
    opacity: 0.6;
  }

  .icon {
    display: inline-flex;
    color: var(--ft-text-muted);
  }

  .icon.dir {
    color: var(--ft-accent);
  }

  /* Long names truncate; icons, badges and markers keep their size. The name
     and the secondary "from …" text shrink together, each with an ellipsis;
     "from" keeps enough width never to collapse to a fragment. */
  .icon,
  .badge,
  .exists {
    flex: none;
  }

  .name {
    flex: 0 1 auto;
    min-width: 3em;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .name.renamed {
    color: var(--ft-accent);
    font-weight: 500;
  }

  .from {
    flex: 0 1 auto;
    min-width: 7em;
    font-size: 12px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .badge {
    white-space: nowrap;
    padding: 0 6px;
    border: 1px solid var(--ft-template);
    border-radius: 999px;
    color: var(--ft-template);
    font-size: 11px;
    line-height: 16px;
  }

  .message {
    display: flex;
    align-items: center;
    gap: 6px;
    margin: 0;
    padding: 12px;
  }

  .warnings {
    padding: 8px 12px;
    border-radius: var(--ft-radius);
    background: var(--ft-warning-bg);
    color: var(--ft-warning-text);
  }

  .warnings h4 {
    margin: 0 0 4px;
    font-size: 12px;
    font-weight: 600;
  }

  .warnings ul {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .warnings li {
    display: flex;
    gap: 6px;
    align-items: baseline;
  }

  .sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
    white-space: nowrap;
  }
</style>
