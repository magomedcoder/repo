import { onBeforeUnmount, watch, type MaybeRefOrGetter, toValue } from 'vue'
import { getLocale, i18n } from '@/i18n'
import { useSite } from '@/composables/useSite'

export type PageMetaInput = {
  title?: string
  description?: string
  robots?: string
  canonicalPath?: string
  ogType?: string
  jsonLd?: Record<string, unknown> | Record<string, unknown>[] | null
  noindex?: boolean
}

function upsertMeta(attr: 'name' | 'property', key: string, content: string) {
  const selector = `meta[${attr}="${CSS.escape(key)}"]`
  let el = document.head.querySelector(selector) as HTMLMetaElement | null
  if (!el) {
    el = document.createElement('meta')
    el.setAttribute(attr, key)
    el.setAttribute('data-seo', '1')
    document.head.appendChild(el)
  }

  el.content = content
}

function upsertLink(rel: string, href: string) {
  let el = document.head.querySelector(`link[rel="${CSS.escape(rel)}"][data-seo="1"]`) as HTMLLinkElement | null
  if (!el) {
    el = document.createElement('link')
    el.rel = rel
    el.setAttribute('data-seo', '1')
    document.head.appendChild(el)
  }

  el.href = href
}

function upsertJSONLD(data: Record<string, unknown> | Record<string, unknown>[] | null | undefined) {
  let el = document.getElementById('seo-jsonld') as HTMLScriptElement | null
  if (!data) {
    el?.remove()
    return
  }

  if (!el) {
    el = document.createElement('script')
    el.type = 'application/ld+json'
    el.id = 'seo-jsonld'
    document.head.appendChild(el)
  }

  el.textContent = JSON.stringify(data)
}

function originOf(publicURL: string): string {
  const configured = publicURL.trim().replace(/\/$/, '')
  if (configured) {
    return configured
  }

  return window.location.origin
}

function buildTitle(title: string, site: string): string {
  const t = title.trim()
  if (!t) {
    return site
  }

  if (t === site || t.endsWith(` | ${site}`)) {
    return t
  }

  const full = `${t} | ${site}`
  return full.length <= 60 ? full : t
}

export function usePageMeta(input: MaybeRefOrGetter<PageMetaInput>) {
  const site = useSite()

  const stop = watch(
    () => ({
      meta: toValue(input),
      siteName: site.name.value,
      publicURL: site.publicURL.value,
      locale: getLocale(),
    }),
    ({ meta, siteName, publicURL, locale }) => {
      const origin = originOf(publicURL)
      const robots = meta.noindex || meta.robots?.includes('noindex')
          ? meta.robots || 'noindex,nofollow'
          : meta.robots || 'index,follow'

      document.title = buildTitle(meta.title || siteName, siteName)

      const description = meta.description?.trim() || String(i18n.global.t('seo.defaultDescription', { site: siteName }))

      document.documentElement.lang = locale === 'ru' ? 'ru' : 'en'

      upsertMeta('name', 'description', description)
      upsertMeta('name', 'robots', robots)

      const path = meta.canonicalPath || `${window.location.pathname}${window.location.search}`
      const canonical = origin + (path.startsWith('/') ? path : `/${path}`)
      upsertLink('canonical', canonical)

      upsertMeta('property', 'og:type', meta.ogType || 'website')
      upsertMeta('property', 'og:title', document.title)
      upsertMeta('property', 'og:description', description)
      upsertMeta('property', 'og:url', canonical)
      upsertMeta('property', 'og:site_name', siteName)
      upsertMeta('property', 'og:locale', locale === 'ru' ? 'ru_RU' : 'en_US')
      upsertMeta('property', 'og:image', `${origin}/og.png`)
      upsertMeta('name', 'twitter:card', 'summary')
      upsertMeta('name', 'twitter:title', document.title)
      upsertMeta('name', 'twitter:description', description)

      upsertJSONLD(meta.jsonLd ?? null)
    },
    { immediate: true },
  )

  onBeforeUnmount(() => {
    stop()
  })
}
