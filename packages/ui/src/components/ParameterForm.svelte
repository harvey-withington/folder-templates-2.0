<script lang="ts">
  import { onMount } from 'svelte'
  import type { Parameter, Values } from '../types'
  import { initialValues, promptedParameters } from '../lib/params'
  import { useUiHost } from '../lib/host'

  interface Props {
    parameters: Parameter[]
    /** Answers keyed by parameter name; filled with defaults for every prompted parameter. */
    values?: Values
    disabled?: boolean
    /** Focus the first input on mount. */
    autofocus?: boolean
    /** Enter in any input. */
    onsubmit?: () => void
  }

  let { parameters, values = $bindable({}), disabled = false, autofocus = false, onsubmit }: Props = $props()

  const { t } = useUiHost()
  const uid = $props.id()
  let container: HTMLDivElement | undefined = $state()

  // First occurrence wins: a broken template with duplicate names must not crash the keyed list.
  const prompted = $derived.by(() => {
    const seen = new Set<string>()
    return promptedParameters(parameters).filter((p) => !seen.has(p.name) && !!seen.add(p.name))
  })

  function sameValues(a: Values, b: Values): boolean {
    const ak = Object.keys(a)
    return ak.length === Object.keys(b).length && ak.every((k) => b[k] === a[k])
  }

  // Keep one entry per prompted parameter; typed answers survive parameter changes.
  $effect.pre(() => {
    const next = initialValues(parameters, values)
    if (!sameValues(next, values)) values = next
  })

  onMount(() => {
    if (autofocus) container?.querySelector('input')?.focus()
  })

  function keydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.isComposing) {
      e.preventDefault()
      onsubmit?.()
    }
  }
</script>

<div class="ft-form" bind:this={container} role="group" aria-label={t('ft.form.label')}>
  {#each prompted as p, i (p.name)}
    <div class="field">
      <label for="{uid}-{i}">{p.prompt}</label>
      <input
        id="{uid}-{i}"
        type="text"
        bind:value={values[p.name]}
        placeholder={p.placeholder ?? ''}
        {disabled}
        autocomplete="off"
        spellcheck="false"
        onkeydown={keydown}
      />
    </div>
  {:else}
    <p class="empty ft-muted">{t('ft.form.empty')}</p>
  {/each}
</div>

<style>
  .ft-form {
    display: flex;
    flex-direction: column;
    gap: 12px;
    font-size: 13px;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 5px;
  }

  label {
    color: var(--ft-text-secondary);
    font-weight: 500;
  }

  input:disabled {
    opacity: 0.6;
  }

  .empty {
    margin: 0;
    padding: 14px;
    border: 1px dashed var(--ft-border-muted);
    border-radius: var(--ft-radius);
    text-align: center;
  }
</style>
