import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/svelte'
import PreviewTree from './PreviewTree.svelte'
import { samplePreview } from '../testing/fakeBackend'

const item = (name: string) => {
  const el = screen.getAllByRole('treeitem').find((n) => n.querySelector('.name')?.textContent?.trim() === name)
  if (!el) throw new Error(`no row ${name}`)
  return el
}

describe('PreviewTree', () => {
  it('summarises folders, files and existing items', () => {
    render(PreviewTree, { props: { result: samplePreview() } })
    expect(screen.getByText('2 folders')).toBeInTheDocument()
    expect(screen.getByText('3 files')).toBeInTheDocument()
    expect(screen.getByText('1 already exist')).toBeInTheDocument()
  })

  it('omits the existing count when nothing exists and uses singular forms', () => {
    const result = samplePreview()
    result.entries = result.entries.slice(0, 3)
    render(PreviewTree, { props: { result } })
    expect(screen.getByText('1 file')).toBeInTheDocument()
    expect(screen.queryByText(/already exist/)).not.toBeInTheDocument()
  })

  it('marks renamed items, filled-in files and existing items', () => {
    render(PreviewTree, { props: { result: samplePreview() } })
    const brief = item('Acme brief.md')
    expect(brief.querySelector('.name')).toHaveClass('renamed')
    expect(brief.querySelector('.name')).toHaveAttribute('title', 'Renamed from {client} brief.md.ft$')
    expect(brief).toHaveTextContent('filled in')

    // .ft$ removal alone is not a rename.
    const readme = item('readme.md')
    expect(readme.querySelector('.name')).not.toHaveClass('renamed')
    expect(readme).toHaveTextContent('filled in')

    const notes = item('notes.txt')
    expect(notes.querySelector('.name')).not.toHaveClass('renamed')
    expect(notes).not.toHaveTextContent('filled in')
    expect(notes.querySelector('[title="Already exists at the target"]')).not.toBeNull()

    // The root has no template-side name to compare with.
    expect(item('Acme').querySelector('.name')).not.toHaveClass('renamed')
  })

  it('selects filled-in files only', async () => {
    const onselect = vi.fn()
    render(PreviewTree, { props: { result: samplePreview(), onselect } })
    await fireEvent.click(item('notes.txt'))
    expect(onselect).not.toHaveBeenCalled()
    await fireEvent.click(item('readme.md'))
    expect(onselect).toHaveBeenCalledWith(expect.objectContaining({ sourceRel: 'readme.md.ft$' }))
    expect(item('readme.md')).toHaveAttribute('aria-selected', 'true')
  })

  it('shows the selected sourceRel passed in', () => {
    render(PreviewTree, { props: { result: samplePreview(), selected: 'docs/{client} brief.md.ft$' } })
    expect(item('Acme brief.md')).toHaveAttribute('aria-selected', 'true')
  })

  it('keeps the previous tree while loading', () => {
    render(PreviewTree, { props: { result: samplePreview(), loading: true } })
    expect(screen.getByRole('tree')).toBeInTheDocument()
    expect(screen.getByText('Updating…')).toBeInTheDocument()
  })

  it('shows errors, warnings and the empty state', async () => {
    const { rerender } = render(PreviewTree, { props: { result: null } })
    expect(screen.getByText(/Fill in the form/)).toBeInTheDocument()

    await rerender({ result: null, error: 'Parameter "client" is required' })
    expect(screen.getByRole('alert')).toHaveTextContent('Parameter "client" is required')
    expect(screen.queryByRole('tree')).not.toBeInTheDocument()

    await rerender({ result: { ...samplePreview(), warnings: ['Two items map to the same name'] }, error: null })
    expect(screen.getByText('Warnings')).toBeInTheDocument()
    expect(screen.getByText('Two items map to the same name')).toBeInTheDocument()
  })
})
