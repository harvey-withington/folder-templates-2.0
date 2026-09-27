<script lang="ts">
  import { onDestroy, onMount } from 'svelte'
  import { FolderOpen, Pencil, TriangleAlert } from 'lucide-svelte'
  import {
    initialValues,
    LivePreview,
    ParameterForm,
    previewValues,
    promptedParameters,
    type Inspection,
    type PreviewResult,
    type Values,
  } from '@harvey-withington/folder-templates-ui'
  import {
    app,
    errorText,
    wailsRuntime,
    type ConflictChoice,
    type GenerateProgress,
    type GenerateResult,
    type Settings,
  } from '../lib/bridge'
  import { t } from '../lib/i18n'
  import { nav } from '../lib/nav.svelte'
  import { confirmer, toasts } from '../lib/ui.svelte'
  import GenerateResultCard from './GenerateResultCard.svelte'

  interface Props {
    dir: string
    settings: Settings | null
  }

  let { dir, settings }: Props = $props()

  let insp = $state<Inspection | null>(null)
  let loadError = $state<string | null>(null)
  let values = $state<Values>({})
  let target = $state('')
  let conflict = $state<ConflictChoice>('refuse')
  let preview = $state<PreviewResult | null>(null)
  let previewError = $state<string | null>(null)
  let running = $state(false)
  let progress = $state<GenerateProgress | null>(null)
  let result = $state<GenerateResult | null>(null)

  const params = $derived(insp?.template?.parameters ?? [])
  const fixedTarget = $derived(nav.pickTarget)
  // Blank name answers preview as their token ("{name}") instead of an error.
  const shownValues = $derived(previewValues(params, values))
  // Until the name answers are in, the preview shows placeholders, so a
  // conflict isn't meaningful yet.
  const blocked = $derived(!!preview?.rootExists && conflict === 'refuse')
  // Answers that end up in file or folder names can't be blank; content-only
  // ones can (an empty "optional" field just fills in nothing).
  const missing = $derived(
    promptedParameters(params).find((p) => p.replaceInFileNames && !(values[p.name] ?? '').trim()),
  )
  const canGenerate = $derived(
    !!insp && !!preview && !previewError && !running && !blocked && !missing && target.trim() !== '',
  )

  let stopProgress: (() => void) | undefined

  onMount(async () => {
    conflict = settings?.defaultConflict ?? 'refuse'
    try {
      const i = await app().Inspect(dir)
      if (!i.isTemplate) {
        await nav.go({ view: 'editor', dir })
        return
      }
      insp = i
      values = initialValues(i.template?.parameters ?? [])
      target = fixedTarget ?? nav.target ?? i.defaultTarget ?? ''
      wailsRuntime()?.WindowSetTitle(`${i.template?.name || i.folderName} — ${t('app.name')}`)
      void app().RecordRecent(i.dir)
    } catch (err) {
      loadError = errorText(err)
    }
    stopProgress = wailsRuntime()?.EventsOn('generate:progress', (data) => {
      progress = data as GenerateProgress
    })
  })

  onDestroy(() => {
    stopProgress?.()
    wailsRuntime()?.WindowSetTitle(t('app.name'))
  })

  function onPreview(r: PreviewResult | null, error: string | null) {
    preview = r
    previewError = error
  }

  async function browseTarget() {
    const picked = await app().PickFolder(t('app.generate.pickTarget'), target)
    if (picked) target = picked
  }

  async function generate() {
    if (!canGenerate || !insp) return
    if (conflict === 'overwrite' && preview?.rootExists) {
      const ok = await confirmer.ask(t('app.generate.confirmOverwrite', { name: preview.rootName }), {
        confirmLabel: t('app.generate.overwrite'),
        danger: true,
      })
      if (!ok) return
    }
    running = true
    progress = null
    try {
      result = await app().Generate({ dir: insp.dir, target, values: $state.snapshot(values), conflict })
      toasts.show(t('app.generate.done', { name: preview?.rootName ?? '' }), 'success')
      if (settings?.openFolderAfter) void app().OpenFolder(result.rootPath)
      if (settings?.closeAfter) void app().Quit()
    } catch (err) {
      toasts.show(errorText(err), 'error')
    } finally {
      running = false
    }
  }

  function again() {
    result = null
    progress = null
  }

  function onKeydown(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
      e.preventDefault()
      void generate()
    }
  }
</script>

<svelte:window onkeydown={onKeydown} />

<section class="view">
  {#if loadError}
    <div class="failed">
      <h1>{t('app.generate.cantOpen')}</h1>
      <p class="ft-error-text selectable">{loadError}</p>
      <button class="ft-btn" onclick={() => nav.go({ view: 'library' })}>{t('app.common.back')}</button>
    </div>
  {:else if insp}
    <header>
      <div class="heading">
        <h1>{insp.template?.name || insp.folderName}</h1>
        {#if insp.template?.description}
          <p class="desc">{insp.template.description}</p>
        {/if}
        <p class="path selectable" title={insp.dir}>{insp.dir}</p>
      </div>
      <div class="tools">
        <button class="ft-btn" onclick={() => nav.go({ view: 'editor', dir: insp?.dir })}><Pencil size={15} /> {t('app.generate.edit')}</button>
        <button class="ft-btn ft-btn-icon" title={t('app.library.reveal')} aria-label={t('app.library.reveal')} onclick={() => insp && app().RevealPath(insp.dir)}>
          <FolderOpen size={16} />
        </button>
      </div>
    </header>

    <div class="columns">
      <div class="form-panel">
        {#if result}
          <GenerateResultCard {result} onagain={again} />
        {:else}
          <h2>{t('app.generate.details')}</h2>
          <ParameterForm parameters={params} bind:values disabled={running} autofocus onsubmit={generate} />

          <h2>{t('app.generate.destination')}</h2>
          <div class="target-row">
            <input
              aria-label={t('app.generate.destination')}
              bind:value={target}
              disabled={running || !!fixedTarget}
              list="recent-targets"
              spellcheck="false"
            />
            <button class="ft-btn" disabled={running || !!fixedTarget} onclick={browseTarget}>{t('app.common.browse')}</button>
          </div>
          <datalist id="recent-targets">
            {#each settings?.recentTargets ?? [] as recent (recent)}
              <option value={recent}></option>
            {/each}
          </datalist>

          {#if preview?.rootExists && !missing}
            <div class="conflict" role="alert">
              <TriangleAlert size={16} />
              <div>
                <p>{t('app.generate.exists', { name: preview.rootName })}</p>
                <select bind:value={conflict} aria-label={t('app.settings.defaultConflict')} disabled={running}>
                  <option value="refuse">{t('app.conflict.refuse')}</option>
                  <option value="merge">{t('app.conflict.merge')}</option>
                  <option value="overwrite">{t('app.conflict.overwrite')}</option>
                </select>
              </div>
            </div>
          {/if}

          <div class="actions">
            {#if running}
              <div class="progress" role="progressbar" aria-valuemin="0" aria-valuemax={progress?.total ?? 0} aria-valuenow={progress?.done ?? 0}>
                <div class="bar" style:width={progress && progress.total ? `${(progress.done / progress.total) * 100}%` : '8%'}></div>
              </div>
              <button class="ft-btn" onclick={() => app().CancelGenerate()}>{t('app.common.cancel')}</button>
            {:else}
              <button class="ft-btn ft-btn-primary create" disabled={!canGenerate} onclick={generate}>
                {preview ? t('app.generate.create', { name: preview.rootName }) : t('app.generate.createPlain')}
              </button>
              {#if nav.launchedFor}
                <button class="ft-btn" onclick={() => app().Quit()}>{t('app.common.close')}</button>
              {/if}
            {/if}
          </div>
          {#if missing && !running}
            <p class="shortcut ft-muted">{t('app.generate.missing', { field: missing.prompt ?? missing.name })}</p>
          {:else}
            <p class="shortcut ft-muted">{t('app.generate.shortcut')}</p>
          {/if}
        {/if}
      </div>

      <div class="preview-panel">
        <h2>{t('app.generate.preview')}</h2>
        <LivePreview dir={insp.dir} values={shownValues} {target} onresult={onPreview} />
      </div>
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

  .desc {
    margin: 4px 0 0;
    color: var(--text-secondary);
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

  .columns {
    flex: 1;
    display: grid;
    grid-template-columns: minmax(300px, 400px) 1fr;
    min-height: 0;
  }

  .form-panel,
  .preview-panel {
    display: flex;
    flex-direction: column;
    gap: 12px;
    min-height: 0;
    padding: 18px 24px 24px 28px;
    overflow-y: auto;
  }

  .preview-panel {
    padding-left: 24px;
    border-left: 1px solid var(--border-muted);
    background: var(--bg-base);
  }

  h2 {
    color: var(--text-secondary);
    font-size: 12px;
    letter-spacing: 0.04em;
    text-transform: uppercase;
  }

  .target-row {
    display: flex;
    gap: 6px;
  }

  .conflict {
    display: flex;
    gap: 10px;
    padding: 10px 12px;
    border: 1px solid var(--warning);
    border-radius: 8px;
    background: var(--warning-bg);
    color: var(--warning-text);
  }

  .conflict p {
    margin: 0 0 8px;
  }

  .actions {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 6px;
  }

  .create {
    flex: 1;
    justify-content: center;
    padding: 9px 14px;
    font-weight: 600;
  }

  .progress {
    flex: 1;
    height: 8px;
    overflow: hidden;
    border-radius: 999px;
    background: var(--bg-subtle-hover);
  }

  .bar {
    height: 100%;
    background: var(--accent);
    transition: width 0.12s linear;
  }

  .shortcut {
    margin: 0;
    font-size: 12px;
  }

  .failed {
    max-width: 520px;
    margin: 60px auto;
  }
</style>
