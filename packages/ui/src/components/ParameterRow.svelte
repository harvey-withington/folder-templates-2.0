<script lang="ts">
  import { onMount } from 'svelte'
  import { ArrowDown, ArrowUp, ChevronDown, CircleAlert, GripVertical, Trash2 } from 'lucide-svelte'
  import type { ValidationIssue } from '../types'
  import { humanize, type ParameterRow } from '../lib/params'
  import { useUiHost } from '../lib/host'
  import ParameterDetails from './ParameterDetails.svelte'

  interface Props {
    row: ParameterRow
    index: number
    count: number
    issues?: ValidationIssue[]
    /** Focus the name input on mount (a freshly added row). */
    autofocus?: boolean
    onmoveup?: () => void
    onmovedown?: () => void
    ondelete?: () => void
    ontestmatch?: (pattern: string, name: string) => void
  }

  let {
    row = $bindable(),
    index,
    count,
    issues = [],
    autofocus = false,
    onmoveup,
    onmovedown,
    ondelete,
    ontestmatch,
  }: Props = $props()

  const { t } = useUiHost()
  const uid = $props.id()
  let nameInput: HTMLInputElement | undefined = $state()
  let expanded = $state(false)

  const p = $derived(row.param)
  const nameless = $derived(p.name.trim() === '')
  /** No name but a pattern: a rename rule, which is never asked and only renames. */
  const isRule = $derived(nameless && !!p.match)
  const cardLabel = $derived(nameless ? t('ft.param.cardUnnamed') : t('ft.param.card', { name: p.name }))

  onMount(() => {
    if (autofocus) nameInput?.focus()
  })

  function setInternal(internal: boolean) {
    if (internal === row.internal) return
    if (internal) {
      row.savedPrompt = row.param.prompt ?? ''
      row.param.prompt = null
    } else {
      row.param.prompt = row.savedPrompt
    }
    row.internal = internal
  }
</script>

<div class="card" class:invalid={issues.length > 0} role="group" aria-label={cardLabel}>
  <div class="main">
    <span class="grip" data-grip title={t('ft.param.drag')} aria-hidden="true"><GripVertical size={16} /></span>

    <div class="field name">
      <label for="{uid}-name">{t('ft.param.name')}</label>
      <input
        id="{uid}-name"
        type="text"
        class="ft-mono"
        bind:this={nameInput}
        bind:value={row.param.name}
        placeholder={t('ft.param.namePlaceholder')}
        spellcheck="false"
        autocomplete="off"
      />
    </div>

    {#if !isRule}
    <div class="field prompt">
      {#if row.internal}
        <span class="label-spacer" aria-hidden="true"></span>
        <p class="internal-note ft-muted">{t('ft.param.internalNote')}</p>
      {:else}
        <label for="{uid}-prompt">{t('ft.param.prompt')}</label>
        <input
          id="{uid}-prompt"
          type="text"
          bind:value={row.param.prompt}
          placeholder={nameless ? t('ft.param.promptPlaceholder') : humanize(p.name.trim())}
          autocomplete="off"
        />
      {/if}
    </div>

    <div class="controls">
      <button
        type="button"
        class="chip"
        aria-pressed={p.replaceInFileNames}
        title={t('ft.param.namesTitle')}
        onclick={() => (row.param.replaceInFileNames = !row.param.replaceInFileNames)}>{t('ft.param.names')}</button
      >
      <button
        type="button"
        class="chip"
        aria-pressed={p.replaceInFiles}
        title={t('ft.param.filesTitle')}
        onclick={() => (row.param.replaceInFiles = !row.param.replaceInFiles)}>{t('ft.param.files')}</button
      >
      <button
        type="button"
        class="chip mode"
        class:internal={row.internal}
        role="switch"
        aria-checked={!row.internal}
        aria-label={t('ft.param.askedLabel')}
        title={t('ft.param.askedLabel')}
        onclick={() => setInternal(!row.internal)}>{row.internal ? t('ft.param.internal') : t('ft.param.asked')}</button
      >
    </div>
    {/if}

    <div class="actions">
      <button type="button" class="ft-btn ft-btn-icon" aria-label={t('ft.param.moveUp')} title={t('ft.param.moveUp')} disabled={index === 0} onclick={onmoveup}>
        <ArrowUp size={14} />
      </button>
      <button type="button" class="ft-btn ft-btn-icon" aria-label={t('ft.param.moveDown')} title={t('ft.param.moveDown')} disabled={index >= count - 1} onclick={onmovedown}>
        <ArrowDown size={14} />
      </button>
      <button
        type="button"
        class="ft-btn ft-btn-icon more"
        class:open={expanded}
        aria-expanded={expanded}
        aria-controls="{uid}-more"
        aria-label={t('ft.param.more')}
        title={t('ft.param.more')}
        onclick={() => (expanded = !expanded)}
      >
        <ChevronDown size={14} />
      </button>
      <button type="button" class="ft-btn ft-btn-icon danger" aria-label={t('ft.param.delete')} title={t('ft.param.delete')} onclick={ondelete}>
        <Trash2 size={14} />
      </button>
    </div>
  </div>

  <ParameterDetails bind:row {expanded} id="{uid}-more" {ontestmatch} />

  {#if issues.length}
    <ul class="issues">
      {#each issues as issue}
        <li class="ft-error-text"><CircleAlert size={14} aria-hidden="true" />{issue.message}</li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .card {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 10px 10px 10px 6px;
    border: 1px solid var(--ft-border-muted);
    border-radius: var(--ft-radius);
    background: var(--ft-surface);
    font-size: 13px;
  }

  .card.invalid {
    border-color: var(--ft-danger);
  }

  .main {
    display: flex;
    flex-wrap: wrap;
    align-items: flex-end;
    gap: 8px;
  }

  .grip {
    align-self: center;
    display: inline-flex;
    color: var(--ft-text-muted);
    cursor: grab;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }

  .field.name {
    flex: 1 1 8rem;
  }

  .field.prompt {
    flex: 2 1 12rem;
  }

  label {
    color: var(--ft-text-secondary);
    font-size: 12px;
  }

  .label-spacer {
    height: 15px;
  }

  .internal-note {
    margin: 0;
    padding: 7px 0;
    font-style: italic;
  }

  .controls,
  .actions {
    display: flex;
    align-items: center;
    gap: 4px;
    padding-bottom: 3px;
  }

  .chip {
    padding: 4px 10px;
    border: 1px solid var(--ft-border);
    border-radius: 999px;
    background: transparent;
    color: var(--ft-text-muted);
    font: inherit;
    font-size: 12px;
    cursor: pointer;
    transition: background var(--ft-duration), color var(--ft-duration), border-color var(--ft-duration);
  }

  .chip[aria-pressed='true'],
  .chip.mode[aria-checked='true'] {
    border-color: var(--ft-accent);
    background: var(--ft-accent-soft);
    color: var(--ft-text-strong);
  }

  .chip.mode.internal {
    border-style: dashed;
  }

  .chip:focus-visible {
    outline: 2px solid var(--ft-accent);
    outline-offset: 1px;
  }

  .more :global(svg) {
    transition: transform var(--ft-duration);
  }

  .more.open :global(svg) {
    transform: rotate(180deg);
  }

  .danger:hover:not(:disabled) {
    color: var(--ft-danger);
  }

  .issues {
    margin: 0;
    padding: 0 0 0 24px;
    list-style: none;
  }

  .issues li {
    display: flex;
    align-items: center;
    gap: 6px;
  }
</style>
