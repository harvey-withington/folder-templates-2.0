<script lang="ts">
  import { untrack } from 'svelte'
  import { CircleAlert, Plus } from 'lucide-svelte'
  import type { TemplateDescriptor, ValidationIssue } from '../types'
  import { blankRow, fromRows, move, toRows, type ParameterRow as Row } from '../lib/params'
  import { useUiHost } from '../lib/host'
  import ParameterRow from './ParameterRow.svelte'

  interface Props {
    /** Edited in place: name, description, defaultTargetPath and (normalized) parameters are written back. */
    descriptor: TemplateDescriptor
    /** From the host's validate(); `param` "" (or an unknown name) shows at the top. */
    issues?: ValidationIssue[]
    /** The template folder's name, used as a placeholder hint. */
    folderName?: string
    ontestmatch?: (pattern: string, name: string) => void
    onchange?: (descriptor: TemplateDescriptor) => void
  }

  let { descriptor = $bindable(), issues = [], folderName = '', ontestmatch, onchange }: Props = $props()

  const { t, confirm } = useUiHost()
  const uid = $props.id()

  interface Meta {
    name: string
    description: string
    defaultTargetPath: string
  }
  const metaOf = (d: TemplateDescriptor): Meta => ({
    name: d.name,
    description: d.description,
    defaultTargetPath: d.defaultTargetPath,
  })
  const build = (m: Meta, r: Row[]): TemplateDescriptor => ({ ...m, parameters: fromRows(r) })
  const keyOf = (m: Meta, r: Row[]) => JSON.stringify(build(m, r))

  // Local editing state, re-initialised only when a different descriptor object arrives.
  let source = untrack(() => descriptor)
  let meta = $state(untrack(() => metaOf(descriptor)))
  let rows = $state<Row[]>(untrack(() => toRows(descriptor.parameters)))
  let written = untrack(() => keyOf(meta, rows))
  let focusId = $state<string | null>(null)

  $effect.pre(() => {
    const d = descriptor
    if (d === source) return
    untrack(() => {
      source = d
      meta = metaOf(d)
      rows = toRows(d.parameters)
      written = keyOf(meta, rows)
    })
  })

  // Write edits back into the descriptor (mutating it, so a bound $state object updates).
  $effect(() => {
    const next = build(meta, rows)
    const key = JSON.stringify(next)
    if (key === written) return
    written = key
    untrack(() => {
      const d = descriptor
      d.name = next.name
      d.description = next.description
      d.defaultTargetPath = next.defaultTargetPath
      d.parameters = next.parameters
      onchange?.(d)
    })
  })

  const rowIssues = (row: Row) => issues.filter((i) => i.param !== '' && i.param === row.param.name.trim())
  const generalIssues = $derived(
    issues.filter((i) => i.param === '' || !rows.some((r) => r.param.name.trim() === i.param)),
  )

  function add() {
    const row = blankRow()
    rows.push(row)
    focusId = row.id
  }

  function shift(id: string, by: number) {
    const from = rows.findIndex((r) => r.id === id)
    rows = move(rows, from, from + by)
  }

  async function remove(row: Row) {
    const name = row.param.name.trim()
    if (name || (row.param.prompt ?? '').trim()) {
      const message = name ? t('ft.editor.confirmDelete', { name }) : t('ft.editor.confirmDeleteUnnamed')
      if (!(await confirm(message, { danger: true, confirmLabel: t('ft.editor.delete') }))) return
    }
    rows = rows.filter((r) => r.id !== row.id)
  }

  // Drag-and-drop reordering, armed only from the grip so text in inputs stays selectable.
  let armed = $state<string | null>(null)
  let dragging = $state<string | null>(null)
  let over = $state<string | null>(null)

  function pointerdown(e: PointerEvent, id: string) {
    armed = e.target instanceof Element && e.target.closest('[data-grip]') ? id : null
  }

  function dragstart(e: DragEvent, id: string) {
    dragging = id
    if (e.dataTransfer) {
      e.dataTransfer.effectAllowed = 'move'
      e.dataTransfer.setData('text/plain', id)
    }
  }

  function dragover(e: DragEvent, id: string) {
    if (!dragging) return
    e.preventDefault()
    over = id
  }

  function drop(e: DragEvent, id: string) {
    e.preventDefault()
    if (dragging && dragging !== id) {
      rows = move(
        rows,
        rows.findIndex((r) => r.id === dragging),
        rows.findIndex((r) => r.id === id),
      )
    }
    dragend()
  }

  function dragend() {
    armed = dragging = over = null
  }
</script>

<div class="ft-editor">
  {#if generalIssues.length}
    <div class="issues" role="alert" aria-label={t('ft.editor.issues')}>
      {#each generalIssues as issue}
        <p class="ft-error-text"><CircleAlert size={14} aria-hidden="true" />{issue.message}</p>
      {/each}
    </div>
  {/if}

  <div class="field">
    <label for="{uid}-name">{t('ft.editor.name')}</label>
    <input id="{uid}-name" type="text" bind:value={meta.name} placeholder={folderName} autocomplete="off" />
  </div>

  <div class="field">
    <label for="{uid}-description">{t('ft.editor.description')}</label>
    <textarea id="{uid}-description" rows="2" bind:value={meta.description} placeholder={t('ft.editor.descriptionPlaceholder')}></textarea>
  </div>

  <div class="field">
    <label for="{uid}-target">{t('ft.editor.target')}</label>
    <input
      id="{uid}-target"
      type="text"
      class="ft-mono"
      bind:value={meta.defaultTargetPath}
      placeholder={t('ft.editor.targetPlaceholder')}
      aria-describedby="{uid}-target-hint"
      autocomplete="off"
      spellcheck="false"
    />
    <span id="{uid}-target-hint" class="hint ft-muted">{t('ft.editor.targetHint')}</span>
  </div>

  <section class="params" aria-labelledby="{uid}-params">
    <div class="params-head">
      <h3 id="{uid}-params">{t('ft.editor.parameters')}</h3>
      <span class="hint ft-muted">{t('ft.editor.parametersHint')}</span>
    </div>

    {#if rows.length === 0}
      <p class="empty ft-muted">{t('ft.editor.noParameters')}</p>
    {/if}

    <ul class="list">
      {#each rows as row, i (row.id)}
        <li
          class:dragging={dragging === row.id}
          class:over={over === row.id && dragging !== row.id}
          draggable={armed === row.id}
          onpointerdown={(e) => pointerdown(e, row.id)}
          ondragstart={(e) => dragstart(e, row.id)}
          ondragover={(e) => dragover(e, row.id)}
          ondrop={(e) => drop(e, row.id)}
          ondragend={dragend}
        >
          <ParameterRow
            bind:row={rows[i]}
            index={i}
            count={rows.length}
            issues={rowIssues(row)}
            autofocus={focusId === row.id}
            onmoveup={() => shift(row.id, -1)}
            onmovedown={() => shift(row.id, 1)}
            ondelete={() => remove(row)}
            {ontestmatch}
          />
        </li>
      {/each}
    </ul>

    <div>
      <button type="button" class="ft-btn" onclick={add}><Plus size={14} aria-hidden="true" />{t('ft.editor.add')}</button>
    </div>
  </section>
</div>

<style>
  .ft-editor {
    display: flex;
    flex-direction: column;
    gap: 14px;
    font-size: 13px;
    color: var(--ft-text);
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

  textarea {
    resize: vertical;
  }

  .hint {
    font-size: 12px;
  }

  .issues {
    padding: 8px 12px;
    border: 1px solid var(--ft-danger);
    border-radius: var(--ft-radius);
    background: var(--ft-danger-bg);
  }

  .issues p {
    display: flex;
    align-items: center;
    gap: 6px;
    margin: 2px 0;
    color: var(--ft-danger-text);
  }

  .params {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .params-head {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  h3 {
    margin: 0;
    color: var(--ft-text-strong);
    font-size: 14px;
    font-weight: 600;
  }

  .empty {
    margin: 0;
    padding: 12px;
    border: 1px dashed var(--ft-border-muted);
    border-radius: var(--ft-radius);
    text-align: center;
  }

  .list {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  li {
    border-radius: var(--ft-radius);
    transition: opacity var(--ft-duration), box-shadow var(--ft-duration);
  }

  li.dragging {
    opacity: 0.5;
  }

  li.over {
    box-shadow: 0 -2px 0 0 var(--ft-accent);
  }
</style>
