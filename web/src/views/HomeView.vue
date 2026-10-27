<script setup lang="ts">
import { btnGhost } from '@/lib/classes'
import { onMounted, ref } from 'vue'
import { reposApi } from '@/api'
import type { Repository } from '@/api/types'
import ContentsList from '@/components/ContentsList.vue'
import { localizeError } from '@/i18n'

const PAGE_SIZE = 50

const repos = ref<Repository[]>([])
const error = ref('')
const loading = ref(true)
const loadingMore = ref(false)
const hasMore = ref(false)
const sort = ref('name')

async function load(append = false) {
  if (append) {
    loadingMore.value = true
  } else {
    loading.value = true
  }
  error.value = ''
  try {
    const offset = append ? repos.value.length : 0
    const repoRes = await reposApi.listOwn(sort.value, { offset, limit: PAGE_SIZE })
    const batch = repoRes.repos ?? []
    repos.value = append ? [...repos.value, ...batch] : batch
    hasMore.value = batch.length >= PAGE_SIZE
  } catch (err) {
    error.value = localizeError(err, 'errors.loadFailed')
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function onSort(key: string) {
  sort.value = key
  load()
}

function loadMore() {
  if (!loadingMore.value && hasMore.value) {
    load(true)
  }
}

onMounted(() => load())
</script>

<template>
  <div>
    <p v-if="loading" class="text-ink-muted">{{ $t('common.loading') }}</p>
    <p v-else-if="error" class="text-warn">{{ error }}</p>
    <template v-else>
      <ContentsList
        :repos="repos"
        show-folder
        @sort="onSort"
      />
      <p v-if="hasMore" class="mt-3">
        <button type="button" :class="btnGhost" :disabled="loadingMore" @click="loadMore">
          {{ loadingMore ? $t('common.loading') : $t('common.loadMore') }}
        </button>
      </p>
    </template>
  </div>
</template>
