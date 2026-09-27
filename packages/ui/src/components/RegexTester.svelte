<script lang="ts">
  import { onDestroy, untrack } from 'svelte'
  import { ArrowRight, FileText, Folder, LoaderCircle } from 'lucide-svelte'
  import type { NameHit } from '../types'
  import { getFolderTemplatesContext } from '../context'
  import { debounce, latestOnly } from '../lib/debounce'
  import { errorMessage } from '../lib/errors'
  import { plural } from '../lib/host'

  interface Props {
    /** Template folder. */
    dir: string
    /** .NET regular expression. */
    pattern?: string
    /** Value the match is replaced with; .NET syntax, so $1 inserts a group. */
    replacement?: string
    debounceMs?: number
  }

  let { dir, pattern = $bindable(''), replacement = $bindable(''), debounceMs = 250 }: Props = $props()

  const { backend, t } = getFolderTemplatesContext()
  const uid = $props.id()

  let hits = $state<NameHit[] | null>(null)
  let error = $state<string | null>(null)
  let loading = $state(false)

  const latest = latestOnly()
  async function run(forDir: string, forPattern: string, forReplacement: string) {
    const ticket = latest.next()
    loading = true
    try {
      const next = await backend.testMatch(forDir, forPattern, forReplacement)
      if (!latest.isLatest(ticket)) return
      hits = next
      error = null
    } catch (e) {
      if (!latest.isLatest(ticket)) return
      hits = null
      error = errorMessage(e)
    }
    loading = false
  }

  const schedule = debounce(run, untrack(() => debounceMs))
  onDestroy(() => schedule.cancel())

  $effect(() => {
    const [d, p, r] = [dir, pattern, replacement]
    untrack(() => {
      if (p === '') {
        schedule.cancel()
        latest.next() // drop anything in flight
        hits = null
        error = null
        loading = false
      } else {
        schedule(d, p, r)
      }
    })
  })
</script>

<section class="ft-regex" aria-busy={loading}>
  <div class="inputs">
    <div class="field">
      <label for="{uid}-pattern">{t('ft.regex.pattern')}</label>
      <input
        id="{uid}-pattern"
        type="text"
        class="ft-mono"
        bind:value={pattern}
        placeholder={t('ft.regex.patternPlaceholder')}
        spellcheck="false"
        autocomplete="off"
      />
    </div>
    <div class="field">
      <label for="{uid}-replacement">{t('ft.regex.replacement')}</label>
      <input
        id="{uid}-replacement"
        type="text"
        class="ft-mono"
        bind:value={replacement}
        placeholder={t('ft.regex.replacementPlaceholder')}
        aria-describedby="{uid}-replacement-hint"
        spellcheck="false"
        autocomplete="off"
      />
      <span id="{uid}-replacement-hint" class="hint ft-muted">{t('ft.regex.replacementHint')}</span>
    </div>
  </div>

  <div class="results" aria-live="polite">
    {#if pattern === ''}
      <p class="ft-muted">{t('ft.regex.idle')}</p>
    {:else if error}
      <div class="error" role="alert">
        <strong>{t('ft.regex.error')}</strong>
        <span class="ft-mono">{error}</span>
      </div>
    {:else if hits === null}
      <p class="ft-muted status"><LoaderCircle size={14} aria-hidden="true" />{t('ft.regex.testing')}</p>
    {:else if hits.length === 0}
      <p class="ft-muted">{t('ft.regex.noMatches')}</p>
    {:else}
      <p class="count ft-muted">{plural(t, 'ft.regex.matches', hits.length)}</p>
      <ul class:stale={loading}>
        {#each hits as hit (hit.sourceRel)}
          <li title={hit.sourceRel === '.' ? t('ft.common.templateFolder') : hit.sourceRel}>
            <span class="icon" class:dir={hit.isDir}>
              {#if hit.isDir}<Folder size={14} aria-hidden="true" />{:else}<FileText size={14} aria-hidden="true" />{/if}
            </span>
            <span class="ft-mono before">{hit.name}</span>
            <ArrowRight size={14} aria-hidden="true" class="arrow" />
            <span class="ft-mono after">{hit.result}</span>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</section>

<style>
  .ft-regex {
    display: flex;
    flex-direction: column;
    gap: 12px;
    font-size: 13px;
    color: var(--ft-text);
  }

  .inputs {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
    gap: 10px;
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

  .hint {
    font-size: 12px;
  }

  p {
    margin: 0;
  }

  .status {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .count {
    margin-bottom: 4px;
    font-size: 12px;
  }

  .error {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 8px 12px;
    border-radius: var(--ft-radius);
    background: var(--ft-danger-bg);
    color: var(--ft-danger-text);
  }

  ul {
    margin: 0;
    padding: 0;
    list-style: none;
    max-height: var(--ft-regex-max-height, 280px);
    overflow: auto;
    transition: opacity var(--ft-duration);
  }

  ul.stale {
    opacity: 0.6;
  }

  li {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 4px 6px;
    border-radius: var(--ft-radius);
    white-space: nowrap;
  }

  li:hover {
    background: var(--ft-subtle);
  }

  .icon {
    display: inline-flex;
    color: var(--ft-text-muted);
  }

  .icon.dir {
    color: var(--ft-accent);
  }

  .before {
    color: var(--ft-text-secondary);
  }

  li :global(.arrow) {
    flex: none;
    color: var(--ft-text-muted);
  }

  .after {
    color: var(--ft-accent);
  }
</style>
