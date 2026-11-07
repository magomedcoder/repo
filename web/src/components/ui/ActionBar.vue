<template>
  <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
    <div
      v-if="$slots.leading"
      class="flex flex-wrap items-center gap-2"
    >
      <slot name="leading" />
    </div>
    <div
      v-if="$slots.actions || $slots.default"
      class="flex flex-wrap items-center gap-2"
    >
      <slot name="actions" />
      <slot />
    </div>
  </div>
</template>
