import React, { useState, useMemo } from 'react';
import { models, Tag } from '../data/models';
import { Search, Tag as TagIcon, Cpu, Zap, Image as ImageIcon, Code2, BrainCircuit, MessageSquare, Globe } from 'lucide-react';
import { motion } from 'motion/react';

const ALL_TAGS: Tag[] = ['programming', 'roleplay', 'art generation', 'general', 'vision', 'fast', 'reasoning', 'open-source'];

const TAG_ICONS: Record<Tag, React.ReactNode> = {
  'programming': <Code2 className="w-3 h-3" />,
  'roleplay': <MessageSquare className="w-3 h-3" />,
  'art generation': <ImageIcon className="w-3 h-3" />,
  'general': <Globe className="w-3 h-3" />,
  'vision': <ImageIcon className="w-3 h-3" />,
  'fast': <Zap className="w-3 h-3" />,
  'reasoning': <BrainCircuit className="w-3 h-3" />,
  'open-source': <Cpu className="w-3 h-3" />
};

export function Models() {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedTags, setSelectedTags] = useState<Tag[]>([]);

  const toggleTag = (tag: Tag) => {
    setSelectedTags(prev => 
      prev.includes(tag) ? prev.filter(t => t !== tag) : [...prev, tag]
    );
  };

  const filteredModels = useMemo(() => {
    return models.filter(model => {
      const matchesSearch = model.name.toLowerCase().includes(searchQuery.toLowerCase()) || 
                            model.provider.toLowerCase().includes(searchQuery.toLowerCase()) ||
                            model.id.toLowerCase().includes(searchQuery.toLowerCase());
      
      const matchesTags = selectedTags.length === 0 || selectedTags.every(tag => model.tags.includes(tag));
      
      return matchesSearch && matchesTags;
    });
  }, [searchQuery, selectedTags]);

  return (
    <div className="max-w-7xl mx-auto px-6 py-12">
      <div className="mb-12">
        <h1 className="text-4xl font-bold tracking-tight mb-4">Model Directory</h1>
        <p className="text-white/60 text-lg max-w-2xl font-light">
          Explore our vast network of supported models. From high-speed open-source LLMs to state-of-the-art art generators.
        </p>
      </div>

      {/* Filters */}
      <div className="flex flex-col md:flex-row gap-6 mb-12">
        <div className="relative flex-1">
          <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-white/40" />
          <input 
            type="text" 
            placeholder="Search models by name, provider, or ID..." 
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full bg-white/5 border border-white/10 rounded-lg py-3 pl-12 pr-4 text-white placeholder:text-white/40 focus:outline-none focus:border-white/30 transition-colors font-mono text-sm"
          />
        </div>
        <div className="flex flex-wrap gap-2 items-center">
          <TagIcon className="w-4 h-4 text-white/40 mr-2" />
          {ALL_TAGS.map(tag => (
            <button
              key={tag}
              onClick={() => toggleTag(tag)}
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-full border text-xs font-mono transition-colors ${
                selectedTags.includes(tag) 
                  ? 'bg-white text-black border-white' 
                  : 'bg-transparent border-white/20 text-white/60 hover:border-white/40 hover:text-white'
              }`}
            >
              {TAG_ICONS[tag]}
              {tag}
            </button>
          ))}
          {selectedTags.length > 0 && (
            <button 
              onClick={() => setSelectedTags([])}
              className="text-xs font-mono text-white/40 hover:text-white ml-2 underline underline-offset-4"
            >
              Clear
            </button>
          )}
        </div>
      </div>

      {/* Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {filteredModels.map((model, idx) => (
          <motion.div 
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: idx * 0.05 }}
            key={model.id} 
            className="border border-white/10 bg-white/[0.02] rounded-xl p-6 hover:bg-white/[0.04] hover:border-white/20 transition-all group flex flex-col"
          >
            <div className="flex justify-between items-start mb-4">
              <div>
                <div className="text-xs font-mono text-white/40 mb-1">{model.provider}</div>
                <h3 className="text-xl font-bold tracking-tight">{model.name}</h3>
              </div>
              <div className="px-2 py-1 bg-white/10 rounded text-[10px] font-mono text-white/60">
                {model.contextLength !== 'N/A' ? `${model.contextLength} ctx` : 'Image'}
              </div>
            </div>
            
            <p className="text-sm text-white/60 font-light leading-relaxed mb-6 flex-1">
              {model.description}
            </p>

            <div className="space-y-4 mt-auto">
              <div className="flex flex-wrap gap-2">
                {model.tags.map(tag => (
                  <span key={tag} className="px-2 py-1 bg-white/5 border border-white/10 rounded text-[10px] font-mono text-white/60 flex items-center gap-1">
                    {TAG_ICONS[tag]} {tag}
                  </span>
                ))}
              </div>

              <div className="pt-4 border-t border-white/10 flex justify-between items-center text-xs font-mono">
                <div className="text-white/40">
                  <span className="text-white">${model.priceInput.toFixed(2)}</span> / 1M in
                </div>
                <div className="text-white/40">
                  <span className="text-white">${model.priceOutput.toFixed(2)}</span> / 1M out
                </div>
              </div>
              
              <div className="pt-2">
                <code className="block w-full bg-black border border-white/10 rounded px-3 py-2 text-[10px] text-white/40 truncate group-hover:text-white/80 transition-colors">
                  {model.id}
                </code>
              </div>
            </div>
          </motion.div>
        ))}
      </div>
      
      {filteredModels.length === 0 && (
        <div className="text-center py-24 border border-white/10 border-dashed rounded-xl">
          <Search className="w-8 h-8 text-white/20 mx-auto mb-4" />
          <h3 className="text-lg font-medium mb-2">No models found</h3>
          <p className="text-white/40 font-mono text-sm">Try adjusting your search or filters.</p>
        </div>
      )}
    </div>
  );
}
