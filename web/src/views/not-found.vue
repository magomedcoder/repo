<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <div class="py-8">
    <h1 class="mb-2 text-lg font-medium text-ink">{{ $t('errors.notFound') }}</h1>
    <p class="mb-4 text-sm text-ink-muted">{{ $t('errors.notFoundHint') }}</p>
    <RouterLink to="/" class="text-ink no-underline hover:underline">{{ $t('nav.root') }}</RouterLink>
  </div>
</template>
