import { vi, type Mock } from 'vitest'
import type {
  Inspection,
  NameHit,
  PreviewResult,
  RenderedFile,
  ScanResult,
  TemplateBackend,
  TemplateDescriptor,
  TreeEntry,
} from '../types'
import { blankParameter } from '../lib/params'

export type FakeBackend = { [K in keyof TemplateBackend]: Mock<TemplateBackend[K]> }

export const sampleDescriptor = (): TemplateDescriptor => ({
  name: 'Client project',
  description: 'A folder per client',
  defaultTargetPath: '../Clients',
  parameters: [
    { ...blankParameter(), name: 'client', prompt: 'Client name', placeholder: 'Acme' },
    { ...blankParameter(), name: 'year', prompt: null, defaultValue: '2026' },
  ],
})

export const samplePreview = (): PreviewResult => ({
  rootName: 'Acme',
  rootPath: '/out/Acme',
  rootExists: false,
  entries: [
    { sourceRel: '', outputRel: 'Acme', isDir: true, processed: false },
    { sourceRel: 'docs', outputRel: 'Acme/docs', isDir: true, processed: false },
    { sourceRel: 'docs/{client} brief.md.ft$', outputRel: 'Acme/docs/Acme brief.md', isDir: false, processed: true },
    { sourceRel: 'docs/notes.txt', outputRel: 'Acme/docs/notes.txt', isDir: false, processed: false, exists: true },
    { sourceRel: 'readme.md.ft$', outputRel: 'Acme/readme.md', isDir: false, processed: true },
  ],
  warnings: [],
})

export const sampleRendered = (): RenderedFile => ({
  before: '# {{$client}}\nplain line\nfor {{$client}}\n',
  after: '# Acme\nplain line\nfor Acme\n',
})

export const sampleTree = (): TreeEntry[] => [
  { rel: 'docs', isDir: true, size: 0, processed: false },
  { rel: 'docs/brief.md.ft$', isDir: false, size: 10, processed: true },
  { rel: 'docs/notes.txt', isDir: false, size: 5, processed: false },
]

export const sampleScan = (): ScanResult => ({
  tokens: [
    { name: 'client', inNames: ['.', 'docs/{client} brief.md.ft$'], inContent: ['readme.md.ft$'], declared: true },
    { name: 'projectCode', inNames: ['{projectCode}'], inContent: [], declared: false },
  ],
  params: [{ name: 'client', match: '\\{client\\}', nameHits: 2, contentHits: 1 }],
  issues: [
    { kind: 'undeclared-token', token: 'projectCode', message: 'Token {projectCode} has no parameter' },
    { kind: 'unused-parameter', param: 'year', message: 'Parameter year is never used' },
  ],
})

export const sampleHits = (): NameHit[] => [
  { sourceRel: '.', name: '{client}', result: 'Acme', isDir: true },
  { sourceRel: 'docs/{client} brief.md.ft$', name: '{client} brief.md.ft$', result: 'Acme brief.md.ft$', isDir: false },
]

const sampleInspection = (): Inspection => ({
  dir: '/templates/client',
  folderName: 'client',
  isTemplate: true,
  template: sampleDescriptor(),
  issues: [],
  defaultTarget: '/Clients',
  files: 3,
  dirs: 1,
  sizeBytes: 15,
})

/** A TemplateBackend of vi.fn() mocks with canned answers; pass overrides to replace any of them. */
export function createFakeBackend(overrides: Partial<TemplateBackend> = {}): FakeBackend {
  const defaults: TemplateBackend = {
    inspect: async () => sampleInspection(),
    validate: async () => [],
    save: async () => {},
    preview: async () => samplePreview(),
    renderFile: async () => sampleRendered(),
    tree: async () => sampleTree(),
    scan: async () => sampleScan(),
    testMatch: async () => sampleHits(),
    setContentProcessing: async (_dir, rel, on) => (on ? rel + '.ft$' : rel.replace(/\.ft\$$/, '')),
    pickFolder: async () => null,
  }
  const impl = { ...defaults, ...overrides }
  return {
    inspect: vi.fn(impl.inspect),
    validate: vi.fn(impl.validate),
    save: vi.fn(impl.save),
    preview: vi.fn(impl.preview),
    renderFile: vi.fn(impl.renderFile),
    tree: vi.fn(impl.tree),
    scan: vi.fn(impl.scan),
    testMatch: vi.fn(impl.testMatch),
    setContentProcessing: vi.fn(impl.setContentProcessing),
    pickFolder: vi.fn(impl.pickFolder),
  }
}

/** A promise with its resolve/reject exposed, for ordering async responses in tests. */
export function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

/** Lets pending promise callbacks and Svelte updates run. */
export async function settle(): Promise<void> {
  for (let i = 0; i < 10; i++) await Promise.resolve()
}
