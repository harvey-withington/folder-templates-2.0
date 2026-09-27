<script lang="ts">
  import {
    LivePreview,
    ParameterForm,
    previewValues,
    type TemplateDescriptor,
    type Values,
  } from '@harvey-withington/folder-templates-ui'
  import { t } from '../lib/i18n'

  interface Props {
    dir: string
    descriptor: TemplateDescriptor
    defaultTarget: string
  }

  let { dir, descriptor, defaultTarget }: Props = $props()

  // ParameterForm fills in each asked parameter's default as it renders.
  let values = $state<Values>({})
  // Previews never write, so any destination works; the template's parent is
  // what a real run would default to.
  const target = $derived(defaultTarget || dir.replace(/[\\/][^\\/]+$/, ''))
</script>

<div class="try">
  <p class="ft-muted">{t('app.editor.tryHint')}</p>
  <ParameterForm parameters={descriptor.parameters} bind:values />
  <LivePreview {dir} values={previewValues(descriptor.parameters, values)} {target} {descriptor} />
</div>

<style>
  .try {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  p {
    margin: 0;
  }
</style>
