<script lang="ts">
  import { onDestroy, untrack } from 'svelte'
  import {
    CircleAlert,
    CircleCheck,
    CircleQuestionMark,
    CircleSlash,
    FileExclamationPoint,
    FileX,
    LoaderCircle,
    Plus,
    RefreshCw,
    ToggleLeft,
  } from 'lucide-svelte'
  import type { Parameter, ScanIssue, ScanIssueKind, ScanResult, TemplateDescriptor, TokenInfo } from '../types'
  import { getFolderTemplatesContext } from '../context'
  import { debounce, latestOnly } from '../lib/debounce'
  import { cloneJson, errorMessage } from '../lib/errors'
  import { parameterFromToken } from '../lib/params'
  import { plural } from '../lib/host'

  interface Props {
    /** Template folder. */
    dir: string
    /** Unsaved descriptor from an editor; omit to scan against the saved template. */
    descriptor?: TemplateDescriptor
    /** Shows "Add parameter" on undeclared tokens. */
    onaddparameter?: (parameter: Parameter) => void
    debounceMs?: number
  }

  let { dir, descriptor, onaddparameter, debounceMs = 400 }: Props = $props()

  const { backend, t } = getFolderTemplatesContext()

  let scan = $state<ScanResult | null>(null)
  let error = $state<string | null>(null)
  let loading = $state(false)

  const latest = latestOnly()
  async function run(forDir: string, forDescriptor: TemplateDescriptor | undefined) {
    const ticket = latest.next()
    loading = true
    try {
      const next = await backend.scan(forDir, forDescriptor)
      if (!latest.isLatest(ticket)) return
      scan = next
      error = null
    } catch (e) {
      if (!latest.isLatest(ticket)) return
      error = errorMessage(e)
    }
    loading = false
  }

  const schedule = debounce(run, untrack(() => debounceMs))
  onDestroy(() => schedule.cancel())

  let lastDir: string | undefined
  let lastKey = ''
  $effect(() => {
    const d = cloneJson(descriptor)
    const key = JSON.stringify({ dir, d })
    if (key === lastKey) return
    lastKey = key
    // A new folder scans at once; descriptor edits wait for a pause.
    if (dir !== lastDir) {
      lastDir = dir
      schedule.cancel()
      void run(dir, d)
    } else {
      schedule(dir, d)
    }
  })

  function refresh() {
    schedule.cancel()
    void run(dir, cloneJson(descriptor))
  }

  const kindOrder: ScanIssueKind[] = [
    'undeclared-token',
    'token-in-unprocessed-file',
    'unprocessable-content-file',
    'unused-parameter',
    'name-replacement-off',
    'content-replacement-off',
  ]
  const kindIcon: Record<ScanIssueKind, typeof CircleAlert> = {
    'undeclared-token': CircleQuestionMark,
    'token-in-unprocessed-file': FileExclamationPoint,
    'unprocessable-content-file': FileX,
    'unused-parameter': CircleSlash,
    'name-replacement-off': ToggleLeft,
    'content-replacement-off': ToggleLeft,
  }

  const groups = $derived.by(() => {
    const byKind = new Map<ScanIssueKind, ScanIssue[]>()
    for (const issue of scan?.issues ?? []) {
      const list = byKind.get(issue.kind) ?? []
      list.push(issue)
      byKind.set(issue.kind, list)
    }
    const known = kindOrder.filter((k) => byKind.has(k))
    const other = [...byKind.keys()].filter((k) => !kindOrder.includes(k))
    return [...known, ...other].map((kind) => ({ kind, issues: byKind.get(kind) ?? [] }))
  })

  const pathList = (paths: string[]) => paths.map((p) => (p === '.' ? t('ft.common.templateFolder') : p)).join('\n')
</script>

<section class="ft-scan" aria-busy={loading}>
  <header>
    <h3>{t('ft.scan.title')}</h3>
    {#if loading}
      <span class="busy ft-muted"><span class="spin"><LoaderCircle size={14} aria-hidden="true" /></span>{t('ft.scan.scanning')}</span>
    {/if}
    <button type="button" class="ft-btn ft-btn-icon" aria-label={t('ft.common.refresh')} title={t('ft.common.refresh')} onclick={refresh}>
      <RefreshCw size={14} />
    </button>
  </header>

  {#if error}
    <p class="ft-error-text" role="alert">{error}</p>
  {/if}

  {#if scan}
    <div class="block">
      <h4>{t('ft.scan.issues')}</h4>
      {#if groups.length === 0}
        <p class="ok"><CircleCheck size={14} aria-hidden="true" />{t('ft.scan.noIssues')}</p>
      {/if}
      {#each groups as group (group.kind)}
        {@const Icon = kindIcon[group.kind] ?? CircleAlert}
        <div class="group">
          <h5><Icon size={14} />{t(`ft.scan.kinds.${group.kind}`)}</h5>
          <ul>
            {#each group.issues as issue}
              <li title={issue.path}>{issue.message}</li>
            {/each}
          </ul>
        </div>
      {/each}
    </div>

    <div class="block">
      <h4>{t('ft.scan.tokens')}</h4>
      {#if scan.tokens.length === 0}
        <p class="ft-muted">{t('ft.scan.noTokens')}</p>
      {:else}
        <ul class="tokens">
          {#each scan.tokens as token (token.name)}
            {@render tokenRow(token)}
          {/each}
        </ul>
      {/if}
    </div>
  {/if}
</section>

{#snippet tokenRow(token: TokenInfo)}
  <li class:undeclared={!token.declared}>
    <span class="token ft-mono">{token.name}</span>
    <span class="where ft-muted">
      {#if token.inNames.length}
        <span title={pathList(token.inNames)}>{plural(t, 'ft.scan.inNames', token.inNames.length)}</span>
      {/if}
      {#if token.inContent.length}
        <span title={pathList(token.inContent)}>{plural(t, 'ft.scan.inContent', token.inContent.length)}</span>
      {/if}
    </span>
    <span class="state">{token.declared ? t('ft.scan.declared') : t('ft.scan.undeclared')}</span>
    {#if !token.declared && onaddparameter}
      <button
        type="button"
        class="ft-btn add"
        aria-label={t('ft.scan.addParameterFor', { name: token.name })}
        onclick={() => onaddparameter(parameterFromToken(token))}
      >
        <Plus size={14} aria-hidden="true" />{t('ft.scan.addParameter')}
      </button>
    {/if}
  </li>
{/snippet}

<style>
  .ft-scan {
    display: flex;
    flex-direction: column;
    gap: 12px;
    font-size: 13px;
    color: var(--ft-text);
  }

  header {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  h3 {
    margin: 0;
    color: var(--ft-text-strong);
    font-size: 14px;
    font-weight: 600;
  }

  .busy {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 12px;
  }

  header .ft-btn {
    margin-left: auto;
  }

  .spin {
    display: inline-flex;
    animation: ft-spin 1s linear infinite;
  }

  @keyframes ft-spin {
    to {
      transform: rotate(360deg);
    }
  }

  p {
    margin: 0;
  }

  .block {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  h4 {
    margin: 0;
    color: var(--ft-text-secondary);
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .ok {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--ft-success);
  }

  .group h5 {
    display: flex;
    align-items: center;
    gap: 6px;
    margin: 0 0 2px;
    color: var(--ft-warning);
    font-size: 13px;
    font-weight: 500;
  }

  ul {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .group li {
    padding: 2px 0 2px 20px;
    color: var(--ft-text-secondary);
  }

  .tokens li {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px;
    padding: 5px 8px;
    border-radius: var(--ft-radius);
  }

  .tokens li:hover {
    background: var(--ft-subtle);
  }

  .token {
    color: var(--ft-template);
  }

  .where {
    display: inline-flex;
    gap: 8px;
    font-size: 12px;
  }

  .where span {
    cursor: help;
  }

  .state {
    margin-left: auto;
    font-size: 12px;
    color: var(--ft-text-muted);
  }

  .undeclared .state {
    color: var(--ft-warning);
  }

  .add {
    padding: 3px 8px;
    font-size: 12px;
  }
</style>
