import { describe, expect, it } from 'vitest'
import { matchesQuery, relativeTime } from './format'
import { nav } from './nav.svelte'
import { confirmer, toasts } from './ui.svelte'
import { errorText } from './bridge'
import { t } from './i18n'

describe('format', () => {
  it('relativeTime', () => {
    const now = new Date('2026-09-27T12:00:00Z')
    expect(relativeTime('2026-09-27T11:59:50Z', now)).toMatch(/now|this minute/i)
    expect(relativeTime('2026-09-24T12:00:00Z', now)).toMatch(/3 days ago/)
    expect(relativeTime('not a date', now)).toBe('')
  })

  it('matchesQuery', () => {
    expect(matchesQuery('', 'x')).toBe(true)
    expect(matchesQuery(' kit ', 'Kitchen Sink', undefined)).toBe(true)
    expect(matchesQuery('zzz', 'Kitchen Sink', 'desc')).toBe(false)
  })
})

describe('nav', () => {
  it('honours a leave guard', async () => {
    await nav.go({ view: 'library' })
    let allow = false
    nav.setGuard(async () => allow)
    expect(await nav.go({ view: 'settings' })).toBe(false)
    expect(nav.view).toBe('library')
    allow = true
    expect(await nav.go({ view: 'generate', dir: 'C:/t' })).toBe(true)
    expect(nav.view).toBe('generate')
    expect(nav.dir).toBe('C:/t')
    // the guard is dropped after a successful navigation
    expect(await nav.go({ view: 'library' })).toBe(true)
  })
})

describe('confirmer', () => {
  it('resolves with the answer, and a new question cancels the old one', async () => {
    const first = confirmer.ask('first?')
    const second = confirmer.ask('second?', { danger: true })
    expect(await first).toBe(false)
    expect(confirmer.current?.message).toBe('second?')
    confirmer.answer(true)
    expect(await second).toBe(true)
    expect(confirmer.current).toBeNull()
  })
})

describe('toasts', () => {
  it('adds and dismisses', () => {
    toasts.show('hi', 'success')
    const id = toasts.items.at(-1)?.id ?? -1
    expect(toasts.items.some((x) => x.message === 'hi')).toBe(true)
    toasts.dismiss(id)
    expect(toasts.items.some((x) => x.id === id)).toBe(false)
  })
})

describe('bridge & i18n', () => {
  it('errorText', () => {
    expect(errorText('go error')).toBe('go error')
    expect(errorText(new Error('js error'))).toBe('js error')
  })

  it('app and component strings share one translator', () => {
    expect(t('app.generate.create', { name: 'X' })).toBe('Create X')
    expect(t('app.library.title')).toBe('Templates')
  })
})
