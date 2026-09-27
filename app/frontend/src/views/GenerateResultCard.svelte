<script lang="ts">
  import { CircleCheck, Copy, FolderOpen, RotateCcw } from 'lucide-svelte'
  import { app, errorText, type GenerateResult } from '../lib/bridge'
  import { t } from '../lib/i18n'
  import { toasts } from '../lib/ui.svelte'

  interface Props {
    result: GenerateResult
    onagain: () => void
  }

  let { result, onagain }: Props = $props()

  async function copyPath() {
    try {
      await navigator.clipboard.writeText(result.rootPath)
      toasts.show(t('app.result.copied'), 'success')
    } catch (err) {
      toasts.show(errorText(err), 'error')
    }
  }
</script>

<div class="result">
  <div class="title">
    <CircleCheck size={22} />
    <h2>{t('app.result.title')}</h2>
  </div>
  <p class="path selectable">{result.rootPath}</p>
  <p class="ft-muted">{t('app.result.files', { n: result.filesWritten })}</p>

  {#if result.skipped?.length}
    <details>
      <summary>{t('app.result.skipped', { n: result.skipped.length })}</summary>
      <ul class="selectable">{#each result.skipped as s (s)}<li>{s}</li>{/each}</ul>
    </details>
  {/if}
  {#if result.overwritten?.length}
    <details>
      <summary>{t('app.result.overwritten', { n: result.overwritten.length })}</summary>
      <ul class="selectable">{#each result.overwritten as s (s)}<li>{s}</li>{/each}</ul>
    </details>
  {/if}
  {#if result.warnings?.length}
    <ul class="warnings selectable">{#each result.warnings as w (w)}<li>{w}</li>{/each}</ul>
  {/if}

  <div class="actions">
    <button class="ft-btn ft-btn-primary" onclick={() => app().OpenFolder(result.rootPath)}><FolderOpen size={15} /> {t('app.result.open')}</button>
    <button class="ft-btn" onclick={copyPath}><Copy size={15} /> {t('app.result.copy')}</button>
    <button class="ft-btn" onclick={onagain}><RotateCcw size={15} /> {t('app.result.again')}</button>
  </div>
</div>

<style>
  .result {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 18px;
    border: 1px solid var(--success);
    border-radius: 10px;
    background: var(--bg-elevated);
  }

  .title {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--success);
  }

  .title h2 {
    color: var(--text-primary);
    font-size: 16px;
    text-transform: none;
    letter-spacing: 0;
  }

  .path {
    margin: 0;
    color: var(--text-primary);
    overflow-wrap: anywhere;
  }

  p {
    margin: 0;
  }

  ul {
    margin: 6px 0 0;
    padding-left: 18px;
    font-size: 12.5px;
  }

  .warnings {
    color: var(--warning);
  }

  summary {
    cursor: pointer;
  }

  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-top: 8px;
  }
</style>
