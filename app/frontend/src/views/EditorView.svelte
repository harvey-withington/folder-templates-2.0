<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import { FolderOpen, FolderTree, Save } from 'lucide-svelte'
  import {
    blankDescriptor,
    ContentToggleTree,
    debounce,
    getFolderTemplatesContext,
    RegexTester,
    sameDescriptor,
    TemplateEditor,
    TokenScanPanel,
    type Inspection,
    type Parameter,
    type TemplateDescriptor,
    type ValidationIssue,
  } from '@harvey-withington/folder-templates-ui'
  import { app, errorText, wailsRuntime } from '../lib/bridge'
  import { t } from '../lib/i18n'
  import { nav } from '../lib/nav.svelte'
  import { confirmer, toasts } from '../lib/ui.svelte'
  import TryItPanel from './TryItPanel.svelte'

  interface Props {
    dir: string
  }

  let { dir }: Props = $props()

  type Tab = 'scan' | 'files' | 'regex' | 'try'
  const tabs: Tab[] = ['scan', 'files', 'regex', 'try']

  const { backend } = getFolderTemplatesContext()

  let insp = $state<Inspection | null>(null)
  let loadError = $state<string | null>(null)
  let descriptor = $state<TemplateDescriptor>(blankDescriptor())
  let saved = $state<TemplateDescriptor | null>(null)
  let issues = $state<ValidationIssue[]>([])
  let saving = $state(false)
  let tab = $state<Tab>('scan')
  let pattern = $state('')
  let replacement = $state('')
  let treeVersion = $state(0)

  const isNew = $derived(!!insp && !insp.isTemplate)
  const dirty = $derived(isNew || (saved !== null && !sameDescriptor(descriptor, saved)))

  const validate = debounce(async (d: TemplateDescriptor) => {
    try {
      issues = await backend.validate(d)
    } catch (err) {
      issues = [{ param: '', message: errorText(err) }]
    }
  }, 300)

  $effect(() => {
    if (insp) validate($state.snapshot(descriptor))
  })

  onMount(async () => {
    try {
      const i = await app().Inspect(dir)
      if (i.loadError) {
        loadError = i.loadError
        return
      }
      insp = i
      const d = i.template ?? { ...blankDescriptor(i.folderName), name: i.folderName }
      descriptor = structuredClone(d)
      saved = structuredClone(d)
      wailsRuntime()?.WindowSetTitle(`${d.name || i.folderName} — ${t('app.name')}`)
    } catch (err) {
      loadError = errorText(err)
    }
    nav.setGuard(async () => !dirty || (await confirmer.ask(t('app.editor.discard'), { confirmLabel: t('app.editor.discardConfirm'), danger: true })))
  })

  onDestroy(() => {
    validate.cancel()
    nav.setGuard(null)
    wailsRuntime()?.WindowSetTitle(t('app.name'))
  })

  async function save() {
    if (!insp || saving) return
    saving = true
    try {
      const snapshot = $state.snapshot(descriptor)
      await backend.save(insp.dir, snapshot)
      saved = structuredClone(snapshot)
      insp = { ...insp, isTemplate: true, template: snapshot }
      toasts.show(t('app.editor.saved'), 'success')
    } catch (err) {
      toasts.show(errorText(err), 'error')
    } finally {
      saving = false
    }
  }

  function addParameter(p: Parameter) {
    descriptor = { ...$state.snapshot(descriptor), parameters: [...$state.snapshot(descriptor).parameters, p] }
    toasts.show(t('app.editor.paramAdded', { name: p.name }), 'success')
  }

  function testMatch(p: string) {
    pattern = p
    tab = 'regex'
  }

  function onKeydown(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
      e.preventDefault()
      void save()
    }
  }
</script>

<svelte:window onkeydown={onKeydown} />

<section class="view">
  {#if loadError}
    <div class="failed">
      <h1>{t('app.editor.cantOpen')}</h1>
      <p class="ft-error-text selectable">{loadError}</p>
      <button class="ft-btn" onclick={() => nav.go({ view: 'library' })}>{t('app.common.back')}</button>
    </div>
  {:else if insp}
    <header>
      <div class="heading">
        <h1>{descriptor.name || insp.folderName}{#if dirty}<span class="dot" title={t('app.editor.unsaved')}>•</span>{/if}</h1>
        <p class="path selectable" title={insp.dir}>{insp.dir}</p>
      </div>
      <div class="tools">
        <button class="ft-btn ft-btn-icon" title={t('app.library.reveal')} aria-label={t('app.library.reveal')} onclick={() => insp && app().RevealPath(insp.dir)}>
          <FolderOpen size={16} />
        </button>
        <button class="ft-btn" disabled={isNew || dirty} title={dirty ? t('app.editor.saveFirst') : ''} onclick={() => nav.go({ view: 'generate', dir: insp?.dir })}>
          <FolderTree size={15} /> {t('app.editor.generate')}
        </button>
        <button class="ft-btn ft-btn-primary" disabled={!dirty || saving} onclick={save}>
          <Save size={15} /> {isNew ? t('app.editor.create') : t('app.editor.save')}
        </button>
      </div>
    </header>

    {#if isNew}
      <p class="new-banner">{t('app.editor.newBanner')}</p>
    {/if}

    <div class="columns">
      <div class="editor-panel">
        <TemplateEditor bind:descriptor {issues} folderName={insp.folderName} ontestmatch={testMatch} />
      </div>
      <aside class="tools-panel">
        <div class="tabs" role="tablist" aria-label={t('app.editor.tools')}>
          {#each tabs as id (id)}
            <button role="tab" aria-selected={tab === id} class:on={tab === id} onclick={() => (tab = id)}>
              {t(`app.editor.tabs.${id}`)}
            </button>
          {/each}
        </div>
        <div class="tab-body" role="tabpanel">
          {#if tab === 'scan'}
            <TokenScanPanel dir={insp.dir} descriptor={descriptor} onaddparameter={addParameter} />
          {:else if tab === 'files'}
            {#key treeVersion}
              <ContentToggleTree dir={insp.dir} onchanged={() => treeVersion++} />
            {/key}
          {:else if tab === 'regex'}
            <RegexTester dir={insp.dir} bind:pattern bind:replacement />
          {:else}
            <TryItPanel dir={insp.dir} {descriptor} defaultTarget={insp.defaultTarget ?? ''} />
          {/if}
        </div>
      </aside>
    </div>
  {/if}
</section>

<style>
  .view {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    padding: 20px 28px 14px;
    border-bottom: 1px solid var(--border-muted);
  }

  .heading {
    min-width: 0;
  }

  .dot {
    margin-left: 6px;
    color: var(--accent);
  }

  .path {
    margin: 4px 0 0;
    overflow: hidden;
    color: var(--text-muted);
    font-size: 12px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .tools {
    display: flex;
    gap: 6px;
  }

  .new-banner {
    margin: 0;
    padding: 10px 28px;
    border-bottom: 1px solid var(--border-muted);
    background: var(--accent-glow-3);
    color: var(--text-primary);
  }

  .columns {
    flex: 1;
    display: grid;
    grid-template-columns: minmax(380px, 1fr) minmax(320px, 440px);
    min-height: 0;
  }

  .editor-panel {
    min-height: 0;
    padding: 18px 24px 28px 28px;
    overflow-y: auto;
  }

  .tools-panel {
    display: flex;
    flex-direction: column;
    min-height: 0;
    border-left: 1px solid var(--border-muted);
    background: var(--bg-base);
  }

  .tabs {
    display: flex;
    gap: 2px;
    padding: 10px 12px 0;
    border-bottom: 1px solid var(--border-muted);
  }

  .tabs button {
    padding: 7px 12px;
    border: none;
    border-bottom: 2px solid transparent;
    background: transparent;
    color: var(--text-secondary);
    font: inherit;
    cursor: pointer;
  }

  .tabs button:hover {
    color: var(--text-primary);
  }

  .tabs button.on {
    border-bottom-color: var(--accent);
    color: var(--text-primary);
  }

  .tab-body {
    flex: 1;
    min-height: 0;
    padding: 14px 16px 20px;
    overflow-y: auto;
  }

  .failed {
    max-width: 520px;
    margin: 60px auto;
  }
</style>
