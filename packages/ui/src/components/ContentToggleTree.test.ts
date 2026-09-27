import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/svelte'
import WithHost from '../testing/WithHost.svelte'
import ContentToggleTree from './ContentToggleTree.svelte'
import { createFakeBackend, deferred, sampleTree, settle, type FakeBackend } from '../testing/fakeBackend'
import type { TreeEntry } from '../types'

function setup(backend: FakeBackend, onchanged = vi.fn(), toast = vi.fn()) {
  render(WithHost, {
    props: { backend, component: ContentToggleTree, toast, componentProps: { dir: '/tpl', onchanged } },
  })
  return { onchanged, toast }
}

describe('ContentToggleTree', () => {
  it('shows files with switches reflecting .ft$ processing', async () => {
    const backend = createFakeBackend()
    setup(backend)
    expect(backend.tree).toHaveBeenCalledWith('/tpl')
    await settle()

    const brief = screen.getByRole('switch', { name: 'Fill in tokens in brief.md' })
    expect(brief).toHaveAttribute('aria-checked', 'true')
    expect(screen.getByRole('switch', { name: 'Fill in tokens in notes.txt' })).toHaveAttribute('aria-checked', 'false')
    // Folders have no switch; the suffix is shown as a badge, not in the name.
    expect(screen.getAllByRole('switch')).toHaveLength(2)
    expect(screen.getByText('brief.md')).toBeInTheDocument()
    expect(screen.getByText('.ft$')).toBeInTheDocument()
  })

  it('toggles processing, reloads the tree and notifies', async () => {
    let entries: TreeEntry[] = sampleTree()
    const backend = createFakeBackend({
      tree: async () => entries,
      setContentProcessing: async (_dir, rel, on) => {
        entries = entries.map((e) => (e.rel === rel ? { ...e, rel: rel + '.ft$', processed: on } : e))
        return rel + '.ft$'
      },
    })
    const { onchanged } = setup(backend)
    await settle()

    await fireEvent.click(screen.getByRole('switch', { name: 'Fill in tokens in notes.txt' }))
    await settle()
    expect(backend.setContentProcessing).toHaveBeenCalledWith('/tpl', 'docs/notes.txt', true)
    expect(backend.tree).toHaveBeenCalledTimes(2)
    expect(onchanged).toHaveBeenCalledOnce()
    expect(screen.getByRole('switch', { name: 'Fill in tokens in notes.txt' })).toHaveAttribute('aria-checked', 'true')
  })

  it('disables a switch while its request is in flight', async () => {
    const gate = deferred<string>()
    const backend = createFakeBackend({ setContentProcessing: () => gate.promise })
    setup(backend)
    await settle()
    const notes = screen.getByRole('switch', { name: 'Fill in tokens in notes.txt' })
    await fireEvent.click(notes)
    expect(notes).toBeDisabled()
    expect(screen.getByRole('switch', { name: 'Fill in tokens in brief.md' })).toBeEnabled()
    gate.resolve('docs/notes.txt.ft$')
    await settle()
    expect(screen.getByRole('switch', { name: 'Fill in tokens in notes.txt' })).toBeEnabled()
  })

  it('reports failures through toast', async () => {
    const backend = createFakeBackend({ setContentProcessing: () => Promise.reject(new Error('File is read-only')) })
    const { toast, onchanged } = setup(backend)
    await settle()
    await fireEvent.click(screen.getByRole('switch', { name: 'Fill in tokens in notes.txt' }))
    await settle()
    expect(toast).toHaveBeenCalledWith('File is read-only', 'error')
    expect(onchanged).not.toHaveBeenCalled()
  })
})
