import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import userEvent from '@testing-library/user-event'
import RowHarness from '../testing/RowHarness.svelte'
import { blankParameter, toRows, type ParameterRow } from '../lib/params'
import type { Parameter } from '../types'

const rowOf = (over: Partial<Parameter>): ParameterRow => toRows([{ ...blankParameter(), ...over }])[0]
const shown = (): ParameterRow => JSON.parse(screen.getByTestId('row').textContent ?? '{}')

describe('ParameterRow', () => {
  it('edits the bound row', async () => {
    const user = userEvent.setup()
    render(RowHarness, { props: { initial: rowOf({ name: 'client', prompt: 'Client' }), index: 0, count: 1 } })
    await user.type(screen.getByLabelText('Prompt'), ' name')
    await user.click(screen.getByRole('button', { name: 'Names' }))
    expect(shown().param).toMatchObject({ prompt: 'Client name', replaceInFileNames: false, replaceInFiles: true })
  })

  it('remembers the prompt in savedPrompt while internal', async () => {
    const user = userEvent.setup()
    render(RowHarness, { props: { initial: rowOf({ name: 'client', prompt: 'Client' }), index: 0, count: 1 } })
    const toggle = screen.getByRole('switch')
    expect(toggle).toHaveTextContent('Asked')
    await user.click(toggle)
    expect(shown()).toMatchObject({ internal: true, savedPrompt: 'Client', param: { prompt: null } })
    expect(toggle).toHaveTextContent('Internal')
    expect(screen.getByText(/always uses the default value/)).toBeInTheDocument()
    await user.click(toggle)
    expect(shown()).toMatchObject({ internal: false, param: { prompt: 'Client' } })
  })

  it('shows a nameless row as a rename rule with its match visible', () => {
    render(RowHarness, { props: { initial: rowOf({ name: '', prompt: null, match: '^_draft' }), index: 0, count: 1 } })
    expect(screen.getByText('Rename rule')).toBeInTheDocument()
    expect(screen.getByLabelText('Match pattern')).toHaveValue('^_draft')
    expect(screen.getByLabelText('Replace with')).toBeInTheDocument()
    // Rules are never asked and only rename: no prompt, no mode chips.
    expect(screen.queryByLabelText('Prompt')).not.toBeInTheDocument()
    expect(screen.queryByRole('switch')).not.toBeInTheDocument()
  })

  it('hides extra fields until "More" is opened', async () => {
    const user = userEvent.setup()
    render(RowHarness, { props: { initial: rowOf({ name: 'client', prompt: 'Client' }), index: 0, count: 1 } })
    expect(screen.queryByLabelText('Match pattern')).not.toBeInTheDocument()
    const more = screen.getByRole('button', { name: 'More options' })
    await user.click(more)
    expect(more).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByLabelText('Default value')).toBeInTheDocument()
    expect(screen.getByLabelText('Placeholder')).toBeInTheDocument()
    expect(screen.getByText(/\.NET regular expression/)).toBeInTheDocument()
  })

  it('wires move and delete, disabling moves at the ends', async () => {
    const user = userEvent.setup()
    const onmoveup = vi.fn()
    const onmovedown = vi.fn()
    const ondelete = vi.fn()
    render(RowHarness, {
      props: { initial: rowOf({ name: 'a', prompt: 'A' }), index: 1, count: 2, onmoveup, onmovedown, ondelete },
    })
    expect(screen.getByRole('button', { name: 'Move down' })).toBeDisabled()
    await user.click(screen.getByRole('button', { name: 'Move up' }))
    await user.click(screen.getByRole('button', { name: 'Delete parameter' }))
    expect(onmoveup).toHaveBeenCalledOnce()
    expect(onmovedown).not.toHaveBeenCalled()
    expect(ondelete).toHaveBeenCalledOnce()
  })

  it('shows its issues and can focus its name', () => {
    render(RowHarness, {
      props: {
        initial: rowOf({ name: 'a', prompt: 'A' }),
        index: 0,
        count: 1,
        autofocus: true,
        issues: [{ param: 'a', message: 'Name is used twice' }],
      },
    })
    expect(screen.getByText('Name is used twice')).toBeInTheDocument()
    expect(screen.getByLabelText('Name')).toHaveFocus()
  })
})
