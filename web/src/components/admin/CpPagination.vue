<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  page: number
  pageSize: number
  total: number
  loading?: boolean
}>()

const emit = defineEmits<{
  'update:page': [page: number]
}>()

const pageCount = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize) || 1))

const from = computed(() => {
  if (props.total <= 0) {
    return 0
  }

  return (props.page - 1) * props.pageSize + 1
})

const to = computed(() => Math.min(props.page * props.pageSize, props.total))

const show = computed(() => props.total > props.pageSize || props.page > 1)

function go(next: number) {
  const clamped = Math.min(Math.max(1, next), pageCount.value)
  if (clamped === props.page || props.loading) {
    return
  }

  emit('update:page', clamped)
}
</script>

<template>
  <div
    v-if="show"
    class="mt-3 flex flex-wrap items-center justify-between gap-2"
  >
    <p class="text-[12px] text-ink-muted">
      {{ $t('common.pageRange', { from, to, total }) }}
    </p>
    <div class="flex gap-1.5">
      <button
        type="button"
        class="btn"
        :disabled="loading || page <= 1"
        @click="go(page - 1)"
      >
        {{ $t('common.prev') }}
      </button>
      <button
        type="button"
        class="btn"
        :disabled="loading || page >= pageCount"
        @click="go(page + 1)"
      >
        {{ $t('common.next') }}
      </button>
    </div>
  </div>
</template>
