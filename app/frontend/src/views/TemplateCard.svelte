<script lang="ts">
  import { FolderOpen, Pencil, Pin, PinOff, TriangleAlert, X } from 'lucide-svelte'
  import type { LibraryEntry } from '../lib/bridge'
  import { t } from '../lib/i18n'
  import { relativeTime } from '../lib/format'

  interface Props {
    entry: LibraryEntry
    onopen: (entry: LibraryEntry) => void
    onedit: (entry: LibraryEntry) => void
    onpin: (entry: LibraryEntry, pinned: boolean) => void
    onreveal: (entry: LibraryEntry) => void
    onforget?: (entry: LibraryEntry) => void
  }

  let { entry, onopen, onedit, onpin, onreveal, onforget }: Props = $props()

  const broken = $derived(!!entry.missing || !!entry.error)
</script>

<article class="card" class:broken>
  <button class="main" disabled={broken} onclick={() => onopen(entry)} title={entry.path}>
    <span class="name">{entry.name}</span>
    {#if entry.description}
      <span class="desc">{entry.description}</span>
    {/if}
    <span class="meta">
      {#if entry.missing}
        <span class="problem"><TriangleAlert size={12} /> {t('app.library.missing')}</span>
      {:else if entry.error}
        <span class="problem" title={entry.error}><TriangleAlert size={12} /> {t('app.library.unreadable')}</span>
      {:else}
        <span>{t('app.library.params', { n: entry.paramCount })}</span>
      {/if}
      {#if entry.lastUsed}
        <span>· {relativeTime(entry.lastUsed)}</span>
      {/if}
      {#if entry.source === 'sample'}
        <span class="badge">{t('app.library.sample')}</span>
      {/if}
    </span>
    <span class="path">{entry.path}</span>
  </button>
  <div class="actions">
    {#if !broken}
      <button class="ft-btn ft-btn-icon" title={t('app.library.edit')} aria-label={t('app.library.edit')} onclick={() => onedit(entry)}>
        <Pencil size={15} />
      </button>
      <button class="ft-btn ft-btn-icon" title={t('app.library.reveal')} aria-label={t('app.library.reveal')} onclick={() => onreveal(entry)}>
        <FolderOpen size={15} />
      </button>
    {/if}
    <button
      class="ft-btn ft-btn-icon"
      class:on={entry.pinned}
      title={entry.pinned ? t('app.library.unpin') : t('app.library.pin')}
      aria-label={entry.pinned ? t('app.library.unpin') : t('app.library.pin')}
      aria-pressed={entry.pinned}
      onclick={() => onpin(entry, !entry.pinned)}
    >
      {#if entry.pinned}<PinOff size={15} />{:else}<Pin size={15} />{/if}
    </button>
    {#if onforget}
      <button class="ft-btn ft-btn-icon" title={t('app.library.forget')} aria-label={t('app.library.forget')} onclick={() => onforget?.(entry)}>
        <X size={15} />
      </button>
    {/if}
  </div>
</article>

<style>
  .card {
    position: relative;
    display: flex;
    border: 1px solid var(--border-muted);
    border-radius: 10px;
    background: var(--bg-elevated);
    transition:
      border-color var(--duration-normal),
      transform var(--duration-normal);
  }

  .card:hover {
    border-color: var(--border-hover);
  }

  .card:has(.main:focus-visible) {
    border-color: var(--accent);
  }

  .main {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 14px 16px;
    border: none;
    background: transparent;
    color: inherit;
    font: inherit;
    text-align: left;
    cursor: pointer;
  }

  .main:focus-visible {
    outline: none;
  }

  .main:disabled {
    cursor: default;
  }

  .broken .name {
    color: var(--text-muted);
  }

  .name {
    color: var(--text-primary);
    font-size: 14px;
    font-weight: 600;
  }

  .desc {
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    color: var(--text-secondary);
  }

  .meta {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 6px;
    margin-top: 2px;
    color: var(--text-muted);
    font-size: 12px;
  }

  .problem {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    color: var(--warning);
  }

  .badge {
    padding: 0 6px;
    border: 1px solid var(--template-accent);
    border-radius: 999px;
    color: var(--template-accent);
    font-size: 11px;
  }

  .path {
    overflow: hidden;
    color: var(--text-muted);
    font-size: 11.5px;
    text-overflow: ellipsis;
    white-space: nowrap;
    direction: rtl;
    text-align: left;
  }

  .actions {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 8px 6px;
    opacity: 0;
    transition: opacity var(--duration-normal);
  }

  .card:hover .actions,
  .card:focus-within .actions,
  .actions :global(.on) {
    opacity: 1;
  }

  .actions :global(.on) {
    color: var(--accent);
  }
</style>
