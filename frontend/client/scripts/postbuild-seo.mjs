/**
 * Runs after `vite build`.
 *
 * The app is client-rendered, so the HTML a crawler first receives is an empty shell. Search engines
 * that run JavaScript recover, but WhatsApp / Facebook / Messenger link previews and many other bots
 * only read the raw <head>. This script writes a copy of index.html for every indexable page with that
 * page's own title, description and share image already in the <head>, and generates sitemap.xml.
 */
import { readFile, writeFile, mkdir, appendFile } from 'node:fs/promises';
import path from 'node:path';
import { transform } from 'esbuild';
import { loadEnv } from 'vite';

const root = process.cwd();
const dist = path.join(root, 'dist');
const SITE_URL = (process.env.VITE_SITE_URL ?? loadEnv('production', root, 'VITE_').VITE_SITE_URL ?? '').replace(/\/$/, '');

async function loadTs(file) {
  const source = await readFile(path.join(root, file), 'utf8');
  const { code } = await transform(source, { loader: 'ts', format: 'esm' });
  return import(`data:text/javascript;base64,${Buffer.from(code).toString('base64')}`);
}

/**
 * Products live in the backend now. To pre-render per-product share tags, set SEO_API_URL to the
 * public API (e.g. SEO_API_URL=https://api.example.com/api/v1) when building. Without it, product
 * pages still work (their tags are set in the browser) and only the pre-rendering/sitemap entries are skipped.
 */
async function loadProducts() {
  const api = (process.env.SEO_API_URL ?? '').replace(/\/$/, '');
  if (!api) return [];
  const out = [];
  try {
    for (let page = 1; page <= 50; page++) {
      const res = await fetch(`${api}/products?page=${page}&per_page=100`);
      const body = await res.json();
      if (!res.ok || !body.success) throw new Error(body.error?.message ?? res.status);
      for (const p of body.data) {
        out.push({
          id: p.slug,
          name: p.name,
          category: '',
          description: p.short_description || p.description || '',
          image: p.images?.[0]?.url ?? `${SITE_URL}/og-image.png`,
        });
      }
      if (page >= (body.meta?.total_pages ?? 1)) break;
    }
  } catch (err) {
    console.warn(`[seo] could not load products from ${api}: ${err.message}. Skipping product pages.`);
    return [];
  }
  return out;
}

const products = await loadProducts();
const { routeMeta, formatTitle, DEFAULT_DESCRIPTION } = await loadTs('src/config/seo.ts');

const esc = (text) => String(text).replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;');
const trim = (text, max = 155) => (text.length > max ? `${text.slice(0, max - 1).trimEnd()}…` : text);
const template = await readFile(path.join(dist, 'index.html'), 'utf8');

const MANAGED = /<meta (?:name|property)="(?:description|robots|og:title|og:description|og:image|og:url|og:type|twitter:title|twitter:description|twitter:image)"[^>]*>\s*|<link rel="canonical"[^>]*>\s*/g;

function render({ title, description, urlPath, image, type = 'website' }) {
  const fullTitle = formatTitle(title);
  const tags = [
    `<meta name="description" content="${esc(description)}" />`,
    `<meta property="og:type" content="${type}" />`,
    `<meta property="og:title" content="${esc(fullTitle)}" />`,
    `<meta property="og:description" content="${esc(description)}" />`,
    `<meta property="og:image" content="${esc(image)}" />`,
    `<meta name="twitter:image" content="${esc(image)}" />`,
  ];
  if (SITE_URL) {
    tags.push(`<link rel="canonical" href="${SITE_URL}${urlPath}" />`, `<meta property="og:url" content="${SITE_URL}${urlPath}" />`);
  }
  return template
    .replace(MANAGED, '')
    .replace(/<title>.*?<\/title>/, `<title>${esc(fullTitle)}</title>`)
    .replace('</head>', `    ${tags.join('\n    ')}\n  </head>`);
}

async function write(urlPath, html) {
  const dir = path.join(dist, urlPath);
  await mkdir(dir, { recursive: true });
  await writeFile(path.join(dir, 'index.html'), html);
}

const defaultImage = `${SITE_URL}/og-image.png`;
const urls = ['/'];
let pages = 0;

for (const [route, meta] of Object.entries(routeMeta)) {
  if (meta.noindex || route === '/') continue;
  await write(route, render({ title: meta.title, description: meta.description, urlPath: route, image: defaultImage }));
  urls.push(route);
  pages++;
}

for (const product of products) {
  const urlPath = `/products/${product.id}`;
  await write(urlPath, render({
    title: product.category ? `${product.name} - ${product.category}` : product.name,
    description: trim(product.description || DEFAULT_DESCRIPTION),
    urlPath,
    image: product.image,
    type: 'product',
  }));
  urls.push(urlPath);
  pages++;
}

if (SITE_URL) {
  const today = new Date().toISOString().slice(0, 10);
  const xml = `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${urls
    .map((u) => `  <url><loc>${SITE_URL}${u}</loc><lastmod>${today}</lastmod></url>`)
    .join('\n')}\n</urlset>\n`;
  await writeFile(path.join(dist, 'sitemap.xml'), xml);
  await appendFile(path.join(dist, 'robots.txt'), `\nSitemap: ${SITE_URL}/sitemap.xml\n`);
  console.log(`[seo] wrote ${pages} page heads + sitemap.xml (${urls.length} URLs) for ${SITE_URL}`);
} else {
  console.warn(`[seo] wrote ${pages} page heads. VITE_SITE_URL is not set, so canonical URLs, absolute share-image URLs and sitemap.xml were skipped.`);
}
