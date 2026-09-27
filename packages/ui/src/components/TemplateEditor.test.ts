import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, within } from '@testing-library/svelte'
import userEvent from '@testing-library/user-event'
import WithHost from '../testing/WithHost.svelte'
import TemplateEditor from './TemplateEditor.svelte'
import { createFakeBackend, sampleDescriptor, settle } from '../testing/fakeBackend'
import type { TemplateDescriptor, ValidationIssue } from '../types'

interface Options {
  descriptor?: TemplateDescriptor
  issues?: ValidationIssue[]
  confirm?: (message: string) => Promise<boolean>
  ontestmatch?: (pattern: string, name: string) => void
}

function setup(options: Options = {}) {
  const descriptor = options.descriptor ?? sampleDescriptor()
  const onchange = vi.fn<(d: TemplateDescriptor) => void>()
  const confirm = vi.fn(options.confirm ?? (() => Promise.resolve(true)))
  const utils = render(WithHost, {
    props: {
      backend: createFakeBackend(),
      component: TemplateEditor,
      confirm,
      componentProps: { descriptor, issues: options.issues ?? [], folderName: 'client-folder', onchange, ontestmatch: options.ontestmatch },
    },
  })
  return { ...utils, descriptor, onchange, confirm, user: userEvent.setup() }
}

const card = (name: string) => screen.getByRole('group', { name: `Parameter ${name}` })
const names = (d: TemplateDescriptor) => d.parameters.map((p) => p.name)

describe('TemplateEditor', () => {
  it('shows the descriptor fields and one card per parameter', () => {
    setup()
    expect(screen.getByLabelText('Template name')).toHaveValue('Client project')
    expect(screen.getByLabelText('Template name')).toHaveAttribute('placeholder', 'client-folder')
    expect(screen.getByLabelText('Description')).toHaveValue('A folder per client')
    expect(screen.getByLabelText('Default target folder')).toHaveValue('../Clients')
    expect(screen.getByText(/resolve against the folder that contains the template/)).toBeInTheDocument()
    expect(within(card('client')).getByLabelText('Prompt')).toHaveValue('Client name')
    expect(within(card('year')).queryByLabelText('Prompt')).not.toBeInTheDocument()
    expect(within(card('year')).getByRole('switch')).toHaveAttribute('aria-checked', 'false')
  })

  it('writes field edits back and reports them', async () => {
    const { user, descriptor, onchange } = setup()
    expect(onchange).not.toHaveBeenCalled()
    await user.type(screen.getByLabelText('Description'), '!')
    expect(descriptor.description).toBe('A folder per client!')
    expect(onchange).toHaveBeenLastCalledWith(descriptor)
  })

  it('adds a parameter, focuses its name and writes it normalized', async () => {
    const { user, descriptor } = setup()
    await user.click(screen.getByRole('button', { name: 'Add parameter' }))
    const fresh = screen.getByRole('group', { name: 'Unnamed parameter' })
    const nameInput = within(fresh).getByLabelText('Name')
    expect(nameInput).toHaveFocus()
    expect(within(fresh).getByText('Rename rule')).toBeInTheDocument()

    await user.type(nameInput, 'project')
    await user.type(within(card('project')).getByLabelText('Prompt'), 'Project?')
    expect(descriptor.parameters[2]).toEqual({
      name: 'project',
      type: 'text',
      prompt: 'Project?',
      placeholder: null,
      defaultValue: null,
      match: null,
      replaceInFileNames: true,
      replaceInFiles: true,
    })
  })

  it('switches between asked and internal, remembering the prompt', async () => {
    const { user, descriptor } = setup()
    const toggle = within(card('client')).getByRole('switch', { name: 'Ask for this value when generating' })
    await user.click(toggle)
    expect(descriptor.parameters[0].prompt).toBeNull()
    expect(within(card('client')).queryByLabelText('Prompt')).not.toBeInTheDocument()
    await user.click(toggle)
    expect(descriptor.parameters[0].prompt).toBe('Client name')
    expect(within(card('client')).getByLabelText('Prompt')).toHaveValue('Client name')
  })

  it('toggles name and file replacement', async () => {
    const { user, descriptor } = setup()
    await user.click(within(card('client')).getByRole('button', { name: 'Files' }))
    expect(descriptor.parameters[0].replaceInFiles).toBe(false)
    expect(within(card('client')).getByRole('button', { name: 'Files' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('reorders with the move buttons', async () => {
    const { user, descriptor } = setup()
    expect(within(card('client')).getByRole('button', { name: 'Move up' })).toBeDisabled()
    await user.click(within(card('client')).getByRole('button', { name: 'Move down' }))
    expect(names(descriptor)).toEqual(['year', 'client'])
    expect(within(card('client')).getByRole('button', { name: 'Move down' })).toBeDisabled()
  })

  it('reorders by drag and drop', async () => {
    const { descriptor } = setup()
    const items = screen.getAllByRole('listitem')
    await fireEvent.dragStart(items[1])
    await fireEvent.dragOver(items[0])
    await fireEvent.drop(items[0])
    expect(names(descriptor)).toEqual(['year', 'client'])
  })

  it('confirms before deleting a named parameter', async () => {
    let answer = false
    const { user, descriptor, confirm } = setup({ confirm: () => Promise.resolve(answer) })
    await user.click(within(card('client')).getByRole('button', { name: 'Delete parameter' }))
    expect(confirm).toHaveBeenCalledWith(expect.stringContaining('client'), expect.objectContaining({ danger: true }))
    expect(names(descriptor)).toEqual(['client', 'year'])

    answer = true
    await user.click(within(card('client')).getByRole('button', { name: 'Delete parameter' }))
    await settle()
    expect(names(descriptor)).toEqual(['year'])
  })

  it('deletes an empty parameter without asking', async () => {
    const { user, confirm } = setup()
    await user.click(screen.getByRole('button', { name: 'Add parameter' }))
    const fresh = screen.getByRole('group', { name: 'Unnamed parameter' })
    await user.click(within(fresh).getByRole('button', { name: 'Delete parameter' }))
    expect(confirm).not.toHaveBeenCalled()
    expect(screen.queryByRole('group', { name: 'Unnamed parameter' })).not.toBeInTheDocument()
  })

  it('shows row issues under their row and the rest at the top', () => {
    setup({
      issues: [
        { param: 'client', message: 'Pattern is not valid' },
        { param: '', message: 'Two parameters share a name' },
        { param: 'gone', message: 'Unknown parameter' },
      ],
    })
    expect(within(card('client')).getByText('Pattern is not valid')).toBeInTheDocument()
    const top = screen.getByRole('alert')
    expect(top).toHaveTextContent('Two parameters share a name')
    expect(top).toHaveTextContent('Unknown parameter')
    expect(top).not.toHaveTextContent('Pattern is not valid')
  })

  it('tests the match pattern, defaulting to {name}', async () => {
    const ontestmatch = vi.fn()
    const { user } = setup({ ontestmatch })
    await user.click(within(card('client')).getByRole('button', { name: 'More options' }))
    const match = within(card('client')).getByLabelText('Match pattern')
    expect(match).toHaveAttribute('placeholder', '\\{client\\}')
    await user.click(within(card('client')).getByRole('button', { name: 'Test pattern' }))
    expect(ontestmatch).toHaveBeenLastCalledWith('\\{client\\}', 'client')
    await user.type(match, '^cl')
    await user.click(within(card('client')).getByRole('button', { name: 'Test pattern' }))
    expect(ontestmatch).toHaveBeenLastCalledWith('^cl', 'client')
  })

  it('stores emptied optional fields as null', async () => {
    const descriptor = sampleDescriptor()
    const { user } = setup({ descriptor })
    await user.click(within(card('client')).getByRole('button', { name: 'More options' }))
    await user.clear(within(card('client')).getByLabelText('Placeholder'))
    expect(descriptor.parameters[0].placeholder).toBeNull()
  })

  it('re-initialises rows only when a different descriptor object arrives', async () => {
    const { rerender, onchange } = setup()
    const other: TemplateDescriptor = { ...sampleDescriptor(), name: 'Other', parameters: [] }
    await rerender({
      backend: createFakeBackend(),
      component: TemplateEditor,
      componentProps: { descriptor: other, onchange },
    })
    expect(screen.getByLabelText('Template name')).toHaveValue('Other')
    expect(screen.queryByRole('group', { name: /Parameter/ })).not.toBeInTheDocument()
    expect(onchange).not.toHaveBeenCalled()
  })
})
