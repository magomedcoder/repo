<script setup lang="ts">
withDefaults(
  defineProps<{
    as?: 'input' | 'select' | 'textarea'
    type?: string
    modelValue?: string | number | null
    disabled?: boolean
    required?: boolean
    placeholder?: string
  }>(),
  {
    as: 'input',
    type: 'text',
  },
)

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

function onInput(ev: Event) {
  const target = ev.target as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement
  emit('update:modelValue', target.value)
}
</script>

<template>
  <component
    :is="as"
    :type="as === 'input' ? type : undefined"
    :value="modelValue ?? ''"
    :disabled="disabled"
    :required="required"
    :placeholder="placeholder"
    class="field"
    @input="onInput"
    @change="onInput"
  >
    <slot v-if="as === 'select'" />
  </component>
</template>
