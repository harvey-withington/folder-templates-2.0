<script lang="ts">
  import { CircleAlert, CircleCheck, Info, X } from 'lucide-svelte'
  import { t } from '../lib/i18n'
  import { toasts } from '../lib/ui.svelte'
</script>

<div class="toasts" role="status" aria-live="polite">
  {#each toasts.items as toast (toast.id)}
    <div class="toast {toast.kind}">
      {#if toast.kind === 'error'}
        <CircleAlert size={16} />
      {:else if toast.kind === 'success'}
        <CircleCheck size={16} />
      {:else}
        <Info size={16} />
      {/if}
      <span class="message selectable">{toast.message}</span>
      <button class="close" aria-label={t('app.common.dismiss')} onclick={() => toasts.dismiss(toast.id)}>
        <X size={14} />
      </button>
    </div>
  {/each}
</div>

<style>
  .toasts {
    position: fixed;
    right: 16px;
    bottom: 16px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-width: min(440px, calc(100vw - 32px));
    z-index: 60;
  }

  .toast {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 10px 12px;
    border: 1px solid var(--border);
    border-radius: 8px;
    background: var(--bg-elevated);
    color: var(--text-primary);
    box-shadow: 0 8px 24px var(--shadow);
  }

  .toast.error {
    border-color: var(--danger);
    background: var(--danger-bg);
    color: var(--danger-text);
  }

  .toast.success {
    border-color: var(--success);
    background: var(--success-bg);
    color: var(--success-text);
  }

  .message {
    flex: 1;
    overflow-wrap: anywhere;
  }

  .close {
    padding: 2px;
    border: none;
    background: transparent;
    color: inherit;
    opacity: 0.7;
    cursor: pointer;
  }

  .close:hover {
    opacity: 1;
  }
</style>
