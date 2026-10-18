import type { Component } from 'vue'
import ChevronIcon from './ChevronIcon.vue'
import CodeIcon from './CodeIcon.vue'
import FileIcon from './FileIcon.vue'
import FolderIcon from './FolderIcon.vue'
import GearIcon from './GearIcon.vue'
import IssueClosedIcon from './IssueClosedIcon.vue'
import IssueIcon from './IssueIcon.vue'
import LockIcon from './LockIcon.vue'
import MarkIcon from './MarkIcon.vue'
import PullIcon from './PullIcon.vue'
import PullMergedIcon from './PullMergedIcon.vue'
import RepoIcon from './RepoIcon.vue'

export type IconName = | 'mark'
  | 'code'
  | 'issue'
  | 'issue-closed'
  | 'pull'
  | 'pull-merged'
  | 'gear'
  | 'folder'
  | 'file'
  | 'repo'
  | 'chevron'
  | 'lock'

export const icons: Record<IconName, Component> = {
  mark: MarkIcon,
  code: CodeIcon,
  issue: IssueIcon,
  'issue-closed': IssueClosedIcon,
  pull: PullIcon,
  'pull-merged': PullMergedIcon,
  gear: GearIcon,
  folder: FolderIcon,
  file: FileIcon,
  repo: RepoIcon,
  chevron: ChevronIcon,
  lock: LockIcon,
}

export {
  ChevronIcon,
  CodeIcon,
  FileIcon,
  FolderIcon,
  GearIcon,
  IssueClosedIcon,
  IssueIcon,
  LockIcon,
  MarkIcon,
  PullIcon,
  PullMergedIcon,
  RepoIcon,
}
