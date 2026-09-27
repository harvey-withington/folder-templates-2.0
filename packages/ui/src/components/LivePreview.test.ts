import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/svelte'
import WithHost from '../testing/WithHost.svelte'
import LivePreview from './LivePreview.svelte'
import FormAndPreview from '../testing/FormAndPreview.svelte'
import { blankParameter } from '../lib/params'
import { createFakeBackend, deferred, samplePreview, settle, type FakeBackend } from '../testing/fakeBackend'
import type { PreviewResult, TemplateDescriptor, Values } from '../types'

type Props = {
  dir: string
  values: Values
  target: string
  descriptor?: TemplateDescriptor
  onresult?: (r: PreviewResult | null, e: string | null) => void
}

function setup(backend: FakeBackend, initial: Partial<Props> = {}) {
  let props: Props = { dir: '/tpl', values: { client: 'A' }, target: '/out', ...initial }
  const utils = render(WithHost, { props: { backend, component: LivePreview, componentProps: props } })
  const update = async (next: Partial<Props>) => {
    props = { ...props, ...next }
    await utils.rerender({ componentProps: props })
  }
  return { ...utils, update }
}

const previewNamed = (rootName: string): PreviewResult => ({
  ...samplePreview(),
  rootName,
  entries: [{ sourceRel: '', outputRel: rootName, isDir: true, processed: false }],
})

const treeItemNamed = (name: string) =>
  screen.getAllByRole('treeitem').find((el) => el.querySelector('.name')?.textContent?.trim() === name)

describe('LivePreview', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('debounces preview calls while inputs change', async () => {
    const backend = createFakeBackend()
    const { update } = setup(backend)
    expect(backend.preview).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(249)
    expect(backend.preview).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(1)
    expect(backend.preview).toHaveBeenCalledExactlyOnceWith('/tpl', { client: 'A' }, '/out', undefined)

    await update({ values: { client: 'Ac' } })
    await vi.advanceTimersByTimeAsync(100)
    await update({ values: { client: 'Acm' } })
    await vi.advanceTimersByTimeAsync(100)
    await update({ values: { client: 'Acme' }, target: '/elsewhere' })
    await vi.advanceTimersByTimeAsync(250)
    expect(backend.preview).toHaveBeenCalledTimes(2)
    expect(backend.preview).toHaveBeenLastCalledWith('/tpl', { client: 'Acme' }, '/elsewhere', undefined)
  })

  it('does not re-request for equal inputs and passes the descriptor', async () => {
    const backend = createFakeBackend()
    const descriptor: TemplateDescriptor = { name: 'n', description: '', defaultTargetPath: '', parameters: [] }
    const { update } = setup(backend, { descriptor })
    await vi.advanceTimersByTimeAsync(250)
    await update({ values: { client: 'A' }, descriptor: { ...descriptor } })
    await vi.advanceTimersByTimeAsync(250)
    expect(backend.preview).toHaveBeenCalledOnce()
    expect(backend.preview.mock.calls[0][3]).toEqual(descriptor)
  })

  it('drops stale responses', async () => {
    const pending = [deferred<PreviewResult>(), deferred<PreviewResult>()]
    let call = 0
    const backend = createFakeBackend({ preview: () => pending[call++].promise })
    const onresult = vi.fn()
    const { update } = setup(backend, { onresult })

    await vi.advanceTimersByTimeAsync(250)
    await update({ values: { client: 'B' } })
    await vi.advanceTimersByTimeAsync(250)
    expect(backend.preview).toHaveBeenCalledTimes(2)

    pending[1].resolve(previewNamed('New'))
    await settle()
    pending[0].resolve(previewNamed('Old'))
    await settle()

    expect(treeItemNamed('New')).toBeDefined()
    expect(treeItemNamed('Old')).toBeUndefined()
    expect(onresult).toHaveBeenCalledOnce()
    expect(onresult).toHaveBeenCalledWith(expect.objectContaining({ rootName: 'New' }), null)
  })

  it('reports errors upward and shows them', async () => {
    const backend = createFakeBackend({ preview: () => Promise.reject(new Error('Target is not a folder')) })
    const onresult = vi.fn()
    setup(backend, { onresult })
    await vi.advanceTimersByTimeAsync(250)
    await settle()
    expect(onresult).toHaveBeenCalledWith(null, 'Target is not a folder')
    expect(screen.getByRole('alert')).toHaveTextContent('Target is not a folder')
  })

  it('renders a selected filled-in file and re-renders when values change', async () => {
    const backend = createFakeBackend()
    const { update } = setup(backend)
    await vi.advanceTimersByTimeAsync(250)
    await settle()

    const readme = treeItemNamed('readme.md')
    expect(readme).toBeDefined()
    await fireEvent.click(readme as HTMLElement)
    expect(backend.renderFile).toHaveBeenCalledExactlyOnceWith('/tpl', 'readme.md.ft$', { client: 'A' }, undefined)
    await settle()
    expect(screen.getByText('Acme/readme.md')).toBeInTheDocument()
    expect(screen.getByText('2 lines changed')).toBeInTheDocument()

    await update({ values: { client: 'Z' } })
    expect(backend.renderFile).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(250)
    expect(backend.renderFile).toHaveBeenLastCalledWith('/tpl', 'readme.md.ft$', { client: 'Z' }, undefined)

    await fireEvent.click(screen.getByRole('button', { name: 'Close file view' }))
    expect(screen.queryByText('Acme/readme.md')).not.toBeInTheDocument()
  })

  it('tracks answers edited in place inside a bound $state object', async () => {
    const backend = createFakeBackend()
    render(WithHost, {
      props: {
        backend,
        component: FormAndPreview,
        componentProps: { parameters: [{ ...blankParameter(), name: 'client', prompt: 'Client' }] },
      },
    })
    await vi.advanceTimersByTimeAsync(250)
    expect(backend.preview).toHaveBeenLastCalledWith('/tpl', { client: '' }, '/out', undefined)
    await fireEvent.input(screen.getByLabelText('Client'), { target: { value: 'Initech' } })
    await vi.advanceTimersByTimeAsync(250)
    expect(backend.preview).toHaveBeenLastCalledWith('/tpl', { client: 'Initech' }, '/out', undefined)
  })
})
