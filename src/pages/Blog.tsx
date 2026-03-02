import React from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'motion/react';
import { ArrowRight, Calendar, Clock } from 'lucide-react';
import { posts } from '../data/blog';

export function Blog() {
  return (
    <div className="max-w-4xl mx-auto px-6 py-12">
      <div className="mb-16">
        <h1 className="text-4xl md:text-5xl font-bold tracking-tight mb-4">Blog</h1>
        <p className="text-white/60 text-lg font-light max-w-2xl">
          Engineering deep-dives, model comparisons, and guides for building with OpenPath.
        </p>
      </div>

      <div className="space-y-1">
        {posts.map((post, idx) => (
          <motion.div
            key={post.slug}
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: idx * 0.05 }}
          >
            <Link
              to={`/blog/${post.slug}`}
              className="block border border-white/10 bg-white/[0.02] rounded-lg p-8 hover:bg-white/[0.04] hover:border-white/20 transition-all group"
            >
              <div className="flex flex-wrap gap-2 mb-4">
                {post.tags.map(tag => (
                  <span key={tag} className="px-2 py-0.5 bg-white/5 border border-white/10 rounded text-[10px] font-mono text-white/50">
                    {tag}
                  </span>
                ))}
              </div>

              <h2 className="text-xl md:text-2xl font-bold tracking-tight mb-3 group-hover:text-white transition-colors">
                {post.title}
              </h2>

              <p className="text-white/50 text-sm leading-relaxed mb-6 font-light">
                {post.excerpt}
              </p>

              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4 text-xs font-mono text-white/30">
                  <span className="flex items-center gap-1.5"><Calendar className="w-3 h-3" />{post.date}</span>
                  <span className="flex items-center gap-1.5"><Clock className="w-3 h-3" />{post.readTime}</span>
                </div>
                <span className="text-sm font-mono text-white/40 group-hover:text-white transition-colors flex items-center gap-1">
                  Read <ArrowRight className="w-3 h-3" />
                </span>
              </div>
            </Link>
          </motion.div>
        ))}
      </div>
    </div>
  );
}
