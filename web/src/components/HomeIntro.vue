<script setup lang="ts">
import Icon from '@/components/ui/Icon.vue'

const email = 'email@email.email'
</script>

<template>
  <aside
    class="home-intro surface relative mb-4 overflow-hidden sm:mb-5"
    role="note"
  >
    <div
      class="pointer-events-none absolute inset-0 bg-[radial-gradient(520px_220px_at_8%_-20%,rgba(255,255,255,0.7),transparent_58%),radial-gradient(420px_200px_at_100%_0%,rgba(26,77,140,0.12),transparent_55%)]"
      aria-hidden="true"
    />
    <div
      class="pointer-events-none absolute inset-y-0 left-0 w-[3px] bg-accent"
      aria-hidden="true"
    />

    <div class="relative flex gap-3.5 px-3.5 py-3.5 sm:gap-4 sm:px-5 sm:py-4">
      <div
        class="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full border border-line-strong/60 bg-white/70 text-accent shadow-[inset_0_1px_0_rgba(255,255,255,0.85)]"
        aria-hidden="true"
      >
        <Icon name="book" size="sm" />
      </div>

      <div class="min-w-0 flex-1">
        <p class="m-0 text-[12px] font-medium tracking-wide text-ink-muted sm:text-[13px]">
          {{ $t('intro.welcome') }}
        </p>
        <p class="mt-1.5 mb-0 text-[1.05rem] font-semibold leading-snug text-ink sm:text-[1.15rem]">
          {{ $t('intro.headline') }}
        </p>
        <p class="mt-2 mb-0 max-w-[46rem] text-[12px] leading-relaxed text-ink-muted sm:text-[13px]">
          {{ $t('intro.body') }}
        </p>
        <p class="mt-3 mb-0">
          <a
            class="btn inline-flex items-center gap-1.5 px-2.5 py-1 text-[12px] no-underline hover:no-underline"
            :href="`mailto:${email}`"
          >
            <Icon name="mail" size="sm" class="text-accent" />
            <span class="font-medium text-ink">{{ email }}</span>
          </a>
        </p>
      </div>
    </div>
  </aside>
</template>
