<script lang="ts">
  import ParameterForm from '../components/ParameterForm.svelte'
  import LivePreview from '../components/LivePreview.svelte'
  import type { Parameter, Values } from '../types'

  let { parameters }: { parameters: Parameter[] } = $props()

  // The host pattern: one $state answers object, edited in place by the form.
  let values = $state<Values>({})
</script>

<ParameterForm {parameters} bind:values />
<LivePreview dir="/tpl" {values} target="/out" />
