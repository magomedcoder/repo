<script setup lang="ts">
import { computed } from 'vue'
import alert from '@/assets/icons/alert.svg?raw'
import book from '@/assets/icons/book.svg?raw'
import branch from '@/assets/icons/branch.svg?raw'
import check from '@/assets/icons/check.svg?raw'
import chevronDown from '@/assets/icons/chevron-down.svg?raw'
import chevronUp from '@/assets/icons/chevron-up.svg?raw'
import clock from '@/assets/icons/clock.svg?raw'
import close from '@/assets/icons/close.svg?raw'
import commit from '@/assets/icons/commit.svg?raw'
import copy from '@/assets/icons/copy.svg?raw'
import device from '@/assets/icons/device.svg?raw'
import download from '@/assets/icons/download.svg?raw'
import eye from '@/assets/icons/eye.svg?raw'
import file from '@/assets/icons/file.svg?raw'
import folder from '@/assets/icons/folder.svg?raw'
import gear from '@/assets/icons/gear.svg?raw'
import home from '@/assets/icons/home.svg?raw'
import issue from '@/assets/icons/issue.svg?raw'
import issueClosed from '@/assets/icons/issue-closed.svg?raw'
import key from '@/assets/icons/key.svg?raw'
import list from '@/assets/icons/list.svg?raw'
import lock from '@/assets/icons/lock.svg?raw'
import logout from '@/assets/icons/logout.svg?raw'
import mail from '@/assets/icons/mail.svg?raw'
import mirror from '@/assets/icons/mirror.svg?raw'
import pencil from '@/assets/icons/pencil.svg?raw'
import plus from '@/assets/icons/plus.svg?raw'
import pull from '@/assets/icons/pull.svg?raw'
import pullMerged from '@/assets/icons/pull-merged.svg?raw'
import repo from '@/assets/icons/repo.svg?raw'
import search from '@/assets/icons/search.svg?raw'
import sort from '@/assets/icons/sort.svg?raw'
import sync from '@/assets/icons/sync.svg?raw'
import tag from '@/assets/icons/tag.svg?raw'
import trash from '@/assets/icons/trash.svg?raw'
import user from '@/assets/icons/user.svg?raw'
import users from '@/assets/icons/users.svg?raw'

export type IconName =
  | 'alert'
  | 'book'
  | 'branch'
  | 'check'
  | 'chevron-down'
  | 'chevron-up'
  | 'clock'
  | 'close'
  | 'commit'
  | 'copy'
  | 'device'
  | 'download'
  | 'eye'
  | 'file'
  | 'folder'
  | 'gear'
  | 'home'
  | 'issue'
  | 'issue-closed'
  | 'key'
  | 'list'
  | 'lock'
  | 'logout'
  | 'mail'
  | 'mirror'
  | 'pencil'
  | 'plus'
  | 'pull'
  | 'pull-merged'
  | 'repo'
  | 'search'
  | 'sort'
  | 'sync'
  | 'tag'
  | 'trash'
  | 'user'
  | 'users'

type IconSize = 'xs' | 'sm' | 'md' | 'lg'

const icons: Record<IconName, string> = {
  alert,
  book,
  branch,
  check,
  'chevron-down': chevronDown,
  'chevron-up': chevronUp,
  clock,
  close,
  commit,
  copy,
  device,
  download,
  eye,
  file,
  folder,
  gear,
  home,
  issue,
  'issue-closed': issueClosed,
  key,
  list,
  lock,
  logout,
  mail,
  mirror,
  pencil,
  plus,
  pull,
  'pull-merged': pullMerged,
  repo,
  search,
  sort,
  sync,
  tag,
  trash,
  user,
  users,
}

const sizeClass: Record<IconSize, string> = {
  xs: 'h-2.5 w-2.5',
  sm: 'h-3.5 w-3.5',
  md: 'h-4 w-4',
  lg: 'h-5 w-5',
}

const props = withDefaults(
  defineProps<{
    name: IconName
    size?: IconSize
  }>(),
  {
    size: 'md',
  },
)

const svg = computed(() => icons[props.name])
</script>

<template>
  <span
    class="inline-block shrink-0 align-middle [&>svg]:block [&>svg]:h-full [&>svg]:w-full"
    :class="sizeClass[size]"
    aria-hidden="true"
    v-html="svg"
  />
</template>
