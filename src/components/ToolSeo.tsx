import { Seo } from './Seo';
import { toolBySlug, toolOgImage } from '../data/tools';

/** Seo for a tool page, sourced from src/data/tools.ts so the client, the
 *  prerendered HTML, and the Go crawler meta all emit the same strings. */
export function ToolSeo({ slug }: { slug: string }) {
  const tool = toolBySlug.get(slug);
  if (!tool) return null;
  return <Seo title={tool.seoTitle} description={tool.seoDescription} path={tool.path} image={toolOgImage(slug)} />;
}
