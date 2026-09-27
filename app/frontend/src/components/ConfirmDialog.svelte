<script lang="ts">
  import { t } from '../lib/i18n'
  import { confirmer } from '../lib/ui.svelte'

  let confirmButton = $state<HTMLButtonElement | null>(null)

  $effect(() => {
    if (confirmer.current) confirmButton?.focus()
  })

  function onKeydown(e: KeyboardEvent) {
    if (!confirmer.current) return
    if (e.key === 'Escape') {
      e.preventDefault()
      confirmer.answer(false)
    }
  }
</script>

<svelte:window onkeydown={onKeydown} />

{#if confirmer.current}
  {@const req = confirmer.current}
  <div class="backdrop" role="presentation" onclick={() => confirmer.answer(false)}></div>
  <div class="dialog" role="alertdialog" aria-modal="true" aria-labelledby="confirm-message">
    <p id="confirm-message" class="selectable">{req.message}</p>
    <div class="actions">
      <button class="ft-btn" onclick={() => confirmer.answer(false)}>{t('app.common.cancel')}</button>
      <button
        bind:this={confirmButton}
        class="ft-btn ft-btn-primary"
        class:danger={req.danger}
        onclick={() => confirmer.answer(true)}
      >
        {req.confirmLabel ?? t('app.common.ok')}
      </button>
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: var(--bg-overlay);
    z-index: 50;
  }

  .dialog {
    position: fixed;
    top: 30%;
    left: 50%;
    transform: translateX(-50%);
    width: min(420px, calc(100vw - 32px));
    padding: 20px;
    border: 1px solid var(--border);
    border-radius: 12px;
    background: var(--bg-elevated);
    box-shadow: 0 20px 50px var(--shadow);
    z-index: 51;
  }

  p {
    margin: 0 0 18px;
    color: var(--text-primary);
    white-space: pre-line;
  }

  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }

  .danger {
    background: var(--danger);
    border-color: var(--danger);
  }
</style>
