import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, within } from '@testing-library/svelte'
import WithHost from '../testing/WithHost.svelte'
import TokenScanPanel from './TokenScanPanel.svelte'
import { createFakeBackend, sampleDescriptor, settle, type FakeBackend } from '../testing/fakeBackend'
import type { Parameter, TemplateDescriptor } from '../types'

type Props = {
  dir: string
  descriptor?: TemplateDescriptor
  onaddparameter?: (p: Parameter) => void
}

function setup(backend: FakeBackend, initial: Partial<Props> = {}) {
  let props: Props = { dir: '/tpl', ...initial }
  const utils = render(WithHost, { props: { backend, component: TokenScanPanel, componentProps: props } })
  const update = async (next: Partial<Props>) => {
    props = { ...props, ...next }
    await utils.rerender({ componentProps: props })
  }
  return { ...utils, update }
}

const tokenRow = (name: string) => {
  const row = screen.getAllByRole('listitem').find((li) => li.querySelector('.token')?.textContent === name)
  if (!row) throw new Error(`no token ${name}`)
  return row
}

describe('TokenScanPanel', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('scans on mount and lists issues by kind, then tokens', async () => {
    const backend = createFakeBackend()
    const descriptor = sampleDescriptor()
    setup(backend, { descriptor })
    expect(backend.scan).toHaveBeenCalledExactlyOnceWith('/tpl', descriptor)
    await settle()

    expect(screen.getByText('Tokens without a parameter')).toBeInTheDocument()
    expect(screen.getByText('Token {projectCode} has no parameter')).toBeInTheDocument()
    expect(screen.getByText('Unused parameters')).toBeInTheDocument()

    const client = tokenRow('client')
    expect(within(client).getByText('2 names')).toHaveAttribute('title', '(template folder)\ndocs/{client} brief.md.ft$')
    expect(within(client).getByText('1 file')).toHaveAttribute('title', 'readme.md.ft$')
    expect(client).toHaveTextContent('Parameter')
    expect(tokenRow('projectCode')).toHaveTextContent('Not a parameter')
  })

  it('adds a parameter for an undeclared token', async () => {
    const onaddparameter = vi.fn()
    setup(createFakeBackend(), { onaddparameter })
    await settle()
    expect(within(tokenRow('client')).queryByRole('button')).not.toBeInTheDocument()
    await fireEvent.click(screen.getByRole('button', { name: 'Add parameter for projectCode' }))
    expect(onaddparameter).toHaveBeenCalledWith(
      expect.objectContaining({ name: 'projectCode', prompt: 'Project code', replaceInFileNames: true, replaceInFiles: false }),
    )
  })

  it('hides add buttons without a handler', async () => {
    setup(createFakeBackend())
    await settle()
    expect(screen.queryByRole('button', { name: /Add parameter/ })).not.toBeInTheDocument()
  })

  it('rescans on refresh and, debounced, on descriptor edits', async () => {
    vi.useFakeTimers()
    const backend = createFakeBackend()
    const { update } = setup(backend, { descriptor: sampleDescriptor() })
    await settle()
    await fireEvent.click(screen.getByRole('button', { name: 'Refresh' }))
    expect(backend.scan).toHaveBeenCalledTimes(2)

    await update({ descriptor: { ...sampleDescriptor(), name: 'edited' } })
    await update({ descriptor: { ...sampleDescriptor(), name: 'edited again' } })
    await vi.advanceTimersByTimeAsync(399)
    expect(backend.scan).toHaveBeenCalledTimes(2)
    await vi.advanceTimersByTimeAsync(1)
    expect(backend.scan).toHaveBeenCalledTimes(3)
    expect(backend.scan.mock.calls[2][1]).toMatchObject({ name: 'edited again' })

    await update({ dir: '/other' })
    expect(backend.scan).toHaveBeenCalledTimes(4)
  })

  it('shows empty states', async () => {
    setup(createFakeBackend({ scan: async () => ({ tokens: [], params: [], issues: [] }) }))
    await settle()
    expect(screen.getByText('No problems found.')).toBeInTheDocument()
    expect(screen.getByText(/No tokens found/)).toBeInTheDocument()
  })

  it('shows scan errors', async () => {
    setup(createFakeBackend({ scan: () => Promise.reject(new Error('Folder not found')) }))
    await settle()
    expect(screen.getByRole('alert')).toHaveTextContent('Folder not found')
  })
})
