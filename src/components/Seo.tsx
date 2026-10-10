import { useEffect } from 'react';
import { withBreadcrumbs } from '../lib/seoSchema';

interface SeoProps {
  title: string;
  description: string;
  path?: string;
  image?: string;
  type?: 'website' | 'article';
  jsonLd?: object | object[];
  noindex?: boolean;
  breadcrumbName?: string;
}

const BASE_URL = 'https://openpaths.io';
const DEFAULT_OG_IMAGE = `${BASE_URL}/og-image.png`;
const TWITTER_HANDLE = '@openpaths';

export function Seo({ title, description, path = '/', image, type = 'website', jsonLd, noindex = false, breadcrumbName }: SeoProps) {
  const jsonLdKey = JSON.stringify(jsonLd ?? null);
  useEffect(() => {
    const normalizedPath = path.startsWith('/') ? path : `/${path}`;
    const canonicalUrl = `${BASE_URL}${normalizedPath}`;
    const imageUrl = image ? absoluteUrl(image) : DEFAULT_OG_IMAGE;

    document.title = title;
    upsertMeta('name', 'description', description);
    upsertMeta('property', 'og:type', type);
    upsertMeta('property', 'og:site_name', 'OpenPaths');
    upsertMeta('property', 'og:url', canonicalUrl);
    upsertMeta('property', 'og:title', title);
    upsertMeta('property', 'og:description', description);
    upsertMeta('property', 'og:image', imageUrl);
    upsertMeta('name', 'twitter:card', 'summary_large_image');
    upsertMeta('name', 'twitter:site', TWITTER_HANDLE);
    upsertMeta('name', 'twitter:title', title);
    upsertMeta('name', 'twitter:description', description);
    upsertMeta('name', 'twitter:image', imageUrl);
    upsertCanonical(canonicalUrl);
    upsertMeta('property', 'og:locale', 'en_US');
    upsertMeta('property', 'og:image:alt', title);
    upsertMeta('name', 'twitter:image:alt', title);
    if (noindex) upsertMeta('name', 'robots', 'noindex,nofollow');
    else removeMeta('name', 'robots');
    const graph = noindex ? [] : withBreadcrumbs(jsonLd, normalizedPath, breadcrumbName);
    upsertJsonLd(graph.length ? graph : undefined);
    return () => {
      removeMeta('name', 'robots');
    };
  }, [description, image, path, title, type, jsonLdKey, noindex, breadcrumbName]);

  return null;
}

function removeMeta(attrName: 'name' | 'property', attrValue: string) {
  document.head.querySelector(`meta[${attrName}="${attrValue}"]`)?.remove();
}

function upsertJsonLd(data?: object | object[]) {
  const id = 'seo-jsonld';
  const existing = document.getElementById(id);
  if (!data) {
    existing?.remove();
    return;
  }
  const script = (existing as HTMLScriptElement) || document.createElement('script');
  script.id = id;
  script.type = 'application/ld+json';
  script.textContent = JSON.stringify(data);
  if (!existing) document.head.appendChild(script);
}

function absoluteUrl(value: string) {
  if (/^https?:\/\//i.test(value)) return value;
  return `${BASE_URL}${value.startsWith('/') ? value : `/${value}`}`;
}

function upsertMeta(attrName: 'name' | 'property', attrValue: string, content: string) {
  let tag = document.head.querySelector(`meta[${attrName}="${attrValue}"]`) as HTMLMetaElement | null;
  if (!tag) {
    tag = document.createElement('meta');
    tag.setAttribute(attrName, attrValue);
    document.head.appendChild(tag);
  }
  tag.setAttribute('content', content);
}

function upsertCanonical(href: string) {
  let link = document.head.querySelector('link[rel="canonical"]') as HTMLLinkElement | null;
  if (!link) {
    link = document.createElement('link');
    link.setAttribute('rel', 'canonical');
    document.head.appendChild(link);
  }
  link.setAttribute('href', href);
}
