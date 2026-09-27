import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/svelte'
import WithHost from '../testing/WithHost.svelte'
import RegexTester from './RegexTester.svelte'
import { createFakeBackend, settle, type FakeBackend } from '../testing/fakeBackend'

function setup(backend: FakeBackend, pattern = '') {
  render(WithHost, { props: { backend, component: RegexTester, componentProps: { dir: '/tpl', pattern } } })
}

describe('RegexTester', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('waits for a pattern', async () => {
    const backend = createFakeBackend()
    setup(backend)
    await vi.advanceTimersByTimeAsync(500)
    expect(backend.testMatch).not.toHaveBeenCalled()
    expect(screen.getByText(/Enter a pattern/)).toBeInTheDocument()
  })

  it('lists hits as name → result, debounced', async () => {
    const backend = createFakeBackend()
    setup(backend)
    await fireEvent.input(screen.getByLabelText('Pattern'), { target: { value: '\\{cl' } })
    await fireEvent.input(screen.getByLabelText('Pattern'), { target: { value: '\\{client\\}' } })
    await fireEvent.input(screen.getByLabelText('Value to test with'), { target: { value: 'Acme' } })
    await vi.advanceTimersByTimeAsync(250)
    await settle()
    expect(backend.testMatch).toHaveBeenCalledExactlyOnceWith('/tpl', '\\{client\\}', 'Acme')
    expect(screen.getByText('2 names match')).toBeInTheDocument()
    expect(screen.getByText('{client}')).toBeInTheDocument()
    expect(screen.getByText('Acme')).toBeInTheDocument()
    expect(screen.getByText('Acme brief.md.ft$')).toBeInTheDocument()
    expect(screen.getByText('{client}').closest('li')).toHaveAttribute('title', '(template folder)')
  })

  it('says when nothing matches', async () => {
    setup(createFakeBackend({ testMatch: async () => [] }), 'zzz')
    await vi.advanceTimersByTimeAsync(250)
    await settle()
    expect(screen.getByText('No names match this pattern.')).toBeInTheDocument()
  })

  it('shows invalid-pattern and timeout errors', async () => {
    setup(createFakeBackend({ testMatch: () => Promise.reject(new Error('parsing "(" - Not enough )\'s')) }), '(')
    await vi.advanceTimersByTimeAsync(250)
    await settle()
    const alert = screen.getByRole('alert')
    expect(alert).toHaveTextContent("Pattern can't be used")
    expect(alert).toHaveTextContent('Not enough')
  })

  it('clears results when the pattern is emptied', async () => {
    setup(createFakeBackend(), 'x')
    await vi.advanceTimersByTimeAsync(250)
    await settle()
    expect(screen.getByText('2 names match')).toBeInTheDocument()
    await fireEvent.input(screen.getByLabelText('Pattern'), { target: { value: '' } })
    expect(screen.queryByText('2 names match')).not.toBeInTheDocument()
    expect(screen.getByText(/Enter a pattern/)).toBeInTheDocument()
  })
})
