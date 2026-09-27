<script lang="ts">
  import { untrack } from 'svelte'
  import ParameterForm from '../components/ParameterForm.svelte'
  import type { Parameter, Values } from '../types'

  interface Props {
    parameters: Parameter[]
    initial?: Values
    autofocus?: boolean
    onsubmit?: () => void
  }

  let { parameters, initial = {}, autofocus = false, onsubmit }: Props = $props()

  // A host-style parent that binds the answers.
  let values = $state<Values>(untrack(() => ({ ...initial })))
</script>

<ParameterForm {parameters} bind:values {autofocus} {onsubmit} />
<output data-testid="values">{JSON.stringify(values)}</output>
