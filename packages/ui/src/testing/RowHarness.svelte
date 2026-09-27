<script lang="ts">
  import { untrack, type ComponentProps } from 'svelte'
  import ParameterRow from '../components/ParameterRow.svelte'
  import type { ParameterRow as Row } from '../lib/params'

  type Props = Omit<ComponentProps<typeof ParameterRow>, 'row'> & { initial: Row }

  let { initial, ...rest }: Props = $props()

  let row = $state<Row>(untrack(() => initial))
</script>

<ParameterRow bind:row {...rest} />
<output data-testid="row">{JSON.stringify(row)}</output>
