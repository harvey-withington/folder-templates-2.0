import { createTranslator, en as ftEn, type Messages } from '@harvey-withington/folder-templates-ui'
import appEn from '../locales/en.json'

/** App strings live under "app.*", the shared components' under "ft.*". */
export const t = createTranslator({ ...(ftEn as Messages), ...(appEn as Messages) })
