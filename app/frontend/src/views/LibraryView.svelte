<script lang="ts">
  import { onMount } from 'svelte'
  import { FolderPlus, FolderSearch, RefreshCw, Search, X } from 'lucide-svelte'
  import { app, errorText, type LibraryEntry, type LibrarySource } from '../lib/bridge'
  import { t } from '../lib/i18n'
  import { nav } from '../lib/nav.svelte'
  import { toasts } from '../lib/ui.svelte'
  import { matchesQuery } from '../lib/format'
  import TemplateCard from './TemplateCard.svelte'

  interface Props {
    openFolder: (dir: string, target?: string) => Promise<void>
  }

  let { openFolder }: Props = $props()

  let entries = $state<LibraryEntry[]>([])
  let loaded = $state(false)
  let query = $state('')
  let searchInput = $state<HTMLInputElement | null>(null)

  const sections: { source: LibrarySource; title: string }[] = [
    { source: 'pinned', title: t('app.library.sections.pinned') },
    { source: 'recent', title: t('app.library.sections.recent') },
    { source: 'library', title: t('app.library.sections.library') },
    { source: 'sample', title: t('app.library.sections.samples') },
  ]

  const visible = $derived(
    entries.filter((e) => matchesQuery(query, e.name, e.description, e.folderName, e.path)),
  )

  onMount(() => {
    void load()
    searchInput?.focus()
  })

  async function load() {
    try {
      entries = await app().ListLibrary()
    } catch (err) {
      toasts.show(errorText(err), 'error')
    } finally {
      loaded = true
    }
  }

  function open(entry: LibraryEntry) {
    if (nav.pickTarget) void nav.go({ view: 'generate', dir: entry.path, target: nav.pickTarget })
    else void nav.go({ view: 'generate', dir: entry.path, target: null })
  }

  function edit(entry: LibraryEntry) {
    void nav.go({ view: 'editor', dir: entry.path })
  }

  async function pin(entry: LibraryEntry, pinned: boolean) {
    await guarded(() => app().SetPinned(entry.path, pinned))
    await load()
  }

  async function forget(entry: LibraryEntry) {
    await guarded(() => app().RemoveRecent(entry.path))
    await load()
  }

  function reveal(entry: LibraryEntry) {
    void guarded(() => app().RevealPath(entry.path))
  }

  async function openOther() {
    const dir = await app().PickFolder(t('app.library.pickTemplate'), '')
    if (dir) await openFolder(dir, nav.pickTarget ?? undefined)
  }

  async function addLibraryFolder() {
    const dir = await app().PickFolder(t('app.library.pickLibrary'), '')
    if (!dir) return
    await guarded(async () => {
      const s = await app().GetSettings()
      await app().SaveSettings({ ...s, libraryFolders: [...s.libraryFolders, dir] })
      toasts.show(t('app.library.folderAdded', { dir }), 'success')
    })
    await load()
  }

  async function guarded(fn: () => Promise<unknown>) {
    try {
      await fn()
    } catch (err) {
      toasts.show(errorText(err), 'error')
    }
  }

  function onKeydown(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'f') {
      e.preventDefault()
      searchInput?.focus()
    }
  }
</script>

<svelte:window onkeydown={onKeydown} />

<section class="view">
  <header>
    <div class="title-row">
      <h1>{t('app.library.title')}</h1>
      <div class="tools">
        <button class="ft-btn" onclick={openOther}><FolderSearch size={15} /> {t('app.library.openFolder')}</button>
        <button class="ft-btn" onclick={addLibraryFolder}><FolderPlus size={15} /> {t('app.library.addFolder')}</button>
        <button class="ft-btn ft-btn-icon" title={t('app.common.refresh')} aria-label={t('app.common.refresh')} onclick={load}>
          <RefreshCw size={15} />
        </button>
      </div>
    </div>
    {#if nav.pickTarget}
      <div class="pick-banner">
        <span>{t('app.library.pickBanner')} <strong class="selectable">{nav.pickTarget}</strong></span>
        <button class="ft-btn ft-btn-icon" aria-label={t('app.common.cancel')} title={t('app.common.cancel')} onclick={() => (nav.pickTarget = null)}>
          <X size={15} />
        </button>
      </div>
    {/if}
    <label class="search">
      <Search size={15} />
      <input bind:this={searchInput} bind:value={query} placeholder={t('app.library.search')} aria-label={t('app.library.search')} />
    </label>
  </header>

  <div class="scroll">
    {#if loaded && entries.length === 0}
      <div class="empty">
        <h2>{t('app.library.emptyTitle')}</h2>
        <p>{t('app.library.emptyBody')}</p>
        <div class="empty-actions">
          <button class="ft-btn ft-btn-primary" onclick={addLibraryFolder}><FolderPlus size={15} /> {t('app.library.addFolder')}</button>
          <button class="ft-btn" onclick={openOther}><FolderSearch size={15} /> {t('app.library.openFolder')}</button>
        </div>
      </div>
    {:else if loaded && visible.length === 0}
      <p class="ft-muted none">{t('app.library.noMatch', { query })}</p>
    {/if}

    {#each sections as section (section.source)}
      {@const items = visible.filter((e) => e.source === section.source)}
      {#if items.length > 0}
        <h2 class="section">{section.title}</h2>
        <div class="grid">
          {#each items as entry (entry.path)}
            <TemplateCard
              {entry}
              onopen={open}
              onedit={edit}
              onpin={pin}
              onreveal={reveal}
              onforget={entry.source === 'recent' ? forget : undefined}
            />
          {/each}
        </div>
      {/if}
    {/each}
  </div>
</section>

<style>
  .view {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  header {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 22px 28px 14px;
  }

  .title-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .tools {
    display: flex;
    gap: 6px;
  }

  .pick-banner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 8px 8px 8px 14px;
    border: 1px solid var(--template-accent);
    border-radius: 8px;
    background: var(--accent-glow-3);
    color: var(--text-primary);
  }

  .search {
    display: flex;
    align-items: center;
    gap: 8px;
    max-width: 420px;
    padding-left: 10px;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: var(--bg-elevated);
    color: var(--text-muted);
  }

  .search:focus-within {
    border-color: var(--accent);
  }

  .search input {
    border: none;
    background: transparent;
    box-shadow: none;
  }

  .scroll {
    flex: 1;
    overflow-y: auto;
    padding: 0 28px 28px;
  }

  .section {
    margin: 18px 0 10px;
    color: var(--text-secondary);
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(270px, 1fr));
    gap: 12px;
  }

  .empty {
    max-width: 460px;
    margin: 60px auto;
    text-align: center;
  }

  .empty p {
    color: var(--text-secondary);
  }

  .empty-actions {
    display: flex;
    justify-content: center;
    gap: 8px;
  }

  .none {
    margin-top: 30px;
    text-align: center;
  }
</style>
