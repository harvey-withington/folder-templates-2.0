<script lang="ts">
  import { onMount } from 'svelte'
  import { setFolderTemplatesContext } from '@harvey-withington/folder-templates-ui'
  import { app, errorText, wailsRuntime, type Settings } from './lib/bridge'
  import { wailsBackend } from './lib/backend'
  import { t } from './lib/i18n'
  import { nav } from './lib/nav.svelte'
  import { applyTheme, confirmer, toasts } from './lib/ui.svelte'
  import Rail from './components/Rail.svelte'
  import Toasts from './components/Toasts.svelte'
  import ConfirmDialog from './components/ConfirmDialog.svelte'
  import LibraryView from './views/LibraryView.svelte'
  import GenerateView from './views/GenerateView.svelte'
  import EditorView from './views/EditorView.svelte'
  import SettingsView from './views/SettingsView.svelte'

  setFolderTemplatesContext({
    backend: wailsBackend(),
    t,
    toast: (message, kind) => toasts.show(message, kind),
    confirm: (message, options) => confirmer.ask(message, options),
  })

  let settings = $state<Settings | null>(null)
  let ready = $state(false)
  let dropping = $state(false)

  onMount(() => {
    void start()
    const rt = wailsRuntime()
    rt?.OnFileDrop((_x, _y, paths) => {
      dropping = false
      if (paths.length > 0) void openFolder(paths[0])
    }, false)
    const onDragEnter = (e: DragEvent) => {
      if (e.dataTransfer?.types.includes('Files')) dropping = true
    }
    const onDragLeave = (e: DragEvent) => {
      if (e.relatedTarget === null) dropping = false
    }
    window.addEventListener('dragenter', onDragEnter)
    window.addEventListener('dragleave', onDragLeave)
    window.addEventListener('drop', () => (dropping = false))
    return () => {
      rt?.OnFileDropOff()
      window.removeEventListener('dragenter', onDragEnter)
      window.removeEventListener('dragleave', onDragLeave)
    }
  })

  async function start() {
    try {
      settings = await app().GetSettings()
      applyTheme(settings.theme)
      const intent = await app().GetLaunchIntent()
      if (intent.mode === 'pick') {
        nav.pickTarget = intent.target ?? null
      } else if (intent.source) {
        nav.launchedFor = intent.source
        nav.target = intent.target ?? null
        if (intent.mode === 'edit') await nav.go({ view: 'editor', dir: intent.source })
        else await openFolder(intent.source, intent.target)
      } else if (intent.target) {
        nav.target = intent.target
      }
    } catch (err) {
      toasts.show(errorText(err), 'error')
    } finally {
      ready = true
    }
  }

  /** 1.0's rule: a template opens to generate, any other folder to "make it a template". */
  async function openFolder(dir: string, target?: string) {
    try {
      const insp = await app().Inspect(dir)
      await nav.go({ view: insp.isTemplate ? 'generate' : 'editor', dir: insp.dir, target: target ?? nav.target })
    } catch (err) {
      toasts.show(errorText(err), 'error')
    }
  }

  function onSettingsChanged(next: Settings) {
    settings = next
    applyTheme(next.theme)
  }

  function onKeydown(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key === ',') {
      e.preventDefault()
      void nav.go({ view: 'settings' })
    }
  }
</script>

<svelte:window onkeydown={onKeydown} />

<div class="shell ft-scope">
  <Rail />
  <main class="content">
    {#if !ready}
      <div class="loading" aria-busy="true"></div>
    {:else if nav.view === 'library'}
      <LibraryView {openFolder} />
    {:else if nav.view === 'generate' && nav.dir}
      {#key nav.dir}
        <GenerateView dir={nav.dir} {settings} />
      {/key}
    {:else if nav.view === 'editor' && nav.dir}
      {#key nav.dir}
        <EditorView dir={nav.dir} />
      {/key}
    {:else if nav.view === 'settings' && settings}
      <SettingsView {settings} onchange={onSettingsChanged} />
    {:else}
      <LibraryView {openFolder} />
    {/if}
  </main>
  {#if dropping}
    <div class="drop-hint" aria-hidden="true">
      <div class="drop-card">{t('app.drop.hint')}</div>
    </div>
  {/if}
  <Toasts />
  <ConfirmDialog />
</div>

<style>
  .shell {
    display: flex;
    height: 100%;
    background: var(--bg-base);
  }

  .content {
    flex: 1;
    min-width: 0;
    height: 100%;
    overflow: hidden;
    background: var(--bg-surface);
    border-left: 1px solid var(--border-muted);
  }

  .loading {
    height: 100%;
  }

  .drop-hint {
    position: fixed;
    inset: 0;
    display: grid;
    place-items: center;
    background: var(--bg-overlay);
    pointer-events: none;
    z-index: 40;
  }

  .drop-card {
    padding: 28px 40px;
    border: 2px dashed var(--template-accent);
    border-radius: 14px;
    background: var(--bg-elevated);
    color: var(--text-primary);
    font-size: 15px;
  }
</style>
