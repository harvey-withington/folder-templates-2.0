import { describe, expect, it, vi } from 'vitest'
import { baseName, buildTree, visibleNodes } from './tree'
import {
  defaultMatch,
  fromRows,
  humanize,
  initialValues,
  move,
  normalizeParameter,
  parameterFromToken,
  promptedParameters,
  sameDescriptor,
  toRows,
  blankParameter,
} from './params'
import { debounce, latestOnly } from './debounce'
import { createTranslator } from '../i18n'
import type { Parameter } from '../types'

const param = (over: Partial<Parameter>): Parameter => ({ ...blankParameter(), ...over })

describe('buildTree', () => {
  const items = [
    { p: 'root', d: true },
    { p: 'root/b.txt', d: false },
    { p: 'root/a', d: true },
    { p: 'root/a/z.md', d: false },
    { p: 'root/implied/deep.txt', d: false },
  ]
  const tree = buildTree(items, (i) => i.p, (i) => i.d)

  it('nests by path, folders first, and creates implied parents', () => {
    expect(tree).toHaveLength(1)
    const root = tree[0]
    expect(root.children.map((c) => c.name)).toEqual(['a', 'implied', 'b.txt'])
    const implied = root.children[1]
    expect(implied.isDir).toBe(true)
    expect(implied.item).toBeUndefined()
    expect(implied.children[0].depth).toBe(2)
  })

  it('hides children of collapsed folders', () => {
    const shown = visibleNodes(tree, new Set(['root/a'])).map((n) => n.path)
    expect(shown).toEqual(['root', 'root/a', 'root/implied', 'root/implied/deep.txt', 'root/b.txt'])
  })

  it('sorts numerically and case-insensitively', () => {
    const t = buildTree(['Ep10', 'ep2', 'Ep1'], (s) => s, () => false)
    expect(t.map((n) => n.name)).toEqual(['Ep1', 'ep2', 'Ep10'])
  })

  it('baseName', () => {
    expect(baseName('a/b/c.txt')).toBe('c.txt')
    expect(baseName('a/b/')).toBe('b')
    expect(baseName('solo')).toBe('solo')
  })
})

describe('params helpers', () => {
  const params = [
    param({ name: 'a', prompt: 'A?', defaultValue: 'x' }),
    param({ name: 'b', prompt: null, defaultValue: 'hidden' }),
    param({ name: '', prompt: 'nameless', match: '^_' }),
    param({ name: 'c', prompt: 'C?' }),
  ]

  it('asks only for named, prompted parameters', () => {
    expect(promptedParameters(params).map((p) => p.name)).toEqual(['a', 'c'])
  })

  it('fills defaults but keeps typed answers', () => {
    expect(initialValues(params)).toEqual({ a: 'x', c: '' })
    expect(initialValues(params, { c: 'typed', zzz: 'dropped' })).toEqual({ a: 'x', c: 'typed' })
  })

  it('escapes the default match like the engine', () => {
    expect(defaultMatch('name')).toBe('\\{name\\}')
    expect(defaultMatch('a.b')).toBe('\\{a\\.b\\}')
  })

  it('humanizes tokens', () => {
    expect(humanize('videoName')).toBe('Video name')
    expect(humanize('client_id')).toBe('Client id')
    expect(humanize('x')).toBe('X')
  })

  it('wires a parameter from a scanned token', () => {
    const p = parameterFromToken({ name: 'client', inNames: ['.'], inContent: [], declared: false })
    expect(p).toMatchObject({ name: 'client', prompt: 'Client', replaceInFileNames: true, replaceInFiles: false })
    const q = parameterFromToken({ name: 'body', inNames: [], inContent: ['a.ft$'], declared: false })
    expect(q).toMatchObject({ replaceInFileNames: false, replaceInFiles: true })
  })

  it('normalizes empty strings to null like the C# app', () => {
    expect(normalizeParameter(param({ name: ' n ', prompt: '', match: '', placeholder: '' }))).toMatchObject({
      name: 'n',
      prompt: null,
      match: null,
      placeholder: null,
    })
  })

  it('round-trips rows with unique ids', () => {
    const rows = toRows(params)
    expect(new Set(rows.map((r) => r.id)).size).toBe(rows.length)
    expect(fromRows(rows).map((p) => p.name)).toEqual(['a', 'b', '', 'c'])
  })

  it('moves items', () => {
    expect(move([1, 2, 3, 4], 0, 2)).toEqual([2, 3, 1, 4])
    expect(move([1, 2, 3, 4], 3, 0)).toEqual([4, 1, 2, 3])
    expect(move([1, 2], 5, 0)).toEqual([1, 2])
  })

  it('compares descriptors by saved form', () => {
    const d = { name: 'n', description: '', defaultTargetPath: '', parameters: [param({ name: 'a', prompt: '' })] }
    const e = { ...d, parameters: [param({ name: 'a', prompt: null })] }
    expect(sameDescriptor(d, e)).toBe(true)
    expect(sameDescriptor(d, { ...d, name: 'm' })).toBe(false)
  })
})

describe('debounce', () => {
  it('runs only the last call', () => {
    vi.useFakeTimers()
    const fn = vi.fn()
    const d = debounce(fn, 100)
    d(1)
    d(2)
    vi.advanceTimersByTime(99)
    expect(fn).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1)
    expect(fn).toHaveBeenCalledExactlyOnceWith(2)
    d(3)
    d.cancel()
    vi.advanceTimersByTime(200)
    expect(fn).toHaveBeenCalledTimes(1)
    d(4)
    d.flush()
    expect(fn).toHaveBeenLastCalledWith(4)
    vi.useRealTimers()
  })

  it('latestOnly', () => {
    const l = latestOnly()
    const a = l.next()
    const b = l.next()
    expect(l.isLatest(a)).toBe(false)
    expect(l.isLatest(b)).toBe(true)
  })
})

describe('createTranslator', () => {
  const t = createTranslator({ ft: { hello: 'Hello {name}, {n} files', plain: 'Plain' } })
  it('looks up dotted keys and interpolates', () => {
    expect(t('ft.hello', { name: 'Ann', n: 3 })).toBe('Hello Ann, 3 files')
    expect(t('ft.plain')).toBe('Plain')
  })
  it('returns the key when missing and leaves unknown placeholders', () => {
    expect(t('ft.nope')).toBe('ft.nope')
    expect(t('ft.hello', { name: 'Ann' })).toBe('Hello Ann, {n} files')
  })
})
