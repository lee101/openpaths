export const modelOgSlug = (id: string) => id.toLowerCase().replace(/[^a-z0-9.]+/g, '-').replace(/^-+|-+$/g, '');
export const modelOgPath = (id: string) => `/og/models/${modelOgSlug(id)}.png`;
