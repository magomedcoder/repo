import type { IconName } from '@/components/ui/Icon.vue'

export type SettingsSection = 'general' | 'mirror' | 'collaborators' | 'teams' | 'danger'

export const SETTINGS_SECTIONS: {
  id: SettingsSection
  labelKey: string
  icon: IconName
}[] = [
  { id: 'general', labelKey: 'settings.general', icon: 'gear' },
  { id: 'mirror', labelKey: 'settings.mirrorTitle', icon: 'mirror' },
  { id: 'collaborators', labelKey: 'collaborators.title', icon: 'users' },
  { id: 'teams', labelKey: 'repoSettings.teams.title', icon: 'users' },
  { id: 'danger', labelKey: 'settings.danger', icon: 'alert' },
]
