<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { AdminMirror } from '@/api/types'
import EmptyState from '@/components/ui/EmptyState.vue'
import Surface from '@/components/ui/Surface.vue'
import { formatDate } from '@/i18n'
import { mirrorRemoteLabel, mirrorUiStatus } from '@/lib/mirror'
import { repoLocation } from '@/lib/repoNav'

const props = defineProps<{
  items: AdminMirror[]
  selected: number[]
  busy?: boolean
  btnClass?: string
}>()

const emit = defineEmits<{
  'update:selected': [value: number[]]
  'toggle-all': []
  sync: [mirror: AdminMirror]
  pause: [mirror: AdminMirror]
}>()

const allSelected = () => props.items.length > 0 && props.selected.length === props.items.length

function rowStatus(m: AdminMirror) {
  return mirrorUiStatus({
    mirror_direction: m.direction,
    mirror_enabled: m.paused ? false : undefined,
    mirror_status: m.status,
    mirror_last_error: m.last_error,
    mirror_last_sync_at: m.last_sync_at,
  }) || 'idle'
}

function onToggle(ev: Event, id: number) {
  const checked = (ev.target as HTMLInputElement).checked
  if (checked) {
    emit('update:selected', [...props.selected, id])
  } else {
    emit('update:selected', props.selected.filter((x) => x !== id))
  }
}
</script>

<template>
  <Surface class="overflow-x-auto">
    <table class="w-full min-w-[960px] border-collapse text-left text-[10pt]">
      <thead class="bg-white/35 text-ink">
        <tr>
          <th class="w-8 border-b border-line px-2 py-1.5">
            <input
              type="checkbox"
              class="accent-ink"
              :checked="allSelected()"
              :aria-label="$t('adminMirrors.selectAll')"
              @change="emit('toggle-all')"
            >
          </th>
          <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('adminMirrors.repository') }}</th>
          <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('adminMirrors.direction') }}</th>
          <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('adminMirrors.status') }}</th>
          <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('adminMirrors.lastSync') }}</th>
          <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('adminMirrors.nextSync') }}</th>
          <th class="border-b border-line px-2 py-1.5 font-semibold">{{ $t('adminMirrors.lastError') }}</th>
          <th class="border-b border-line px-2 py-1.5" />
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="m in items"
          :key="m.id"
          class="align-top odd:bg-transparent even:bg-row-alt"
        >
          <td class="border-b border-line px-2 py-1.5">
            <input
              type="checkbox"
              class="accent-ink"
              :checked="selected.includes(m.id)"
              :aria-label="m.repo_path"
              @change="onToggle($event, m.id)"
            >
          </td>
          <td class="border-b border-line px-2 py-1.5">
            <RouterLink
              :to="repoLocation('repo-settings', m.repo_path)"
              class="font-mono text-[90%] no-underline hover:underline"
            >
              {{ m.repo_path }}
            </RouterLink>
            <p
              v-if="m.url"
              class="break-all font-mono text-[80%] text-ink-muted"
            >
              {{ mirrorRemoteLabel(m.url) }}
            </p>
          </td>
          <td class="border-b border-line px-2 py-1.5 font-mono text-[90%]">{{ m.direction }}</td>
          <td
            class="whitespace-nowrap border-b border-line px-2 py-1.5"
            :class="rowStatus(m) === 'error' && 'text-warn'"
          >
            {{ $t(`mirror.status.${rowStatus(m)}`) }}
            <span
              v-if="(m.fail_count ?? 0) > 0"
              class="text-ink-muted"
            > *{{ m.fail_count }}</span>
          </td>
          <td class="whitespace-nowrap border-b border-line px-2 py-1.5 text-ink-muted">
            {{ m.last_sync_at ? formatDate(m.last_sync_at) : '-' }}
          </td>
          <td class="whitespace-nowrap border-b border-line px-2 py-1.5 text-ink-muted">
            {{ m.next_sync_at && !m.paused ? formatDate(m.next_sync_at) : '-' }}
          </td>
          <td class="max-w-[22rem] break-words border-b border-line px-2 py-1.5 text-warn">
            {{ m.last_error || '' }}
          </td>
          <td class="whitespace-nowrap border-b border-line px-2 py-1.5 text-right">
            <button
              type="button"
              :class="[btnClass || 'btn', 'mr-1 px-2 py-0.5']"
              :disabled="busy || m.paused"
              @click="emit('sync', m)"
            >
              {{ $t('settings.mirrorSync') }}
            </button>
            <button
              type="button"
              :class="[btnClass || 'btn', 'px-2 py-0.5']"
              :disabled="busy"
              @click="emit('pause', m)"
            >
              {{ m.paused ? $t('adminMirrors.resume') : $t('adminMirrors.pause') }}
            </button>
          </td>
        </tr>
        <tr v-if="!items.length">
          <EmptyState
            as="td"
            :colspan="8"
            :message="$t('adminMirrors.empty')"
          />
        </tr>
      </tbody>
    </table>
  </Surface>
</template>
