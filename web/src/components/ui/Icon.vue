<script setup lang="ts">
defineProps<{
  name: 'mark' | 'code' | 'issue' | 'issue-closed' | 'gear' | 'folder' | 'file' | 'repo' | 'chevron' | 'lock'
}>()
</script>

<template>
  <svg viewBox="0 0 16 16" class="inline-block h-4 w-4 shrink-0" fill="currentColor" aria-hidden="true">
    <path v-if="name === 'mark'" d="M8 0a8 8 0 1 0 0 16A8 8 0 0 0 8 0Zm3.2 11.2-1.1-1.1a2.4 2.4 0 0 1-3.2-3.2L5.8 5.8a3.6 3.6 0 0 0 5.4 5.4Z" />
    <path v-else-if="name === 'code'" d="M4.7 5.3 1.9 8l2.8 2.7-.7.7L.5 8l3.5-3.4.7.7Zm6.6 0 .7-.7L15.5 8l-3.5 3.4-.7-.7L14.1 8l-2.8-2.7ZM9.3 3.2l-2.8 9.6-.9-.3 2.8-9.6.9.3Z" />
    <path v-else-if="name === 'issue'" d="M8 1.5a6.5 6.5 0 1 0 0 13 6.5 6.5 0 0 0 0-13ZM0 8a8 8 0 1 1 16 0A8 8 0 0 1 0 8Zm8-3.5a.75.75 0 0 1 .75.75v2.5a.75.75 0 0 1-1.5 0v-2.5A.75.75 0 0 1 8 4.5Zm0 7.5a1 1 0 1 1 0-2 1 1 0 0 1 0 2Z" />
    <path v-else-if="name === 'issue-closed'" d="M11.3 6.3 7.5 10.1 5.2 7.8l-1 1.1 3.3 3.3 5.8-5.8-1-1.1ZM8 1.5a6.5 6.5 0 1 0 0 13 6.5 6.5 0 0 0 0-13ZM0 8a8 8 0 1 1 16 0A8 8 0 0 1 0 8Z" />
    <path v-else-if="name === 'gear'" d="M8 4.8a3.2 3.2 0 1 0 0 6.4 3.2 3.2 0 0 0 0-6.4ZM8 6a2 2 0 1 1 0 4 2 2 0 0 1 0-4Zm6.4 2.6-.1-1.2 1.3-.8-1.2-2-1.4.4a5.2 5.2 0 0 0-1-.6L11.6 3H9.4l-.4 1.4c-.4.1-.7.3-1 .6L6.6 4.6l-1.2 2 1.3.8-.1 1.2 1.3.8a5 5 0 0 0 .4 1.1l-.8 1.2 1.2 2 1.4-.4c.3.2.7.4 1 .5l.4 1.4h2.2l.4-1.4c.4-.1.7-.3 1-.5l1.4.4 1.2-2-.8-1.2c.2-.3.3-.7.4-1.1l1.3-.8-.1-1.2Z" />
    <path v-else-if="name === 'folder'" d="M1.8 2.5h4.2l1.3 1.5h7a1 1 0 0 1 1 1v8a1 1 0 0 1-1 1h-12a1 1 0 0 1-1-1v-9a1 1 0 0 1 1-1Z" />
    <path v-else-if="name === 'file'" d="M3 1.5h6.2L13 5.3V14a.5.5 0 0 1-.5.5h-9a.5.5 0 0 1-.5-.5v-12a.5.5 0 0 1 .5-.5Zm6 1.2V5h2.3L9 2.7Z" />
    <path v-else-if="name === 'repo'" d="M2 2.5A1.5 1.5 0 0 1 3.5 1h9A1.5 1.5 0 0 1 14 2.5v11a1.5 1.5 0 0 1-1.5 1.5h-9A1.5 1.5 0 0 1 2 13.5v-11ZM3.5 2a.5.5 0 0 0-.5.5v11a.5.5 0 0 0 .5.5h9a.5.5 0 0 0 .5-.5v-11a.5.5 0 0 0-.5-.5h-9ZM5 5h6v1H5V5Zm0 3h6v1H5V8Zm0 3h4v1H5v-1Z" />
    <path v-else-if="name === 'chevron'" d="M4.4 6.3 8 9.9l3.6-3.6.8.8L8 11.5 3.6 7.1l.8-.8Z" />
    <path v-else-if="name === 'lock'" d="M4 7V5a4 4 0 1 1 8 0v2h1v7H3V7h1Zm2 0h4V5a2 2 0 1 0-4 0v2Z" />
  </svg>
</template>
