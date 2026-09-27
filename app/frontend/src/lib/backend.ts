import type { TemplateBackend } from '@harvey-withington/folder-templates-ui'
import { app } from './bridge'

/**
 * The shared components' TemplateBackend over the Wails bindings. Go returns
 * null for empty slices in a few places; normalize to [] here so components
 * never see null.
 */
export function wailsBackend(): TemplateBackend {
  return {
    inspect: (dir) => app().Inspect(dir),
    validate: async (descriptor) => (await app().Validate(descriptor)) ?? [],
    save: (dir, descriptor) => app().SaveTemplate(dir, descriptor),
    preview: (dir, values, target, descriptor) => app().Preview(dir, descriptor ?? null, values, target),
    renderFile: (dir, sourceRel, values, descriptor) => app().RenderFile(dir, descriptor ?? null, sourceRel, values),
    tree: async (dir) => (await app().Tree(dir)) ?? [],
    scan: (dir, descriptor) => app().Scan(dir, descriptor ?? null),
    testMatch: async (dir, pattern, replacement) => (await app().TestMatch(dir, pattern, replacement)) ?? [],
    setContentProcessing: (dir, rel, on) => app().SetContentProcessing(dir, rel, on),
    pickFolder: async (title, initial) => {
      const picked = await app().PickFolder(title, initial ?? '')
      return picked === '' ? null : picked
    },
  }
}
