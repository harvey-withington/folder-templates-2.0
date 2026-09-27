import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/svelte'
import FileDiffView from './FileDiffView.svelte'
import { sampleRendered } from '../testing/fakeBackend'

const lines = () => [...document.querySelectorAll('.line')]

describe('FileDiffView', () => {
  it('shows the result with changed lines highlighted', () => {
    render(FileDiffView, { props: { path: 'Acme/readme.md', rendered: sampleRendered() } })
    expect(screen.getByText('Acme/readme.md')).toBeInTheDocument()
    expect(screen.getByText('2 lines changed')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Result' })).toHaveAttribute('aria-pressed', 'true')
    expect(lines().map((l) => l.querySelector('.text')?.textContent)).toEqual(['# Acme', 'plain line', 'for Acme'])
    expect(lines().map((l) => l.classList.contains('add'))).toEqual([true, false, true])
    expect(lines()[2].querySelector('.ln')).toHaveTextContent('3')
  })

  it('switches to changes: each changed line as before and after', async () => {
    render(FileDiffView, { props: { path: 'readme.md', rendered: sampleRendered() } })
    await fireEvent.click(screen.getByRole('button', { name: 'Changes' }))
    expect(screen.getByRole('button', { name: 'Changes' })).toHaveAttribute('aria-pressed', 'true')
    const shown = lines().map((l) => [l.classList.contains('remove') ? '-' : '+', l.querySelector('.text')?.textContent])
    expect(shown).toEqual([
      ['-', '# {{$client}}'],
      ['+', '# Acme'],
      ['-', 'for {{$client}}'],
      ['+', 'for Acme'],
    ])
  })

  it('says when nothing changes', () => {
    render(FileDiffView, { props: { path: 'x', rendered: { before: 'a\nb', after: 'a\nb' }, mode: 'changes' } })
    expect(screen.getByText(/Nothing in this file is replaced/)).toBeInTheDocument()
  })

  it('shows loading, error and empty states', async () => {
    const { rerender } = render(FileDiffView, { props: { path: 'x', rendered: null, loading: true } })
    expect(screen.getAllByText('Rendering…').length).toBeGreaterThan(0)
    await rerender({ path: 'x', rendered: null, loading: false })
    expect(screen.getByText(/Select a filled-in file/)).toBeInTheDocument()
    await rerender({ path: 'x', rendered: null, error: 'File is binary' })
    expect(screen.getByRole('alert')).toHaveTextContent('File is binary')
  })

  it('offers a close button only when onclose is given', async () => {
    const onclose = vi.fn()
    const { rerender } = render(FileDiffView, { props: { path: 'x', rendered: sampleRendered() } })
    expect(screen.queryByRole('button', { name: 'Close file view' })).not.toBeInTheDocument()
    await rerender({ path: 'x', rendered: sampleRendered(), onclose })
    await fireEvent.click(screen.getByRole('button', { name: 'Close file view' }))
    expect(onclose).toHaveBeenCalledOnce()
  })
})
