<script lang="ts">
  import { FolderTree, LayoutTemplate, PencilRuler, Settings } from 'lucide-svelte'
  import { t } from '../lib/i18n'
  import { nav, type View } from '../lib/nav.svelte'
  import icon from '../assets/icon.png'

  interface Item {
    view: View
    label: string
    icon: typeof LayoutTemplate
    needsDir: boolean
  }

  const items: Item[] = [
    { view: 'library', label: t('app.rail.library'), icon: LayoutTemplate, needsDir: false },
    { view: 'generate', label: t('app.rail.generate'), icon: FolderTree, needsDir: true },
    { view: 'editor', label: t('app.rail.editor'), icon: PencilRuler, needsDir: true },
  ]
</script>

<nav class="rail" aria-label={t('app.rail.label')}>
  <div class="brand" title={t('app.name')}>
    <img src={icon} alt="" width="30" height="30" />
  </div>
  {#each items as item (item.view)}
    {@const Icon = item.icon}
    <button
      class="item"
      class:active={nav.view === item.view}
      disabled={item.needsDir && !nav.dir}
      aria-current={nav.view === item.view ? 'page' : undefined}
      title={item.label}
      onclick={() => nav.go({ view: item.view })}
    >
      <Icon size={19} />
      <span>{item.label}</span>
    </button>
  {/each}
  <div class="spacer"></div>
  <button
    class="item"
    class:active={nav.view === 'settings'}
    aria-current={nav.view === 'settings' ? 'page' : undefined}
    title={t('app.rail.settings')}
    onclick={() => nav.go({ view: 'settings' })}
  >
    <Settings size={19} />
    <span>{t('app.rail.settings')}</span>
  </button>
</nav>

<style>
  .rail {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    width: 72px;
    padding: 12px 6px;
    background: var(--bg-base);
  }

  .brand {
    display: grid;
    place-items: center;
    height: 40px;
    margin-bottom: 10px;
  }

  .item {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 3px;
    width: 60px;
    padding: 8px 2px 6px;
    border: none;
    border-radius: 8px;
    background: transparent;
    color: var(--text-secondary);
    font: inherit;
    font-size: 11px;
    cursor: pointer;
    transition:
      background var(--duration-normal),
      color var(--duration-normal);
  }

  .item:hover:not(:disabled) {
    background: var(--bg-subtle-hover);
    color: var(--text-primary);
  }

  .item.active {
    background: var(--accent-glow-3);
    color: var(--accent-light, var(--accent));
  }

  .item:disabled {
    opacity: 0.35;
    cursor: default;
  }

  .spacer {
    flex: 1;
  }
</style>
