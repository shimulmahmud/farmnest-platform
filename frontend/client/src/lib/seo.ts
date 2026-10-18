import { DEFAULT_DESCRIPTION, SITE_NAME, formatTitle } from '@/config/seo';

interface PageMeta {
  title: string;
  description?: string;
  /** Absolute image URL for link previews. */
  image?: string;
  noindex?: boolean;
  type?: 'website' | 'product';
}

function upsertMeta(attr: 'name' | 'property', key: string, content: string) {
  let el = document.head.querySelector<HTMLMetaElement>(`meta[${attr}="${key}"]`);
  if (!el) {
    el = document.createElement('meta');
    el.setAttribute(attr, key);
    document.head.appendChild(el);
  }
  el.setAttribute('content', content);
}

function upsertCanonical(href: string) {
  let el = document.head.querySelector<HTMLLinkElement>('link[rel="canonical"]');
  if (!el) {
    el = document.createElement('link');
    el.setAttribute('rel', 'canonical');
    document.head.appendChild(el);
  }
  el.setAttribute('href', href);
}

/** Keeps <title>, description, canonical, robots and Open Graph tags in step with the current page. */
export function applyPageMeta({ title, description = DEFAULT_DESCRIPTION, image, noindex = false, type = 'website' }: PageMeta) {
  const fullTitle = formatTitle(title);
  const url = `${window.location.origin}${window.location.pathname}`;

  document.title = fullTitle;
  upsertMeta('name', 'description', description);
  upsertMeta('name', 'robots', noindex ? 'noindex, nofollow' : 'index, follow');
  upsertCanonical(url);

  upsertMeta('property', 'og:site_name', SITE_NAME);
  upsertMeta('property', 'og:type', type === 'product' ? 'product' : 'website');
  upsertMeta('property', 'og:title', fullTitle);
  upsertMeta('property', 'og:description', description);
  upsertMeta('property', 'og:url', url);
  upsertMeta('name', 'twitter:title', fullTitle);
  upsertMeta('name', 'twitter:description', description);
  if (image) {
    upsertMeta('property', 'og:image', image);
    upsertMeta('name', 'twitter:image', image);
  }
}

/** Adds (or replaces) a JSON-LD block; returns a cleanup that removes it. */
export function setJsonLd(id: string, data: object): () => void {
  document.getElementById(id)?.remove();
  const script = document.createElement('script');
  script.type = 'application/ld+json';
  script.id = id;
  script.textContent = JSON.stringify(data);
  document.head.appendChild(script);
  return () => script.remove();
}
