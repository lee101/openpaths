import React, { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Eye, MessageSquare, Sparkles } from 'lucide-react';
import { saveConversation, type ConvMessage } from '../lib/conversations';
import { getApiKey, onAuthChange } from '../lib/api';
import { AuthModal } from '../components/AuthModal';

type SharedChatData = {
  slug: string;
  title: string;
  model: string;
  system_prompt: string;
  messages: { role: string; content: string }[];
  views: number;
  created_at: string;
};

export function SharedChat() {
  const { slug } = useParams();
  const navigate = useNavigate();
  const [data, setData] = useState<SharedChatData | null>(null);
  const [error, setError] = useState('');
  const [authOpen, setAuthOpen] = useState(false);
  const [hasKey, setHasKey] = useState(() => !!getApiKey());

  useEffect(() => onAuthChange(() => setHasKey(!!getApiKey())), []);

  useEffect(() => {
    if (!slug) return;
    setData(null);
    setError('');
    fetch(`/v1/chats/shared?slug=${encodeURIComponent(slug)}`)
      .then(async r => {
        const d = await r.json().catch(() => null);
        if (!r.ok || !d) throw new Error(d?.error?.message || 'not found');
        return d as SharedChatData;
      })
      .then(setData)
      .catch(() => setError('This shared chat could not be found.'));
  }, [slug]);

  function continueInPlayground() {
    if (!data) return;
    const messages: ConvMessage[] = (data.messages || [])
      .filter(m => m && typeof m.content === 'string' && (m.role === 'user' || m.role === 'assistant' || m.role === 'system'))
      .map(m => ({ role: m.role as ConvMessage['role'], content: m.content, createdAt: Date.now() }));
    const conv = saveConversation({
      title: data.title || undefined,
      model: data.model || 'auto',
      systemPrompt: data.system_prompt || undefined,
      messages,
    });
    navigate(`/playground?conv=${encodeURIComponent(conv.id)}`);
  }

  if (error) {
    return (
      <div className="max-w-3xl mx-auto px-6 py-24 text-center">
        <MessageSquare className="w-8 h-8 mx-auto mb-4 text-white/35" />
        <h1 className="text-2xl font-bold tracking-tight mb-2">Chat not found</h1>
        <p className="text-sm font-mono text-white/55">{error}</p>
      </div>
    );
  }

  if (!data) {
    return <div className="max-w-3xl mx-auto px-6 py-24 font-mono text-sm text-white/50">Loading shared chat...</div>;
  }

  const createdAt = data.created_at ? new Date(data.created_at) : null;

  return (
    <div className="max-w-3xl mx-auto px-4 sm:px-6 py-10">
      <div className="mb-8">
        <p className="text-xs font-mono uppercase tracking-[0.2em] text-white/50 mb-2">Shared chat</p>
        <h1 className="text-2xl md:text-3xl font-bold tracking-tight mb-3">{data.title || 'Shared conversation'}</h1>
        <div className="flex flex-wrap items-center gap-3 text-xs font-mono text-white/55">
          {data.model && (
            <span className="rounded border border-white/15 bg-white/[0.07] px-2 py-1 text-white/70" data-testid="shared-chat-model">
              {data.model}
            </span>
          )}
          <span className="flex items-center gap-1" data-testid="shared-chat-views">
            <Eye className="w-3.5 h-3.5" /> {(data.views || 0).toLocaleString()} views
          </span>
          {createdAt && !Number.isNaN(createdAt.getTime()) && (
            <span>{createdAt.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })}</span>
          )}
        </div>
      </div>

      {data.system_prompt && (
        <div className="mb-6 rounded-xl border border-white/20 bg-white/[0.05] px-4 py-3">
          <p className="text-[10px] font-mono uppercase tracking-[0.16em] text-white/45 mb-1">System prompt</p>
          <p className="text-sm text-white/60 whitespace-pre-wrap break-words">{data.system_prompt}</p>
        </div>
      )}

      <div className="space-y-4 mb-10" data-testid="shared-chat-messages">
        {(data.messages || []).map((m, i) => (
          m.role === 'user' ? (
            <div key={i} className="flex justify-end">
              <div className="max-w-[85%] bg-white/10 rounded-2xl rounded-br-sm px-4 py-2.5 text-sm leading-relaxed whitespace-pre-wrap break-words">
                {m.content}
              </div>
            </div>
          ) : (
            <div key={i} className="flex justify-start">
              <div className="max-w-[85%] rounded-2xl rounded-bl-sm border border-white/20 bg-white/[0.05] px-4 py-2.5 text-sm leading-relaxed text-white/90 whitespace-pre-wrap break-words">
                {m.content}
              </div>
            </div>
          )
        ))}
      </div>

      <div className="flex flex-wrap items-center gap-3 border-t border-white/20 pt-6">
        <button
          onClick={continueInPlayground}
          className="bg-white text-black px-5 py-2.5 rounded-lg font-mono font-bold text-sm hover:bg-white/90 transition-colors flex items-center gap-2"
          data-testid="shared-chat-continue"
        >
          <MessageSquare className="w-4 h-4" /> Continue in Playground
        </button>
        {!hasKey && (
          <button
            onClick={() => setAuthOpen(true)}
            className="border border-white/15 px-5 py-2.5 rounded-lg font-mono text-sm text-white/70 hover:text-white hover:border-white/50 transition-colors flex items-center gap-2"
            data-testid="shared-chat-signup"
          >
            <Sparkles className="w-4 h-4" /> Try OpenPaths free
          </button>
        )}
      </div>

      <AuthModal open={authOpen} onClose={() => setAuthOpen(false)} onSuccess={() => setAuthOpen(false)} />
    </div>
  );
}
