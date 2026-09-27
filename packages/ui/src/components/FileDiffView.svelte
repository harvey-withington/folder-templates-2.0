<script lang="ts">
  import { LoaderCircle, X } from 'lucide-svelte'
  import type { RenderedFile } from '../types'
  import { changedLines, diffLines } from '../lib/diff'
  import { plural, useUiHost } from '../lib/host'

  type Mode = 'result' | 'changes'

  interface Props {
    /** Shown in the header, usually the output path. */
    path: string
    rendered: RenderedFile | null
    loading?: boolean
    error?: string | null
    /** "result": the filled-in file; "changes": only changed lines, before and after. */
    mode?: Mode
    /** When set, a close button is shown in the header. */
    onclose?: () => void
  }

  let { path, rendered, loading = false, error = null, mode = $bindable('result'), onclose }: Props = $props()

  const { t } = useUiHost()

  const lines = $derived(rendered ? diffLines(rendered.before, rendered.after) : [])
  const changed = $derived(changedLines(lines))
  const modes: Mode[] = ['result', 'changes']
</script>

<section class="ft-diff" aria-busy={loading}>
  <header>
    <span class="path ft-mono" title={path}>{path}</span>
    {#if rendered}
      <span class="count ft-muted">{plural(t, 'ft.diff.changedCount', changed.length)}</span>
    {/if}
    {#if loading}
      <span class="spin" title={t('ft.diff.loading')}><LoaderCircle size={14} aria-hidden="true" /></span>
    {/if}
    <div class="modes" role="group" aria-label={t('ft.diff.mode')}>
      {#each modes as m (m)}
        <button type="button" aria-pressed={mode === m} onclick={() => (mode = m)}>{t(`ft.diff.${m}`)}</button>
      {/each}
    </div>
    {#if onclose}
      <button type="button" class="ft-btn ft-btn-icon" aria-label={t('ft.diff.close')} title={t('ft.diff.close')} onclick={onclose}>
        <X size={14} />
      </button>
    {/if}
  </header>

  {#if error}
    <p class="message ft-error-text" role="alert">{error}</p>
  {:else if !rendered}
    <p class="message ft-muted">{loading ? t('ft.diff.loading') : t('ft.diff.empty')}</p>
  {:else if mode === 'result'}
    {#if lines.length === 0}
      <p class="message ft-muted">{t('ft.diff.emptyFile')}</p>
    {:else}
      <div class="code ft-mono" class:stale={loading}>
        {#each lines as line (line.number)}
          <div class="line" class:add={line.changed}>
            <span class="ln">{line.number}</span><span class="text">{line.after}</span>
          </div>
        {/each}
      </div>
    {/if}
  {:else if changed.length === 0}
    <p class="message ft-muted">{t('ft.diff.noChanges')}</p>
  {:else}
    <div class="code ft-mono" class:stale={loading}>
      {#each changed as line (line.number)}
        <div class="line remove" title={t('ft.diff.before')}>
          <span class="ln">{line.number}</span><span class="sign" aria-hidden="true">−</span><span class="sr-only">{t('ft.diff.before')}</span><span class="text">{line.before}</span>
        </div>
        <div class="line add" title={t('ft.diff.after')}>
          <span class="ln"></span><span class="sign" aria-hidden="true">+</span><span class="sr-only">{t('ft.diff.after')}</span><span class="text">{line.after}</span>
        </div>
      {/each}
    </div>
  {/if}
</section>

<style>
  .ft-diff {
    display: flex;
    flex-direction: column;
    min-width: 0;
    border: 1px solid var(--ft-border-muted);
    border-radius: var(--ft-radius);
    background: var(--ft-surface);
    font-size: 13px;
    overflow: hidden;
  }

  header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 8px 6px 12px;
    border-bottom: 1px solid var(--ft-border-muted);
  }

  .path {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--ft-text-strong);
  }

  .count {
    font-size: 12px;
    white-space: nowrap;
  }

  .spin {
    display: inline-flex;
    color: var(--ft-text-muted);
    animation: ft-spin 1s linear infinite;
  }

  @keyframes ft-spin {
    to {
      transform: rotate(360deg);
    }
  }

  .modes {
    display: inline-flex;
    margin-left: auto;
    padding: 2px;
    border-radius: var(--ft-radius);
    background: var(--ft-subtle);
  }

  .modes button {
    padding: 3px 10px;
    border: none;
    border-radius: calc(var(--ft-radius) - 2px);
    background: transparent;
    color: var(--ft-text-secondary);
    font: inherit;
    font-size: 12px;
    cursor: pointer;
  }

  .modes button[aria-pressed='true'] {
    background: var(--ft-elevated);
    color: var(--ft-text-strong);
  }

  .modes button:focus-visible {
    outline: 2px solid var(--ft-accent);
    outline-offset: 1px;
  }

  .code {
    max-height: var(--ft-diff-max-height, 360px);
    overflow: auto;
    padding: 6px 0;
    line-height: 20px;
    transition: opacity var(--ft-duration);
  }

  .code.stale {
    opacity: 0.6;
  }

  .line {
    display: flex;
    min-width: max-content;
    white-space: pre;
  }

  .line.add {
    background: var(--ft-diff-add);
  }

  .line.remove {
    background: var(--ft-diff-remove);
  }

  .ln {
    flex: none;
    width: 3.5em;
    padding-right: 10px;
    text-align: right;
    color: var(--ft-text-muted);
    user-select: none;
  }

  .sign {
    flex: none;
    width: 1.2em;
    color: var(--ft-text-muted);
    user-select: none;
  }

  .text {
    padding-right: 12px;
    color: var(--ft-text);
  }

  .message {
    margin: 0;
    padding: 12px;
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
