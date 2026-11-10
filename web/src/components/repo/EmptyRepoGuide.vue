<script setup lang="ts">
import { computed } from 'vue'
import { cloneUrl } from '@/api'
import { normalizeRepoPath } from '@/lib/repoPath'

const props = defineProps<{
  repoPath: string
  defaultBranch?: string
}>()

const branch = computed(() => (props.defaultBranch || 'main').trim() || 'main')
const url = computed(() => cloneUrl(props.repoPath))
const name = computed(() => normalizeRepoPath(props.repoPath))

const createCommands = computed(() =>
  [
    `echo "# ${name.value}" >> README.md`,
    'git init',
    'git add README.md',
    'git commit -m "first commit"',
    `git branch -M ${branch.value}`,
    `git remote add origin ${url.value}`,
    `git push -u origin ${branch.value}`,
  ].join('\n'),
)

const pushCommands = computed(() =>
  [
    `git remote add origin ${url.value}`,
    `git branch -M ${branch.value}`,
    `git push -u origin ${branch.value}`,
  ].join('\n'),
)
</script>

<template>
  <section class="surface overflow-hidden">
    <div class="border-b border-line-strong/40 px-3 py-2.5 sm:px-4">
      <h2 class="m-0 text-[15px] font-semibold text-ink">
        {{ $t('repo.emptyGuide.title') }}
      </h2>
      <p class="mt-1 mb-0 text-[12px] text-ink-muted">
        {{ $t('repo.emptyGuide.hint') }}
      </p>
    </div>

    <div class="space-y-4 p-3 sm:p-4">
      <div>
        <h3 class="mb-1.5 m-0 text-[13px] font-semibold text-ink">
          {{ $t('repo.emptyGuide.createTitle') }}
        </h3>
        <pre
          class="m-0 overflow-x-auto rounded-lg border border-line-strong/50 bg-white/45 px-3 py-2.5 font-mono text-[11px] leading-relaxed text-ink whitespace-pre"
        >{{ createCommands }}</pre>
      </div>

      <div>
        <h3 class="mb-1.5 m-0 text-[13px] font-semibold text-ink">
          {{ $t('repo.emptyGuide.pushTitle') }}
        </h3>
        <pre
          class="m-0 overflow-x-auto rounded-lg border border-line-strong/50 bg-white/45 px-3 py-2.5 font-mono text-[11px] leading-relaxed text-ink whitespace-pre"
        >{{ pushCommands }}</pre>
      </div>

      <p class="m-0 text-[12px] text-ink-muted">
        {{ $t('repo.emptyGuide.authHint') }}
      </p>
    </div>
  </section>
</template>
