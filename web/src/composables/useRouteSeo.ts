import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { i18n } from '@/i18n'
import { usePageMeta } from '@/composables/usePageMeta'
import { useSite } from '@/composables/useSite'
import { useRepo } from '@/composables/useRepo'
import { normalizeRepoPath } from '@/lib/repoPath'

const NOINDEX_NAMES = new Set([
  'login',
  'register',
  'profile',
  'profile-email',
  'profile-password',
  'sessions',
  'tokens',
  'teams',
  'team',
  'cp',
  'users',
  'access-logs',
  'admin-mirrors',
  'cp-repos',
  'repo-settings',
])

export function useRouteSeo() {
  const route = useRoute()
  const site = useSite()

  usePageMeta(() => {
    const siteName = site.name.value
    const name = String(route.name || '')
    const repoPath = String(route.params.repoPath || '')

    if (
      NOINDEX_NAMES.has(name) ||
      route.matched.some((r) => r.meta.requiresAdmin || r.meta.requiresRepoAdmin || r.meta.guest)
    ) {
      return {
        title: String(route.meta.title || name || siteName),
        noindex: true,
        canonicalPath: route.path,
      }
    }

    if (name === 'home') {
      return {
        title: siteName,
        description: String(i18n.global.t('seo.homeDescription', {
          site: siteName
        })),
        canonicalPath: '/',
        ogType: 'website',
        jsonLd: {
          '@context': 'https://schema.org',
          '@type': 'WebSite',
          name: siteName,
          url: (site.publicURL.value || window.location.origin) + '/',
          potentialAction: {
            '@type': 'SearchAction',
            target: `${site.publicURL.value || window.location.origin}/search?q={search_term_string}`,
            'query-input': 'required name=search_term_string',
          },
        },
      }
    }

    if (name === 'search') {
      const q = String(route.query.q || '').trim()
      if (!q) {
        return {
          title: String(i18n.global.t('seo.searchTitle')),
          description: String(i18n.global.t('seo.searchDescription', { site: siteName })),
          noindex: true,
          canonicalPath: '/search',
        }
      }

      return {
        title: q,
        description: String(i18n.global.t('seo.searchResultsDescription', { q, site: siteName })),
        canonicalPath: `/search?q=${encodeURIComponent(q)}`,
      }
    }

    if (name === 'not-found') {
      return {
        title: String(i18n.global.t('seo.notFoundTitle')),
        noindex: true,
        robots: 'noindex,follow',
        canonicalPath: route.path,
      }
    }

    if (repoPath && String(name).startsWith('repo')) {
      return {
        title: normalizeRepoPath(repoPath) || siteName,
        canonicalPath: route.path,
        ogType: 'article',
        noindex: name === 'repo-settings',
      }
    }

    return {
      title: siteName,
      canonicalPath: route.path,
    }
  })
}

export function useRepoSeo(repoPath: () => string) {
  const route = useRoute()
  const site = useSite()
  const { repo } = useRepo(repoPath)

  usePageMeta(
    computed(() => {
      const siteName = site.name.value
      const r = repo.value
      const path = normalizeRepoPath(repoPath())
      if (!r) {
        return {
          title: path || siteName,
          canonicalPath: route.path,
          ogType: 'article',
          noindex: route.name === 'repo-settings',
        }
      }

      const desc = r.description?.trim() || String(i18n.global.t('seo.repoDescriptionFallback', {
        name: r.name,
        site: siteName
      }))

      const section = repoSectionTitle(String(route.name || ''))
      const title = section ? `${section} | ${r.name}` : r.name
      const origin = site.publicURL.value || window.location.origin

      return {
        title,
        description: desc,
        canonicalPath: route.path,
        ogType: 'article',
        noindex: !!r.is_private || route.name === 'repo-settings',
        jsonLd: [
          {
            '@context': 'https://schema.org',
            '@type': 'SoftwareSourceCode',
            name: r.name,
            description: desc,
            codeRepository: `${origin}/${r.name}.git`,
            url: `${origin}/${r.name}`,
          },
          {
            '@context': 'https://schema.org',
            '@type': 'BreadcrumbList',
            itemListElement: [
              { 
                '@type': 'ListItem', 
                position: 1, 
                name: siteName, 
                item: `${origin}/` 
              },
              { 
                '@type': 'ListItem', 
                position: 2, 
                name: r.name, 
                item: `${origin}/${r.name}` 
              },
            ],
          },
        ],
      }
    }),
  )
}

function repoSectionTitle(name: string): string {
  switch (name) {
    case 'repo-issues':
    case 'repo-issue':
      return 'Issues'
    case 'repo-pulls':
    case 'repo-pull':
      return 'Pull requests'
    case 'repo-commits':
    case 'repo-commit':
      return 'Commits'
    case 'repo-blob':
      return 'File'
    case 'repo-refs':
      return 'Refs'
    case 'repo-settings':
      return 'Settings'
    default:
      return ''
  }
}
