<script lang="ts">
  import Toast from "./Toast.svelte";

  import { dismissToast, toasts } from "./core";
</script>

{#if $toasts}
  <section>
    {#each $toasts as toast (toast.id)}
      <Toast
        type={toast.type}
        dismissible={toast.dismissible}
        on:dismiss={() => dismissToast(toast.id)}>{toast.message}</Toast
      >
    {/each}
  </section>
{/if}

<style lang="postcss">
    section {
        position: fixed;
        top: 0;
        right: 0;
        display: flex;
        margin-top: 1rem;
        flex-direction: column;
        z-index: 1000;
    }
</style>
