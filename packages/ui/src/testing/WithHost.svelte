<script lang="ts">
  import { untrack, type Component } from 'svelte'
  import { setFolderTemplatesContext, type FolderTemplatesHost } from '../context'
  import type { TemplateBackend } from '../types'

  interface Props {
    backend: TemplateBackend
    /** Any component; Component<never> accepts every props shape. */
    component: Component<never>
    /** Not called `props`: testing-library reads rerender({ props }) as its deprecated options form. */
    componentProps?: Record<string, unknown>
    toast?: FolderTemplatesHost['toast']
    confirm?: FolderTemplatesHost['confirm']
  }

  let { backend, component, componentProps = {}, toast, confirm }: Props = $props()

  untrack(() => setFolderTemplatesContext({ backend, toast, confirm }))

  const Inner = $derived(component as unknown as Component<Record<string, unknown>>)
</script>

<div class="ft-scope">
  <Inner {...componentProps} />
</div>
