<script lang="ts">
  import { onMount } from 'svelte'
  import { ArrowDown, ArrowUp, FolderPlus, Trash2 } from 'lucide-svelte'
  import {
    app,
    errorText,
    type ConflictChoice,
    type Settings,
    type ShellFeature,
    type ShellStatus,
    type Theme,
  } from '../lib/bridge'
  import { t } from '../lib/i18n'
  import { toasts } from '../lib/ui.svelte'

  interface Props {
    settings: Settings
    onchange: (next: Settings) => void
  }

  let { settings, onchange }: Props = $props()

  let shell = $state<ShellStatus | null>(null)
  let shellBusy = $state<ShellFeature | null>(null)
  let version = $state('')

  const themes: Theme[] = ['system', 'dark', 'light']
  const conflicts: ConflictChoice[] = ['refuse', 'merge', 'overwrite']
  const shellFeatures: ShellFeature[] = ['sendToProcess', 'sendToEdit', 'folderMenu', 'backgroundMenu']

  onMount(async () => {
    try {
      version = await app().Version()
      shell = await app().ShellStatus()
    } catch (err) {
      toasts.show(errorText(err), 'error')
    }
  })

  async function save(patch: Partial<Settings>) {
    try {
      onchange(await app().SaveSettings({ ...settings, ...patch }))
    } catch (err) {
      toasts.show(errorText(err), 'error')
    }
  }

  async function addFolder() {
    const dir = await app().PickFolder(t('app.library.pickLibrary'), '')
    if (dir) await save({ libraryFolders: [...settings.libraryFolders, dir] })
  }

  function moveFolder(from: number, to: number) {
    const list = settings.libraryFolders.slice()
    const [item] = list.splice(from, 1)
    list.splice(to, 0, item)
    void save({ libraryFolders: list })
  }

  async function toggleShell(feature: ShellFeature, on: boolean) {
    shellBusy = feature
    try {
      shell = await app().SetShellFeature(feature, on)
    } catch (err) {
      toasts.show(errorText(err), 'error')
    } finally {
      shellBusy = null
    }
  }
</script>

<section class="view">
  <div class="scroll">
    <h1>{t('app.settings.title')}</h1>

    <div class="group">
      <h2>{t('app.settings.appearance')}</h2>
      <div class="segmented" role="radiogroup" aria-label={t('app.settings.theme')}>
        {#each themes as theme (theme)}
          <button role="radio" aria-checked={settings.theme === theme} class:on={settings.theme === theme} onclick={() => save({ theme })}>
            {t(`app.settings.themes.${theme}`)}
          </button>
        {/each}
      </div>
    </div>

    <div class="group">
      <h2>{t('app.settings.libraryFolders')}</h2>
      <p class="hint">{t('app.settings.libraryHint')}</p>
      {#if settings.libraryFolders.length === 0}
        <p class="ft-muted">{t('app.settings.noFolders')}</p>
      {:else}
        <ul class="folders">
          {#each settings.libraryFolders as folder, i (folder)}
            <li>
              <span class="path selectable" title={folder}>{folder}</span>
              <button class="ft-btn ft-btn-icon" disabled={i === 0} aria-label={t('app.common.moveUp')} title={t('app.common.moveUp')} onclick={() => moveFolder(i, i - 1)}><ArrowUp size={14} /></button>
              <button class="ft-btn ft-btn-icon" disabled={i === settings.libraryFolders.length - 1} aria-label={t('app.common.moveDown')} title={t('app.common.moveDown')} onclick={() => moveFolder(i, i + 1)}><ArrowDown size={14} /></button>
              <button class="ft-btn ft-btn-icon" aria-label={t('app.common.remove')} title={t('app.common.remove')} onclick={() => save({ libraryFolders: settings.libraryFolders.filter((f) => f !== folder) })}><Trash2 size={14} /></button>
            </li>
          {/each}
        </ul>
      {/if}
      <div class="row">
        <button class="ft-btn" onclick={addFolder}><FolderPlus size={15} /> {t('app.library.addFolder')}</button>
      </div>
      <label class="check">
        <input type="checkbox" checked={settings.showSamples} onchange={(e) => save({ showSamples: e.currentTarget.checked })} />
        {t('app.settings.showSamples')}
      </label>
    </div>

    <div class="group">
      <h2>{t('app.settings.generating')}</h2>
      <label class="field">
        <span>{t('app.settings.defaultConflict')}</span>
        <select value={settings.defaultConflict} onchange={(e) => save({ defaultConflict: e.currentTarget.value as ConflictChoice })}>
          {#each conflicts as c (c)}
            <option value={c}>{t(`app.conflict.${c}`)}</option>
          {/each}
        </select>
      </label>
      <label class="check">
        <input type="checkbox" checked={settings.openFolderAfter} onchange={(e) => save({ openFolderAfter: e.currentTarget.checked })} />
        {t('app.settings.openFolderAfter')}
      </label>
      <label class="check">
        <input type="checkbox" checked={settings.closeAfter} onchange={(e) => save({ closeAfter: e.currentTarget.checked })} />
        {t('app.settings.closeAfter')}
      </label>
    </div>

    <div class="group">
      <h2>{t('app.settings.explorer')}</h2>
      <p class="hint">{t('app.settings.explorerHint')}</p>
      {#each shellFeatures as feature (feature)}
        <label class="check">
          <input
            type="checkbox"
            checked={shell?.[feature] ?? false}
            disabled={shell === null || shellBusy !== null}
            onchange={(e) => toggleShell(feature, e.currentTarget.checked)}
          />
          <span>
            {t(`app.settings.shell.${feature}`)}
            <span class="sub">{t(`app.settings.shell.${feature}Hint`)}</span>
          </span>
        </label>
      {/each}
    </div>

    <div class="group">
      <h2>{t('app.settings.about')}</h2>
      <p class="ft-muted selectable">{t('app.settings.version', { version })}</p>
    </div>
  </div>
</section>

<style>
  .view {
    height: 100%;
  }

  .scroll {
    height: 100%;
    overflow-y: auto;
    padding: 22px 28px 40px;
  }

  .scroll > :global(*) {
    max-width: 680px;
  }

  .group {
    display: flex;
    flex-direction: column;
    gap: 10px;
    margin-top: 22px;
    padding: 16px 18px;
    border: 1px solid var(--border-muted);
    border-radius: 10px;
    background: var(--bg-elevated);
  }

  .hint {
    margin: -4px 0 0;
    color: var(--text-secondary);
  }

  .segmented {
    display: inline-flex;
    align-self: flex-start;
    padding: 3px;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: var(--bg-surface);
  }

  .segmented button {
    padding: 5px 14px;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: var(--text-secondary);
    font: inherit;
    cursor: pointer;
  }

  .segmented button.on {
    background: var(--accent);
    color: var(--on-color);
  }

  .folders {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .folders li {
    display: flex;
    align-items: center;
    gap: 2px;
    padding: 4px 4px 4px 10px;
    border: 1px solid var(--border-muted);
    border-radius: 6px;
  }

  .path {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .row {
    display: flex;
    gap: 8px;
  }

  .check {
    display: flex;
    align-items: flex-start;
    gap: 9px;
    color: var(--text-primary);
    cursor: pointer;
  }

  .check input {
    margin-top: 3px;
  }

  .sub {
    display: block;
    color: var(--text-muted);
    font-size: 12px;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 5px;
    max-width: 320px;
    color: var(--text-secondary);
  }
</style>
