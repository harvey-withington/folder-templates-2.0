import { describe, expect, it, vi } from 'vitest'
import { createRawSnippet } from 'svelte'
import { fireEvent, render, screen, within } from '@testing-library/svelte'
import userEvent from '@testing-library/user-event'
import TreeView from './TreeView.svelte'
import { buildTree, type TreeNode } from '../lib/tree'

const nodes = buildTree(['root', 'root/a', 'root/a/one.txt', 'root/b.txt'], (p) => p, (p) => p === 'root' || p === 'root/a')

const row = createRawSnippet((node: () => TreeNode<string>) => ({
  render: () => `<span>${node().name}</span>`,
}))

const names = () => screen.getAllByRole('treeitem').map((el) => el.textContent?.trim())

describe('TreeView', () => {
  it('renders an accessible tree, folders expanded by default', () => {
    render(TreeView<string>, { props: { nodes, row, label: 'Files' } })
    expect(screen.getByRole('tree', { name: 'Files' })).toBeInTheDocument()
    expect(names()).toEqual(['root', 'a', 'one.txt', 'b.txt'])
    const [root, a, file] = screen.getAllByRole('treeitem')
    expect(root).toHaveAttribute('aria-expanded', 'true')
    expect(root).toHaveAttribute('aria-level', '1')
    expect(a).toHaveAttribute('aria-level', '2')
    expect(file).not.toHaveAttribute('aria-expanded')
    expect(root).toHaveAttribute('tabindex', '0')
    expect(a).toHaveAttribute('tabindex', '-1')
  })

  it('collapses and expands a folder from its chevron', async () => {
    const onselect = vi.fn()
    render(TreeView<string>, { props: { nodes, row, label: 'Files', onselect } })
    const a = screen.getAllByRole('treeitem')[1]
    const chevron = a.querySelector('[data-chevron]')
    expect(chevron).not.toBeNull()
    await fireEvent.click(chevron as Element)
    expect(names()).toEqual(['root', 'a', 'b.txt'])
    expect(screen.getAllByRole('treeitem')[1]).toHaveAttribute('aria-expanded', 'false')
    expect(onselect).not.toHaveBeenCalled()
    await fireEvent.click(screen.getAllByRole('treeitem')[1].querySelector('[data-chevron]') as Element)
    expect(names()).toHaveLength(4)
  })

  it('selects on click and marks aria-selected', async () => {
    const onselect = vi.fn()
    render(TreeView<string>, { props: { nodes, row, label: 'Files', onselect } })
    await fireEvent.click(within(screen.getByRole('tree')).getByText('b.txt'))
    expect(onselect).toHaveBeenCalledWith(expect.objectContaining({ path: 'root/b.txt' }))
    expect(screen.getAllByRole('treeitem')[3]).toHaveAttribute('aria-selected', 'true')
  })

  it('skips nodes that are not selectable', async () => {
    const onselect = vi.fn()
    render(TreeView<string>, { props: { nodes, row, label: 'Files', onselect, selectable: (n) => !n.isDir } })
    await fireEvent.click(screen.getByText('a'))
    expect(onselect).not.toHaveBeenCalled()
  })

  it('is keyboard operable', async () => {
    const user = userEvent.setup()
    const onselect = vi.fn()
    render(TreeView<string>, { props: { nodes, row, label: 'Files', onselect } })
    await user.tab()
    expect(screen.getAllByRole('treeitem')[0]).toHaveFocus()
    await user.keyboard('{ArrowDown}')
    expect(screen.getAllByRole('treeitem')[1]).toHaveFocus()
    await user.keyboard('{ArrowLeft}') // collapse "a"
    expect(names()).toEqual(['root', 'a', 'b.txt'])
    await user.keyboard('{ArrowRight}{ArrowDown}{ArrowDown}{Enter}')
    expect(onselect).toHaveBeenLastCalledWith(expect.objectContaining({ path: 'root/b.txt' }))
    await user.keyboard('{Home}')
    expect(screen.getAllByRole('treeitem')[0]).toHaveFocus()
  })

  it('has no chevrons or aria-expanded when not collapsible', () => {
    render(TreeView<string>, { props: { nodes, row, label: 'Files', collapsible: false } })
    expect(document.querySelector('[data-chevron]')).toBeNull()
    expect(screen.getAllByRole('treeitem')[0]).not.toHaveAttribute('aria-expanded')
  })
})
