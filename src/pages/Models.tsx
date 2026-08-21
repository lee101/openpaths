import React, { useState, useMemo, useEffect } from 'react';
import { Link, useSearchParams, useNavigate } from 'react-router-dom';
import { models, Tag, SortOption, parseContextLength } from '../data/models';
import { providersByName, getProviderLogo } from '../data/providers';
import { Search, Tag as TagIcon, Cpu, Zap, Image as ImageIcon, Code2, BrainCircuit, MessageSquare, Globe, ArrowUpDown, Video, Gift, Database, AudioLines, Box } from 'lucide-react';
import { motion } from 'motion/react';
import { Seo } from '../components/Seo';
import { modelPath, providerPath } from '../lib/paths';

const ALL_TAGS: Tag[] = ['programming', 'reasoning', 'agentic', 'general', 'vision', 'fast', 'audio', 'embedding', 'open-source', 'free', 'art generation', 'text-to-image', 'image-to-image', 'text-to-video', 'image-to-video', 'video-to-video', 'image-to-3d', 'outpainting', 'video generation', 'forecasting', 'roleplay'];

const TAG_ICONS: Record<Tag, React.ReactNode> = {
  'programming': <Code2 className="w-3 h-3" />,
  'roleplay': <MessageSquare className="w-3 h-3" />,
  'art generation': <ImageIcon className="w-3 h-3" />,
  'text-to-image': <ImageIcon className="w-3 h-3" />,
  'image-to-image': <ImageIcon className="w-3 h-3" />,
  'image-to-3d': <Box className="w-3 h-3" />,
  'outpainting': <ImageIcon className="w-3 h-3" />,
  'video generation': <Video className="w-3 h-3" />,
  'text-to-video': <Video className="w-3 h-3" />,
  'image-to-video': <Video className="w-3 h-3" />,
  'video-to-video': <Video className="w-3 h-3" />,
  'audio': <AudioLines className="w-3 h-3" />,
  'embedding': <Database className="w-3 h-3" />,
  'forecasting': <ArrowUpDown className="w-3 h-3" />,
  'general': <Globe className="w-3 h-3" />,
  'vision': <ImageIcon className="w-3 h-3" />,
  'fast': <Zap className="w-3 h-3" />,
  'reasoning': <BrainCircuit className="w-3 h-3" />,
  'agentic': <BrainCircuit className="w-3 h-3" />,
  'open-source': <Cpu className="w-3 h-3" />,
  'free': <Gift className="w-3 h-3" />
};

const SORT_OPTIONS: { value: SortOption; label: string }[] = [
  { value: 'popular', label: 'Most Popular' },
  { value: 'newest', label: 'Newest' },
  { value: 'price-low', label: 'Price: Low to High' },
  { value: 'price-high', label: 'Price: High to Low' },
  { value: 'context-high', label: 'Context: High to Low' },
];

function sortModels(items: typeof models, sort: SortOption) {
  const sorted = [...items];
  switch (sort) {
    case 'popular':
      return sorted.sort((a, b) => a.popularity - b.popularity);
    case 'newest':
      return sorted.sort((a, b) => b.released.localeCompare(a.released));
    case 'price-low':
      return sorted.sort((a, b) => (a.priceInput + a.priceOutput) - (b.priceInput + b.priceOutput));
    case 'price-high':
      return sorted.sort((a, b) => (b.priceInput + b.priceOutput) - (a.priceInput + a.priceOutput));
    case 'context-high':
      return sorted.sort((a, b) => parseContextLength(b.contextLength) - parseContextLength(a.contextLength));
    default:
      return sorted;
  }
}

const CHAT_TAGS: Tag[] = ['art generation', 'text-to-image', 'image-to-image', 'image-to-3d', 'outpainting', 'video generation', 'audio', 'embedding'];
function isChatModel(model: typeof models[0]) {
  return !model.tags.every(t => CHAT_TAGS.includes(t)) && model.contextLength !== 'N/A';
}

function isImageGenerationModel(model: typeof models[0]) {
  return model.tags.includes('art generation');
}

function formatTokenPrice(price: number) {
  const precision = Math.abs(price * 100 - Math.round(price * 100)) > 1e-9 ? 3 : 2;
  return price.toFixed(precision);
}

export function Models() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [searchQuery, setSearchQuery] = useState(searchParams.get('q') || '');
  const [selectedTags, setSelectedTags] = useState<Tag[]>([]);
  const [sortBy, setSortBy] = useState<SortOption>('popular');

  useEffect(() => {
    const q = searchParams.get('q');
    if (q) setSearchQuery(q);
  }, [searchParams]);

  const toggleTag = (tag: Tag) => {
    setSelectedTags(prev =>
      prev.includes(tag) ? prev.filter(t => t !== tag) : [...prev, tag]
    );
  };

  const filteredModels = useMemo(() => {
    const filtered = models.filter(model => {
      const normalizedQuery = searchQuery.toLowerCase().replace(/\s+/g, '-');
      const tagText = model.tags.join(' ');
      const tagTextWithSpaces = model.tags.join(' ').replace(/-/g, ' ');
      const matchesSearch = model.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
                            model.provider.toLowerCase().includes(searchQuery.toLowerCase()) ||
                            model.id.toLowerCase().includes(searchQuery.toLowerCase()) ||
                            (model.aliases || []).some(alias => alias.toLowerCase().includes(searchQuery.toLowerCase()) || alias.toLowerCase().includes(normalizedQuery)) ||
                            tagText.includes(normalizedQuery) ||
                            tagTextWithSpaces.includes(searchQuery.toLowerCase());

      const matchesTags = selectedTags.length === 0 || selectedTags.every(tag => model.tags.includes(tag));

      return matchesSearch && matchesTags;
    });
    return sortModels(filtered, sortBy);
  }, [searchQuery, selectedTags, sortBy]);

  const jsonLd = useMemo(() => ({
    '@context': 'https://schema.org',
    '@type': 'CollectionPage',
    name: 'AI Model Directory',
    url: 'https://openpaths.io/models',
    description: `Compare ${models.length}+ AI models by provider, price, context window, and capability.`,
    mainEntity: {
      '@type': 'ItemList',
      numberOfItems: models.length,
      itemListElement: [...models]
        .sort((a, b) => a.popularity - b.popularity)
        .slice(0, 30)
        .map((model, i) => ({
          '@type': 'ListItem',
          position: i + 1,
          url: `https://openpaths.io/models/${encodeURIComponent(model.id)}`,
          name: model.name,
        })),
    },
  }), []);

  return (
    <>
      <Seo
        title="AI Model Directory | OpenPaths"
        description={`Compare ${models.length}+ AI models by provider, price, context window, and capability across chat, image, video, audio, and embedding APIs.`}
        path="/models"
        jsonLd={jsonLd}
      />

      <div className="max-w-7xl mx-auto px-6 py-12">
      <div className="mb-12">
        <h1 className="text-4xl font-bold tracking-tight mb-4">Model Directory</h1>
        <p className="text-white/60 text-lg max-w-2xl font-light">
          Access {models.length}+ models from <Link to="/providers" className="underline underline-offset-4 hover:text-white transition-colors">leading providers</Link>. From frontier LLMs to image and video generators.
        </p>
      </div>

      {/* Filters */}
      <div className="flex flex-col gap-6 mb-12">
        <div className="flex flex-col md:flex-row gap-4">
          <div className="relative flex-1">
            <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-white/55" />
            <input
              type="text"
              placeholder="Search models by name, provider, or ID..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full bg-white/10 border border-white/20 rounded-lg py-3 pl-12 pr-4 text-white placeholder:text-white/45 focus:outline-none focus:border-white/50 transition-colors font-mono text-sm"
            />
          </div>
          <div className="relative">
            <ArrowUpDown className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-white/55 pointer-events-none" />
            <select
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value as SortOption)}
              className="appearance-none bg-white/10 border border-white/20 rounded-lg py-3 pl-9 pr-8 text-white text-sm font-mono focus:outline-none focus:border-white/50 transition-colors cursor-pointer"
            >
              {SORT_OPTIONS.map(opt => (
                <option key={opt.value} value={opt.value} className="bg-black text-white">
                  {opt.label}
                </option>
              ))}
            </select>
          </div>
        </div>
        <div className="flex flex-wrap gap-2 items-center">
          <TagIcon className="w-4 h-4 text-white/55 mr-2" />
          {ALL_TAGS.map(tag => (
            <button
              key={tag}
              onClick={() => toggleTag(tag)}
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-full border text-xs font-mono transition-colors ${
                selectedTags.includes(tag)
                  ? 'bg-white text-black border-white'
                  : 'bg-transparent border-white/20 text-white/60 hover:border-white/60 hover:text-white'
              }`}
            >
              {TAG_ICONS[tag]}
              {tag}
            </button>
          ))}
          {selectedTags.length > 0 && (
            <button
              onClick={() => setSelectedTags([])}
              className="text-xs font-mono text-white/55 hover:text-white ml-2 underline underline-offset-4"
            >
              Clear
            </button>
          )}
        </div>
      </div>

      {/* Count */}
      <div className="mb-6 text-sm font-mono text-white/55">
        {filteredModels.length} model{filteredModels.length !== 1 ? 's' : ''}
      </div>

      {/* Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {filteredModels.map((model, idx) => (
          <motion.div
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: idx * 0.05 }}
            key={model.id}
            className="border border-white/20 bg-white/[0.05] rounded-xl p-6 hover:bg-white/[0.07] hover:border-white/40 transition-all group flex flex-col"
          >
            <div className="flex justify-between items-start mb-4">
              <div>
                <div className="text-xs font-mono text-white/55 mb-1 flex items-center gap-1.5">
                  <img src={getProviderLogo(model.provider)} alt="" className={`w-4 h-4 rounded-sm object-contain ${model.provider === 'Black Forest Labs' ? 'bg-white p-px' : ''}`} />
                  {providersByName[model.provider] ? (
                    <Link to={providerPath(providersByName[model.provider].slug)} className="hover:text-white transition-colors underline underline-offset-2 decoration-white/20">
                      {model.provider}
                    </Link>
                  ) : model.provider}
                </div>
                <Link to={modelPath(model.id)} className="text-xl font-bold tracking-tight hover:underline underline-offset-4">
                  {model.name}
                </Link>
              </div>
              <div className="px-2 py-1 bg-white/10 rounded text-[10px] font-mono text-white/60">
                {model.contextLength !== 'N/A' ? `${model.contextLength} ctx` : model.tags.includes('video generation') ? 'Video' : model.tags.includes('audio') ? 'Audio' : 'Image'}
              </div>
            </div>

            <p className="text-sm text-white/60 font-light leading-relaxed mb-6 flex-1">
              {model.description}
            </p>

            <div className="space-y-4 mt-auto">
              <div className="flex flex-wrap gap-2">
                {model.tags.map(tag => (
                  <span key={tag} className="px-2 py-1 bg-white/10 border border-white/20 rounded text-[10px] font-mono text-white/60 flex items-center gap-1">
                    {TAG_ICONS[tag]} {tag}
                  </span>
                ))}
              </div>

              <div className="pt-4 border-t border-white/20 flex justify-between items-center text-xs font-mono">
                {model.pricingType === 'request' ? (
                  <div className="text-white/55">
                    <span className="text-white">${model.priceInput < 0.01 ? model.priceInput.toFixed(3) : model.priceInput.toFixed(2)}</span> / request
                  </div>
                ) : model.pricingType === 'chars' ? (
                  <div className="text-white/55">
                    <span className="text-white">${model.priceInput.toFixed(2)}</span> / 1M chars
                  </div>
                ) : model.pricingType === 'hour' ? (
                  <div className="text-white/55">
                    <span className="text-white">${model.priceInput.toFixed(2)}</span> / hour
                  </div>
                ) : model.pricingType === 'second' ? (
                  <div className="text-white/55">
                    <span className="text-white">${model.priceInput.toFixed(2)}</span> / second
                  </div>
                ) : model.pricingType === 'megapixel' ? (
                  <div className="text-white/55">
                    <span className="text-white">${model.priceInput.toFixed(3)}</span> / MP
                  </div>
                ) : (
                  <>
                    <div className="text-white/55">
                      <span className="text-white">${formatTokenPrice(model.priceInput)}</span> / 1M in
                    </div>
                    <div className="text-white/55">
                      <span className="text-white">${formatTokenPrice(model.priceOutput)}</span> / 1M out
                    </div>
                  </>
                )}
              </div>

              <div className="pt-2 flex items-center gap-2">
                <code className="flex-1 bg-white/[0.06] border border-white/30 rounded px-3 py-2 text-[10px] text-white/55 truncate group-hover:text-white/80 transition-colors">
                  {model.id}
                </code>
                {isChatModel(model) && (
                  <button
                    onClick={() => navigate(`/playground?model=${encodeURIComponent(model.id)}`)}
                    title="Chat with this model"
                    className="shrink-0 flex items-center gap-1 px-2.5 py-2 bg-white/10 border border-white/20 rounded text-[10px] font-mono text-white/50 hover:text-white hover:bg-white/10 hover:border-white/45 transition-colors"
                  >
                    <MessageSquare className="w-3 h-3" /> Chat
                  </button>
                )}
                {isImageGenerationModel(model) && (
                  <button
                    onClick={() => navigate(`/playground?model=${encodeURIComponent(model.id)}&mode=image`)}
                    title="Generate images with this model"
                    className="shrink-0 flex items-center gap-1 px-2.5 py-2 bg-white/10 border border-white/20 rounded text-[10px] font-mono text-white/50 hover:text-white hover:bg-white/10 hover:border-white/45 transition-colors"
                  >
                    <ImageIcon className="w-3 h-3" /> Generate
                  </button>
                )}
              </div>
            </div>
          </motion.div>
        ))}
      </div>

      {filteredModels.length === 0 && (
        <div className="text-center py-24 border border-white/20 border-dashed rounded-xl">
          <Search className="w-8 h-8 text-white/35 mx-auto mb-4" />
          <h3 className="text-lg font-medium mb-2">No models found</h3>
          <p className="text-white/55 font-mono text-sm">Try adjusting your search or filters.</p>
        </div>
      )}
      </div>
    </>
  );
}
