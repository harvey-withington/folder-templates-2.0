<script lang="ts">
  import { FlaskConical } from 'lucide-svelte'
  import { defaultMatch, type ParameterRow } from '../lib/params'
  import { useUiHost } from '../lib/host'

  /** The secondary fields of a ParameterRow: the rename-rule block and the "More" section. */
  interface Props {
    row: ParameterRow
    /** Show the "More" section. */
    expanded: boolean
    /** Element id of the "More" section (the toggle's aria-controls). */
    id: string
    ontestmatch?: (pattern: string, name: string) => void
  }

  let { row = $bindable(), expanded, id, ontestmatch }: Props = $props()

  const { t } = useUiHost()
  const uid = $props.id()

  const name = $derived(row.param.name.trim())
  const nameless = $derived(name === '')
  /** No name but a pattern: a pure rename rule (e.g. strip "^_Template - "). */
  const isRule = $derived(nameless && !!row.param.match)

  function testMatch() {
    ontestmatch?.(row.param.match || defaultMatch(name), name)
  }
</script>

{#snippet matchField()}
  <div class="field wide">
    <label for="{uid}-match">{t('ft.param.match')}</label>
    <div class="with-button">
      <input
        id="{uid}-match"
        type="text"
        class="ft-mono"
        bind:value={row.param.match}
        placeholder={nameless ? t('ft.param.matchPlaceholderUnnamed') : defaultMatch(name)}
        spellcheck="false"
        autocomplete="off"
      />
      {#if ontestmatch}
        <button type="button" class="ft-btn" onclick={testMatch}>
          <FlaskConical size={14} aria-hidden="true" />{t('ft.param.testMatch')}
        </button>
      {/if}
    </div>
    <span class="hint ft-muted">{t('ft.param.matchHint')}</span>
  </div>
{/snippet}

{#if isRule}
  <div class="rule">
    <span class="rule-label">{t('ft.param.renameRule')}</span>
    <span class="hint ft-muted">{t('ft.param.renameRuleHint')}</span>
  </div>
  <div class="grid">
    {@render matchField()}
    <div class="field">
      <label for="{uid}-default">{t('ft.param.replaceWith')}</label>
      <input id="{uid}-default" type="text" bind:value={row.param.defaultValue} autocomplete="off" />
    </div>
  </div>
{/if}

{#if expanded}
  <div class="grid" {id}>
    <div class="field">
      <label for="{uid}-placeholder">{t('ft.param.placeholder')}</label>
      <input id="{uid}-placeholder" type="text" bind:value={row.param.placeholder} autocomplete="off" />
    </div>
    {#if !isRule}
      <div class="field">
        <label for="{uid}-default">{t('ft.param.defaultValue')}</label>
        <input id="{uid}-default" type="text" bind:value={row.param.defaultValue} autocomplete="off" />
      </div>
      {@render matchField()}
    {/if}
  </div>
{/if}

<style>
  .field {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }

  label {
    color: var(--ft-text-secondary);
    font-size: 12px;
  }

  .field.wide {
    grid-column: 1 / -1;
  }

  .rule {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px;
    padding-left: 24px;
  }

  .rule-label {
    padding: 1px 8px;
    border-radius: 999px;
    background: var(--ft-subtle);
    color: var(--ft-template);
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
    gap: 10px;
    padding-left: 24px;
  }

  .with-button {
    display: flex;
    gap: 6px;
  }

  .with-button .ft-btn {
    flex: none;
    white-space: nowrap;
  }

  .hint {
    font-size: 12px;
  }
</style>
