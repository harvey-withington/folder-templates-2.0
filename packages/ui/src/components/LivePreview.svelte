<script lang="ts">
  import { onDestroy, untrack } from 'svelte'
  import type { PreviewResult, RenderedFile, TemplateDescriptor, Values } from '../types'
  import { getFolderTemplatesContext } from '../context'
  import { debounce, latestOnly } from '../lib/debounce'
  import { cloneJson, errorMessage } from '../lib/errors'
  import PreviewTree from './PreviewTree.svelte'
  import FileDiffView from './FileDiffView.svelte'

  interface Props {
    /** Template folder. */
    dir: string
    values: Values
    /** Folder the output root is created in. */
    target: string
    /** Unsaved descriptor from an editor; omit to use the saved template. */
    descriptor?: TemplateDescriptor
    debounceMs?: number
    /** Every settled preview: the result, or null with the error message. */
    onresult?: (result: PreviewResult | null, error: string | null) => void
  }

  let { dir, values, target, descriptor, debounceMs = 250, onresult }: Props = $props()

  const { backend } = getFolderTemplatesContext()

  let result = $state<PreviewResult | null>(null)
  let error = $state<string | null>(null)
  let loading = $state(false)

  let selected = $state<string | undefined>()
  let rendered = $state<RenderedFile | null>(null)
  let renderError = $state<string | null>(null)
  let renderLoading = $state(false)

  interface Inputs {
    dir: string
    values: Values
    target: string
    descriptor?: TemplateDescriptor
  }

  interface RenderInputs {
    dir: string
    sourceRel: string
    values: Values
    descriptor?: TemplateDescriptor
  }

  const previews = latestOnly()
  async function runPreview(req: Inputs) {
    const ticket = previews.next()
    loading = true
    let next: PreviewResult | null = null
    let failure: string | null = null
    try {
      next = await backend.preview(req.dir, req.values, req.target, req.descriptor)
    } catch (e) {
      failure = errorMessage(e)
    }
    if (!previews.isLatest(ticket)) return
    result = next
    error = failure
    loading = false
    if (selected !== undefined && !next?.entries.some((e) => e.processed && e.sourceRel === selected)) {
      selected = undefined
    }
    onresult?.(next, failure)
  }

  const renders = latestOnly()
  async function runRender(req: RenderInputs) {
    const ticket = renders.next()
    renderLoading = true
    try {
      const file = await backend.renderFile(req.dir, req.sourceRel, req.values, req.descriptor)
      if (!renders.isLatest(ticket)) return
      rendered = file
      renderError = null
    } catch (e) {
      if (!renders.isLatest(ticket)) return
      rendered = null
      renderError = errorMessage(e)
    }
    renderLoading = false
  }

  const wait = untrack(() => debounceMs)
  const schedulePreview = debounce((req: Inputs) => void runPreview(req), wait)
  const scheduleRender = debounce((req: RenderInputs) => void runRender(req), wait)
  onDestroy(() => {
    schedulePreview.cancel()
    scheduleRender.cancel()
  })

  // cloneJson reads every field, so edits deep inside values/descriptor re-run these effects.
  let previewKey = ''
  $effect(() => {
    const req: Inputs = { dir, target, values: cloneJson(values), descriptor: cloneJson(descriptor) }
    const key = JSON.stringify(req)
    if (key === previewKey) return
    previewKey = key
    if (req.dir) schedulePreview(req)
  })

  let renderKey = ''
  let renderedSource: string | undefined
  $effect(() => {
    const sourceRel = selected
    const req: RenderInputs = { dir, sourceRel: sourceRel ?? '', values: cloneJson(values), descriptor: cloneJson(descriptor) }
    const key = JSON.stringify(req)
    if (key === renderKey) return
    renderKey = key
    untrack(() => {
      if (sourceRel === undefined) {
        scheduleRender.cancel()
        renders.next() // drop anything in flight
        rendered = null
        renderError = null
        renderLoading = false
      } else if (sourceRel !== renderedSource) {
        scheduleRender.cancel()
        rendered = null
        void runRender(req)
      } else {
        scheduleRender(req)
      }
      renderedSource = sourceRel
    })
  })

  const selectedPath = $derived(
    result?.entries.find((e) => e.processed && e.sourceRel === selected)?.outputRel ?? selected ?? '',
  )
</script>

<div class="ft-live">
  <div class="layout" class:with-file={selected !== undefined}>
    <div class="pane">
      <PreviewTree {result} {loading} {error} bind:selected />
    </div>
    {#if selected !== undefined}
      <div class="pane">
        <FileDiffView
          path={selectedPath}
          {rendered}
          loading={renderLoading}
          error={renderError}
          onclose={() => (selected = undefined)}
        />
      </div>
    {/if}
  </div>
</div>

<style>
  .ft-live {
    container-type: inline-size;
  }

  .layout {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 12px;
  }

  @container (min-width: 720px) {
    .layout.with-file {
      grid-template-columns: minmax(0, 1fr) minmax(0, 1.2fr);
    }
  }

  .pane {
    min-width: 0;
  }
</style>
