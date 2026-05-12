import React, { useMemo } from 'react';
import { useParams, Link, Navigate, useLocation } from 'react-router-dom';
import { motion } from 'motion/react';
import { ArrowLeft, Calendar, Clock, User } from 'lucide-react';
import { posts } from '../data/blog';
import { CodeBlock } from '../components/CodeBlock';
import { Seo } from '../components/Seo';

function renderMarkdown(content: string): React.ReactNode[] {
  const lines = content.split('\n');
  const elements: React.ReactNode[] = [];
  let inCodeBlock = false;
  let codeLines: string[] = [];
  let codeLang = '';
  let inTable = false;
  let tableRows: string[][] = [];
  let key = 0;

  const flushTable = () => {
    if (tableRows.length < 2) return;
    const headers = tableRows[0];
    const body = tableRows.slice(2); // skip separator row
    elements.push(
      <div key={key++} className="overflow-x-auto my-6">
        <table className="w-full text-sm font-mono">
          <thead>
            <tr className="border-b border-white/20">
              {headers.map((h, i) => (
                <th key={i} className="text-left py-2 px-3 text-white/60 font-medium">{h.trim()}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {body.map((row, ri) => (
              <tr key={ri} className="border-b border-white/5">
                {row.map((cell, ci) => (
                  <td key={ci} className="py-2 px-3 text-white/50">{cell.trim()}</td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
    tableRows = [];
    inTable = false;
  };

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    if (line.startsWith('```')) {
      if (inTable) flushTable();
      if (inCodeBlock) {
        elements.push(
          <React.Fragment key={key++}>
            <CodeBlock
              code={codeLines.join('\n')}
              language={codeLang}
              label={codeLang}
              containerClassName="my-6 border border-white/10 rounded-lg overflow-hidden bg-black/40"
              headerClassName="bg-white/[0.03]"
              preClassName="text-sm"
            />
          </React.Fragment>
        );
        codeLines = [];
        codeLang = '';
        inCodeBlock = false;
      } else {
        inCodeBlock = true;
        codeLang = line.slice(3).trim();
      }
      continue;
    }

    if (inCodeBlock) {
      codeLines.push(line);
      continue;
    }

    // Table detection
    if (line.startsWith('|')) {
      const cells = line.split('|').filter((_, idx) => idx > 0 && idx < line.split('|').length - 1);
      if (!inTable) inTable = true;
      tableRows.push(cells);
      continue;
    } else if (inTable) {
      flushTable();
    }

    const imageMatch = line.match(/^!\[([^\]]*)\]\(([^)]+)\)$/);
    if (imageMatch) {
      elements.push(
        <figure key={key++} className="my-8">
          <img
            src={imageMatch[2]}
            alt={imageMatch[1]}
            loading="lazy"
            className="w-full h-auto rounded-2xl border border-white/10 bg-white/5"
          />
        </figure>
      );
      continue;
    }

    // Empty line
    if (line.trim() === '') {
      continue;
    }

    // Headers
    if (line.startsWith('## ')) {
      elements.push(<h2 key={key++} className="text-2xl font-bold tracking-tight mt-12 mb-4">{line.slice(3)}</h2>);
      continue;
    }
    if (line.startsWith('### ')) {
      elements.push(<h3 key={key++} className="text-xl font-bold tracking-tight mt-8 mb-3">{line.slice(4)}</h3>);
      continue;
    }

    // Paragraph with inline formatting
    elements.push(<p key={key++} className="text-white/60 leading-relaxed mb-4 font-light">{renderInline(line)}</p>);
  }

  if (inTable) flushTable();

  return elements;
}

function renderInline(text: string): React.ReactNode[] {
  const parts: React.ReactNode[] = [];
  let remaining = text;
  let k = 0;

  while (remaining.length > 0) {
    const linkMatch = remaining.match(/\[([^\]]+)\]\(([^)]+)\)/);
    // Bold
    const boldMatch = remaining.match(/\*\*(.+?)\*\*/);
    // Inline code
    const codeMatch = remaining.match(/`([^`]+)`/);

    let nextMatch: { idx: number; len: number; node: React.ReactNode; raw: string } | null = null;

    if (linkMatch && linkMatch.index !== undefined) {
      const href = linkMatch[2];
      const isExternal = /^https?:\/\//.test(href);
      const candidate = {
        idx: linkMatch.index,
        len: linkMatch[0].length,
        node: (
          <a
            key={`l${k++}`}
            href={href}
            className="text-white underline underline-offset-4 decoration-white/25 hover:decoration-white"
            target={isExternal ? '_blank' : undefined}
            rel={isExternal ? 'noreferrer noopener' : undefined}
          >
            {linkMatch[1]}
          </a>
        ),
        raw: linkMatch[0],
      };
      if (!nextMatch || candidate.idx < nextMatch.idx) nextMatch = candidate;
    }

    if (boldMatch && boldMatch.index !== undefined) {
      const candidate = { idx: boldMatch.index, len: boldMatch[0].length, node: <strong key={`b${k++}`} className="text-white font-medium">{boldMatch[1]}</strong>, raw: boldMatch[0] };
      if (!nextMatch || candidate.idx < nextMatch.idx) nextMatch = candidate;
    }
    if (codeMatch && codeMatch.index !== undefined) {
      const candidate = { idx: codeMatch.index, len: codeMatch[0].length, node: <code key={`c${k++}`} className="px-1.5 py-0.5 bg-white/10 rounded text-sm font-mono text-white/80">{codeMatch[1]}</code>, raw: codeMatch[0] };
      if (!nextMatch || candidate.idx < nextMatch.idx) nextMatch = candidate;
    }

    if (!nextMatch) {
      parts.push(remaining);
      break;
    }

    if (nextMatch.idx > 0) {
      parts.push(remaining.slice(0, nextMatch.idx));
    }
    parts.push(nextMatch.node);
    remaining = remaining.slice(nextMatch.idx + nextMatch.len);
  }

  return parts;
}

export function BlogPost() {
  const { slug } = useParams<{ slug: string }>();
  const location = useLocation();
  const post = useMemo(() => {
    if (location.pathname.startsWith('/alternatives/')) {
      return posts.find(p => p.alternativePath === location.pathname);
    }
    return posts.find(p => p.slug === slug);
  }, [location.pathname, slug]);

  if (!post) {
    return <Navigate to="/blog" replace />;
  }

  const rendered = useMemo(() => renderMarkdown(post.content), [post.content]);

  return (
    <>
      <Seo
        title={`${post.title} | OpenPaths Blog`}
        description={post.excerpt}
        path={location.pathname.startsWith('/alternatives/') ? location.pathname : `/blog/${post.slug}`}
      />

      <motion.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
        className="max-w-3xl mx-auto px-6 py-12"
      >
      <Link to="/blog" className="inline-flex items-center gap-2 text-sm font-mono text-white/40 hover:text-white transition-colors mb-12">
        <ArrowLeft className="w-4 h-4" /> All Posts
      </Link>

      <div className="flex flex-wrap gap-2 mb-6">
        {post.tags.map(tag => (
          <span key={tag} className="px-2 py-0.5 bg-white/5 border border-white/10 rounded text-[10px] font-mono text-white/50">
            {tag}
          </span>
        ))}
      </div>

      <h1 className="text-3xl md:text-4xl font-bold tracking-tight mb-6 leading-tight">{post.title}</h1>

      <div className="flex flex-wrap items-center gap-4 text-xs font-mono text-white/30 mb-12 pb-8 border-b border-white/10">
        <span className="flex items-center gap-1.5"><User className="w-3 h-3" />{post.author}</span>
        <span className="flex items-center gap-1.5"><Calendar className="w-3 h-3" />{post.date}</span>
        <span className="flex items-center gap-1.5"><Clock className="w-3 h-3" />{post.readTime}</span>
      </div>

      <article className="prose-invert">
        {rendered}
      </article>

      <div className="mt-16 pt-8 border-t border-white/10">
        <Link to="/blog" className="inline-flex items-center gap-2 text-sm font-mono text-white/40 hover:text-white transition-colors">
          <ArrowLeft className="w-4 h-4" /> Back to all posts
        </Link>
      </div>
      </motion.div>
    </>
  );
}
