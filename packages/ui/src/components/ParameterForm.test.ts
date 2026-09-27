import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/svelte'
import userEvent from '@testing-library/user-event'
import FormHarness from '../testing/FormHarness.svelte'
import ParameterForm from './ParameterForm.svelte'
import { blankParameter } from '../lib/params'
import type { Parameter } from '../types'

const param = (over: Partial<Parameter>): Parameter => ({ ...blankParameter(), ...over })

const parameters = [
  param({ name: 'client', prompt: 'Client name', placeholder: 'Acme', defaultValue: 'Default Co' }),
  param({ name: 'year', prompt: null, defaultValue: '2026' }),
  param({ name: '', prompt: 'Nameless rule', match: '^_' }),
  param({ name: 'code', prompt: 'Project code' }),
]

const shownValues = () => JSON.parse(screen.getByTestId('values').textContent ?? '{}')

describe('ParameterForm', () => {
  it('renders one labelled input per prompted parameter only', () => {
    render(ParameterForm, { props: { parameters } })
    expect(screen.getAllByRole('textbox')).toHaveLength(2)
    expect(screen.getByLabelText('Client name')).toHaveAttribute('placeholder', 'Acme')
    expect(screen.getByLabelText('Project code')).toBeInTheDocument()
    expect(screen.queryByLabelText('Nameless rule')).not.toBeInTheDocument()
  })

  it('fills defaults into the bound values and keeps typed answers', async () => {
    const user = userEvent.setup()
    render(FormHarness, { props: { parameters, initial: { code: 'X1' } } })
    expect(shownValues()).toEqual({ client: 'Default Co', code: 'X1' })
    expect(screen.getByLabelText('Project code')).toHaveValue('X1')

    const client = screen.getByLabelText('Client name')
    await user.clear(client)
    await user.type(client, 'Globex')
    expect(shownValues()).toEqual({ client: 'Globex', code: 'X1' })
  })

  it('calls onsubmit on Enter', async () => {
    const user = userEvent.setup()
    const onsubmit = vi.fn()
    render(FormHarness, { props: { parameters, onsubmit } })
    await user.type(screen.getByLabelText('Project code'), 'abc{Enter}')
    expect(onsubmit).toHaveBeenCalledOnce()
  })

  it('focuses the first input when asked', () => {
    render(ParameterForm, { props: { parameters, autofocus: true } })
    expect(screen.getByLabelText('Client name')).toHaveFocus()
  })

  it('disables inputs', () => {
    render(ParameterForm, { props: { parameters, disabled: true } })
    expect(screen.getByLabelText('Client name')).toBeDisabled()
  })

  it('shows an empty state when nothing is asked', () => {
    render(ParameterForm, { props: { parameters: [param({ name: 'x', prompt: null })] } })
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument()
    expect(screen.getByText(/doesn't ask any questions/)).toBeInTheDocument()
  })

  it('gives every instance unique input ids', () => {
    render(ParameterForm, { props: { parameters } })
    render(ParameterForm, { props: { parameters } })
    const ids = screen.getAllByRole('textbox').map((el) => el.id)
    expect(new Set(ids).size).toBe(4)
  })
})
