// Regenerates website/images/screenshots from the built app, fully headless.
//
//   node scripts/screenshots.mjs                 # all shots
//   node scripts/screenshots.mjs generate editor-try
//
// Nothing appears on the desktop and focus never moves: the app runs as
// `FolderTemplates.exe --serve-ui` (its UI served over loopback HTTP, the Go
// methods behind it), and headless Edge renders and drives the page over the
// DevTools protocol. Build first (scripts/build.ps1 or `wails build` in app/).
//
// A portable staged copy of the exe is used, so real settings and recents are
// never read or written. The sample templates are copied to a temp folder
// mapped to a spare drive letter (subst, per-user, removed afterwards), so
// the paths in the screenshots read T:\Templates\… and give nothing away
// about the machine or its user.

import { spawn, execFileSync } from 'node:child_process'
import { cpSync, existsSync, mkdirSync, mkdtempSync, readdirSync, rmSync, writeFileSync, copyFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const repo = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const outDir = join(repo, 'website', 'images', 'screenshots')
const only = process.argv.slice(2)

const edgePaths = [
  'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe',
  'C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe',
]
const edge = edgePaths.find(existsSync)
if (!edge) throw new Error('Microsoft Edge not found')

const VIEW = { width: 1180, height: 760, scale: 2 }
const UI_PORT = 7931
const CDP_PORT = 9333

// --- staging ---------------------------------------------------------------------
const libHost = mkdtempSync(join(tmpdir(), 'ftshots-lib-'))
const drive = ['T:', 'U:', 'V:', 'W:', 'X:', 'Y:'].find((d) => !existsSync(d + '/'))
if (!drive) throw new Error('no free drive letter for the screenshot library')
execFileSync('subst', [drive, libHost])
const lib = drive + '\\Templates'
const stage = mkdtempSync(join(tmpdir(), 'ftshots-'))
copyFileSync(join(repo, 'app', 'build', 'bin', 'FolderTemplates.exe'), join(stage, 'FolderTemplates.exe'))
writeFileSync(join(stage, 'portable'), 'screenshots')
mkdirSync(lib)
cpSync(join(repo, 'samples'), lib, { recursive: true })
const templateIn = (group) => {
  const dir = join(lib, group)
  return join(dir, readdirSync(dir).find((n) => !n.startsWith('.')))
}
const kitchen = templateIn('Kitchen Sink')
const video = templateIn('Video Episode')
const code = templateIn('Code Project')

function writeSettings(theme = 'dark') {
  const now = Date.now()
  const iso = (ms) => new Date(ms).toISOString()
  writeFileSync(join(stage, 'settings.json'), JSON.stringify({
    theme,
    libraryFolders: [lib],
    recents: [
      { path: video, lastUsed: iso(now - 3 * 3600e3) },
      { path: code, lastUsed: iso(now - 2 * 86400e3) },
    ],
    pinned: [kitchen],
    recentTargets: [],
    defaultConflict: 'refuse',
    openFolderAfter: false,
    closeAfter: false,
    showSamples: false,
  }, null, 2))
}

// --- tiny DevTools client -------------------------------------------------------------
const sleep = (ms) => new Promise((r) => setTimeout(r, ms))

async function waitForHttp(url, timeoutMs = 20000) {
  const end = Date.now() + timeoutMs
  while (Date.now() < end) {
    try {
      const res = await fetch(url)
      if (res.ok) return res
    } catch { /* not up yet */ }
    await sleep(200)
  }
  throw new Error(`timed out waiting for ${url}`)
}

async function connect() {
  const targets = await (await waitForHttp(`http://127.0.0.1:${CDP_PORT}/json/list`)).json()
  const page = targets.find((t) => t.type === 'page')
  const ws = new WebSocket(page.webSocketDebuggerUrl)
  await new Promise((res, rej) => { ws.onopen = res; ws.onerror = rej })
  let id = 0
  const pending = new Map()
  ws.onmessage = (ev) => {
    const msg = JSON.parse(ev.data)
    if (msg.id && pending.has(msg.id)) {
      const { resolve: ok, reject: fail } = pending.get(msg.id)
      pending.delete(msg.id)
      msg.error ? fail(new Error(msg.error.message)) : ok(msg.result)
    }
  }
  const send = (method, params = {}) => new Promise((ok, fail) => {
    id += 1
    pending.set(id, { resolve: ok, reject: fail })
    ws.send(JSON.stringify({ id, method, params }))
  })
  return { send, close: () => ws.close() }
}

// Page-side helpers, evaluated in the app page. Fields are found by their
// visible label, buttons and tabs by their text or accessible name.
const pageHelpers = `
  window.__ft = {
    field(label) {
      for (const l of document.querySelectorAll('label')) {
        if (l.textContent.trim() === label && l.htmlFor) return document.getElementById(l.htmlFor)
      }
      return document.querySelector('input[aria-label="' + label + '"], textarea[aria-label="' + label + '"]')
    },
    set(label, value) {
      const el = this.field(label)
      if (!el) return false
      const proto = el instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype
      Object.getOwnPropertyDescriptor(proto, 'value').set.call(el, value)
      el.dispatchEvent(new Event('input', { bubbles: true }))
      return true
    },
    click(text) {
      // Tabs first: a tab and a button can share a label (the editor's
      // "Files" tab vs each parameter's "Files" chip).
      const els = [...document.querySelectorAll('[role="tab"]'), ...document.querySelectorAll('button:not([role="tab"])')]
      for (const el of els) {
        const name = (el.getAttribute('aria-label') || el.textContent || '').trim()
        if (name === text) { el.click(); return true }
      }
      return false
    },
  }
`

async function evaluate(cdp, expression) {
  const r = await cdp.send('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true })
  return r.result?.value
}

async function retry(cdp, expression, what, timeoutMs = 8000) {
  const end = Date.now() + timeoutMs
  while (Date.now() < end) {
    if (await evaluate(cdp, expression)) return
    await sleep(200)
  }
  throw new Error(`${what} not found`)
}

// --- shots ----------------------------------------------------------------------------
const q = (p) => p // spawn passes args without shell quoting

const shots = [
  { name: 'library', args: [] },
  {
    name: 'generate', args: [q(kitchen)], steps: [
      ['set', 'Client name', 'Acme Studio'],
      ['set', 'Project name', 'Spring Campaign'],
      ['set', 'Project owner (optional)', 'Sam Rivera'],
    ],
  },
  { name: 'editor-scan', args: ['-edit', '-sourceFolder', q(kitchen)] },
  { name: 'editor-pattern', args: ['-edit', '-sourceFolder', q(kitchen)], steps: [['click', 'Test pattern']] },
  {
    name: 'editor-try', args: ['-edit', '-sourceFolder', q(video)], steps: [
      ['click', 'Try it'],
      ['set', 'Show', 'The Weekly Build'],
      ['set', 'Episode number', '042'],
      ['set', 'Episode title', 'Why the sky is blue'],
    ],
  },
  { name: 'settings', args: [], steps: [['click', 'Settings']] },
  { name: 'editor-files-light', theme: 'light', args: ['-edit', '-sourceFolder', q(kitchen)], steps: [['click', 'Files']] },
  {
    name: 'conflict', args: [q(kitchen)],
    before() {
      execFileSync(join(repo, 'app', 'build', 'bin', 'ft.exe'), [
        'generate', kitchen, '--target', join(dirname(kitchen), 'Generated'),
        '--set', 'client=Acme Studio', '--set', 'project=Spring Campaign', '--no-prompt',
      ])
    },
    steps: [
      ['set', 'Client name', 'Acme Studio'],
      ['set', 'Project name', 'Spring Campaign'],
    ],
  },
]

const profile = mkdtempSync(join(tmpdir(), 'ftshots-edge-'))
const browser = spawn(edge, [
  '--headless=new', '--disable-gpu', '--no-first-run', '--no-default-browser-check',
  `--user-data-dir=${profile}`, `--remote-debugging-port=${CDP_PORT}`,
  `--window-size=${VIEW.width},${VIEW.height}`, '--hide-scrollbars', 'about:blank',
], { stdio: 'ignore' })

let server
try {
  mkdirSync(outDir, { recursive: true })
  const cdp = await connect()
  await cdp.send('Page.enable')
  await cdp.send('Runtime.enable')
  await cdp.send('Emulation.setDeviceMetricsOverride', {
    width: VIEW.width, height: VIEW.height, deviceScaleFactor: VIEW.scale, mobile: false,
  })
  await cdp.send('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value: 'dark' }] })

  for (const shot of shots) {
    if (only.length && !only.includes(shot.name)) continue
    writeSettings(shot.theme)
    shot.before?.()
    server = spawn(join(stage, 'FolderTemplates.exe'), ['--serve-ui', `127.0.0.1:${UI_PORT}`, ...shot.args], { stdio: 'ignore' })
    await waitForHttp(`http://127.0.0.1:${UI_PORT}/`)
    await cdp.send('Page.navigate', { url: `http://127.0.0.1:${UI_PORT}/` })
    await sleep(1500)
    await evaluate(cdp, pageHelpers)
    for (const [kind, target, value] of shot.steps ?? []) {
      const expr = kind === 'set'
        ? `window.__ft.set(${JSON.stringify(target)}, ${JSON.stringify(value)})`
        : `window.__ft.click(${JSON.stringify(target)})`
      await retry(cdp, expr, `${kind} ${target}`)
      await sleep(600)
    }
    await evaluate(cdp, 'document.activeElement && document.activeElement.blur()')
    await sleep(1200) // debounced previews and scans settle
    const { data } = await cdp.send('Page.captureScreenshot', { format: 'png' })
    writeFileSync(join(outDir, `${shot.name}.png`), Buffer.from(data, 'base64'))
    console.log(`saved ${shot.name}`)
    server.kill()
    server = undefined
    await sleep(300)
  }
  cdp.close()
} finally {
  server?.kill()
  browser.kill()
  await sleep(500)
  try { execFileSync('subst', [drive, '/d']) } catch { /* already gone */ }
  rmSync(libHost, { recursive: true, force: true })
  rmSync(stage, { recursive: true, force: true })
  rmSync(profile, { recursive: true, force: true, maxRetries: 5, retryDelay: 300 })
}
