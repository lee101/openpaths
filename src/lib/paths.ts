export function providerPath(slug: string) {
  return `/providers/${encodeURIComponent(slug)}`;
}

export function providerDocsPath(slug: string) {
  return `/${encodeURIComponent(slug)}/docs`;
}

export function modelPath(id: string) {
  return `/models/${encodeURIComponent(id)}`;
}
