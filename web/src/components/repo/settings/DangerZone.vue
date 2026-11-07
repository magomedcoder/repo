<script setup lang="ts">
import Icon from '@/components/ui/Icon.vue'
import Surface from '@/components/ui/Surface.vue'
import SurfaceHeader from '@/components/ui/SurfaceHeader.vue'

defineProps<{
  deleting: boolean
}>()

const emit = defineEmits<{
  delete: []
}>()
</script>

<template>
  <Surface>
    <SurfaceHeader as="h2" class="text-warn">
      <Icon name="alert" size="sm" />
      {{ $t('settings.danger') }}
    </SurfaceHeader>
    <div class="flex flex-wrap items-center justify-between gap-3 px-3 py-3 sm:px-4">
      <div>
        <p class="font-medium">{{ $t('settings.delete') }}</p>
        <p class="text-sm text-ink-muted">{{ $t('settings.dangerText') }}</p>
      </div>
      <button type="button" class="btn-warn" :disabled="deleting" @click="emit('delete')">
        <Icon name="trash" size="sm" />
        {{ deleting ? $t('settings.deleting') : $t('settings.delete') }}
      </button>
    </div>
  </Surface>
</template>
