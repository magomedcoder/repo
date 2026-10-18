<script setup lang="ts">
import { computed } from 'vue'
import { icons, type IconName } from './icons'

const props = defineProps<{
  name: IconName
}>()

const icon = computed(() => icons[props.name])
</script>

<template>
  <component :is="icon" />
</template>
