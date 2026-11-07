<script setup lang="ts">
import { RouterView } from 'vue-router'
import CpNav from '@/components/admin/CpNav.vue'
import Icon from '@/components/ui/Icon.vue'
</script>

<template>
  <div class="page-enter space-y-4">
    <div class="surface overflow-hidden">
      <div class="flex flex-wrap items-center gap-2 border-b border-line-strong/40 px-3 py-2.5 sm:px-4">
        <span class="inline-flex h-8 w-8 items-center justify-center rounded-full border border-line-strong/70 bg-white/45 text-ink-muted">
          <Icon name="gear" size="md" />
        </span>
        <div class="min-w-0">
          <h1 class="text-[1.05rem] font-semibold leading-snug text-ink">
            {{ $t('nav.controlPanel') }}
          </h1>
          <p class="text-[12px] text-ink-muted">{{ $t('cp.subtitle') }}</p>
        </div>
      </div>
    </div>

    <div class="flex flex-col gap-4 md:flex-row md:items-start md:gap-5">
      <CpNav />
      <div class="min-w-0 flex-1">
        <RouterView />
      </div>
    </div>
  </div>
</template>
