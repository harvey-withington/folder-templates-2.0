import { describe, expect, it } from 'vitest'
import { changedLines, diffLines, splitLines } from './diff'
import { cloneJson, errorMessage } from './errors'
import { stripFtSuffix } from './tree'
import { blankParameter, blankRow, fromRows, toRows } from './params'
import { plural } from './host'
import { createTranslator } from '../i18n'

describe('diff', () => {
  it('splits lines on any line break and ignores a trailing one', () => {
    expect(splitLines('a\r\nb\nc\rd\n')).toEqual(['a', 'b', 'c', 'd'])
    expect(splitLines('')).toEqual([])
    expect(splitLines('\n')).toEqual([''])
  })

  it('pairs lines by index and flags changes', () => {
    const lines = diffLines('# {{$x}}\nsame\n{{$x}}!', '# Acme\nsame\nAcme!')
    expect(lines.map((l) => l.changed)).toEqual([true, false, true])
    expect(lines[0]).toEqual({ number: 1, before: '# {{$x}}', after: '# Acme', changed: true })
    expect(changedLines(lines).map((l) => l.number)).toEqual([1, 3])
  })

  it('pads a length mismatch and marks the extra lines changed', () => {
    const lines = diffLines('a\nb', 'a')
    expect(lines).toHaveLength(2)
    expect(lines[1]).toMatchObject({ before: 'b', after: '', changed: true })
    expect(diffLines('a\n\n', 'a\n')[1]).toMatchObject({ before: '', after: '', changed: true })
  })
})

describe('errors', () => {
  it('extracts a message from anything', () => {
    expect(errorMessage(new Error('boom'))).toBe('boom')
    expect(errorMessage('plain')).toBe('plain')
    expect(errorMessage({ message: 'wails' })).toBe('wails')
    expect(errorMessage({ code: 3 })).toBe('{"code":3}')
    expect(errorMessage(undefined)).toBe('undefined')
  })

  it('deep-copies JSON data', () => {
    const src = { a: { b: [1, 2] } }
    const copy = cloneJson(src)
    expect(copy).toEqual(src)
    expect(copy.a).not.toBe(src.a)
    expect(cloneJson(undefined)).toBeUndefined()
  })
})

describe('misc helpers', () => {
  it('strips the .ft$ suffix only at the end', () => {
    expect(stripFtSuffix('a.md.ft$')).toBe('a.md')
    expect(stripFtSuffix('a.ft$.md')).toBe('a.ft$.md')
  })

  it('keeps internal apart from an empty prompt', () => {
    const row = blankRow()
    expect(row.internal).toBe(false)
    expect(toRows([{ ...blankParameter(), name: 'x', prompt: null }])[0].internal).toBe(true)
    const internal = { ...blankRow(), internal: true }
    internal.param.prompt = 'kept in the row'
    expect(fromRows([internal])[0].prompt).toBeNull()
  })

  it('pluralises by count', () => {
    const t = createTranslator({ n: { one: 'one thing', other: '{count} things' } })
    expect(plural(t, 'n', 1)).toBe('one thing')
    expect(plural(t, 'n', 0)).toBe('0 things')
  })
})
