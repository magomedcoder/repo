<script setup lang="ts" generic="T extends string">
defineProps<{
  modelValue: T
  options: {
    value: T
    label: string
    icon?: string
  }[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: T]
}>()
</script>

<template>
  <div class="inline-flex overflow-hidden rounded-lg border border-line bg-white/40 text-sm backdrop-blur-md">
    <button
      v-for="(option, index) in options"
      :key="option.value"
      type="button"
      class="inline-flex items-center gap-1.5 px-3 py-1.5"
      :class="[
        index > 0 && 'border-l border-line',
        modelValue === option.value ? 'bg-white/55 font-medium text-ink' : 'text-ink-muted hover:text-ink',
      ]"
      @click="emit('update:modelValue', option.value)"
    >
      <slot
        name="option"
        :option="option"
        :active="modelValue === option.value"
      >
        {{ option.label }}
      </slot>
    </button>
  </div>
</template>
