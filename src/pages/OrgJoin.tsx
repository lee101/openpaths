import React, { useCallback, useEffect, useState } from 'react';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import { AuthModal } from '../components/AuthModal';

type JoinState = 'joining' | 'auth' | 'success' | 'error';

export function OrgJoin() {
  const { slug = '' } = useParams();
  const [params] = useSearchParams();
  const token = params.get('token') || '';
  const [state, setState] = useState<JoinState>('joining');
  const [message, setMessage] = useState('Accepting your invitation…');
  const [authOpen, setAuthOpen] = useState(false);

  const accept = useCallback(async () => {
    if (!slug || !token) {
      setState('error');
      setMessage('This invitation link is incomplete. Ask the organization owner for a new invitation.');
      return;
    }
    setState('joining');
    setMessage('Accepting your invitation…');
    try {
      const response = await fetch(`/account/orgs/${encodeURIComponent(slug)}/join?token=${encodeURIComponent(token)}`, {
        method: 'POST',
        credentials: 'same-origin',
      });
      const body = await response.json().catch(() => ({}));
      if (response.status === 401) {
        setState('auth');
        setMessage('Sign in or create an account with the invited email address to continue.');
        return;
      }
      if (!response.ok) {
        setState('error');
        setMessage(body.error?.message || 'This invitation could not be accepted.');
        return;
      }
      setState('success');
      setMessage(`You joined ${body.org?.name || slug}.`);
    } catch {
      setState('error');
      setMessage('Could not reach OpenPaths. Please try again.');
    }
  }, [slug, token]);

  useEffect(() => {
    void accept();
  }, [accept]);

  return (
    <div className="mx-auto flex min-h-[65vh] max-w-xl items-center px-6 py-16">
      <div className="w-full rounded-3xl border border-white/20 bg-white/[0.06] p-8 text-center shadow-2xl">
        <p className="mb-3 font-mono text-xs uppercase tracking-[0.22em] text-white/50">Organization invitation</p>
        <h1 className="mb-4 text-3xl font-bold tracking-tight">
          {state === 'success' ? 'Invitation accepted' : 'Join OpenPaths'}
        </h1>
        <p className={state === 'error' ? 'text-red-300' : 'text-white/60'}>{message}</p>
        <div className="mt-8 flex flex-wrap justify-center gap-3">
          {state === 'auth' && (
            <button onClick={() => setAuthOpen(true)} className="rounded-lg bg-white px-5 py-3 font-mono text-sm font-bold text-black hover:bg-white/90">
              Sign in to accept
            </button>
          )}
          {state === 'error' && (
            <button onClick={() => void accept()} className="rounded-lg bg-white px-5 py-3 font-mono text-sm font-bold text-black hover:bg-white/90">
              Try again
            </button>
          )}
          {state === 'success' && <Link to="/account" className="rounded-lg bg-white px-5 py-3 font-mono text-sm font-bold text-black hover:bg-white/90">Open account</Link>}
        </div>
      </div>
      <AuthModal open={authOpen} onClose={() => setAuthOpen(false)} onSuccess={() => void accept()} />
    </div>
  );
}
