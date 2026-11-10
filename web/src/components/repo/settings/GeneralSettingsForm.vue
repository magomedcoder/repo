<script setup lang="ts">
import Icon from '@/components/ui/Icon.vue'
import Surface from '@/components/ui/Surface.vue'
import SurfaceHeader from '@/components/ui/SurfaceHeader.vue'

defineProps<{
  branchOptions: string[]
  formError: string
  message: string
  saving: boolean
}>()

const emit = defineEmits<{
  submit: []
}>()

const name = defineModel<string>('name', { required: true })
const description = defineModel<string>('description', { required: true })
const defaultBranch = defineModel<string>('defaultBranch', { required: true })
const isPrivate = defineModel<boolean>('isPrivate', { required: true })
const protectBranch = defineModel<boolean>('protectBranch', { required: true })
const issuesEnabled = defineModel<boolean>('issuesEnabled', { required: true })
</script>

<template>
  <Surface as="form" @submit.prevent="emit('submit')">
    <SurfaceHeader>
      <Icon name="gear" size="sm" class="text-ink-muted" />
      {{ $t('settings.general') }}
    </SurfaceHeader>
    <div class="space-y-4 p-3 sm:p-4">
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('settings.name') }}</span>
        <input
          v-model="name"
          :class="['field', 'font-mono']"
          required
          :placeholder="$t('modal.repoNamePlaceholder')"
        />
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('settings.description') }}</span>
        <textarea v-model="description" rows="3" :class="['field', 'min-h-20 resize-y']" />
      </label>
      <label class="block text-sm">
        <span class="mb-1 block font-semibold">{{ $t('settings.defaultBranch') }}</span>
        <select
          v-model="defaultBranch"
          :class="['field', 'font-mono']"
          :disabled="!branchOptions.length"
        >
          <option v-for="b in branchOptions" :key="b" :value="b">{{ b }}</option>
        </select>
        <p v-if="!branchOptions.length" class="mt-1 text-xs text-ink-muted">
          {{ $t('settings.noBranches') }}
        </p>
      </label>
      <label class="flex items-center gap-2 text-sm">
        <input v-model="isPrivate" type="checkbox" class="accent-ink" />
        {{ $t('settings.private') }}
      </label>
      <label class="flex items-center gap-2 text-sm">
        <input v-model="protectBranch" type="checkbox" class="accent-ink" />
        {{ $t('settings.protectBranch') }}
      </label>
      <label class="flex items-center gap-2 text-sm">
        <input v-model="issuesEnabled" type="checkbox" class="accent-ink" />
        {{ $t('settings.issues') }}
      </label>
      <p v-if="formError" class="text-sm text-warn">{{ formError }}</p>
      <p v-else-if="message" class="text-sm text-ink">{{ message }}</p>
      <button type="submit" class="btn-primary" :disabled="saving">
        {{ saving ? $t('settings.saving') : $t('settings.save') }}
      </button>
    </div>
  </Surface>
</template>
