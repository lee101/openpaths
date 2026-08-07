import React, { useEffect, useState } from 'react';
import { Eye, EyeOff, X } from 'lucide-react';
import { motion, AnimatePresence } from 'motion/react';
import { setApiKey } from '../lib/api';

export function AuthModal({ open, onClose, onSuccess }: { open: boolean; onClose: () => void; onSuccess?: () => void }) {
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [name, setName] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!open) return;
    setError('');
    setLoading(false);
  }, [open]);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const body = mode === 'register' ? { email, password, name } : { email, password };
      const res = await fetch(`/auth/${mode}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const data = await res.json();
      if (!res.ok) {
        setError(data.error?.message || 'Failed');
        return;
      }
	  const token = data.token || data.api_key;
	  if (data.api_key) setApiKey(data.api_key);
	  if (token) localStorage.setItem('op_token', token);
      localStorage.setItem('op_user', JSON.stringify(data.user));
	  window.userData = { id: data.user?.id, email: data.user?.email, name: data.user?.name, secret: token, authenticated: true };
      window.dispatchEvent(new Event('auth-change'));
      onSuccess?.();
      onClose();
    } catch {
      setError('Network error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4"
          data-testid="auth-modal-backdrop"
          onClick={e => {
            if (e.target === e.currentTarget) onClose();
          }}
        >
          <motion.div
            initial={{ opacity: 0, scale: 0.96, y: 10 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.96, y: 10 }}
            className="bg-[#090909] border border-white/10 rounded-3xl w-full max-w-md p-6 md:p-8 shadow-[0_24px_80px_rgba(0,0,0,0.45)]"
            data-testid="auth-modal"
          >
            <div className="flex items-start justify-between gap-4 mb-6">
              <div>
                <p className="text-xs font-mono uppercase tracking-[0.2em] text-white/35 mb-2">OpenPaths</p>
                <h2 className="text-2xl font-bold tracking-tight">{mode === 'login' ? 'Sign In' : 'Create Account'}</h2>
              </div>
              <button onClick={onClose} className="text-white/40 hover:text-white transition-colors shrink-0" data-testid="auth-modal-close">
                <X className="w-5 h-5" />
              </button>
            </div>
            <form onSubmit={submit} className="space-y-4">
              {mode === 'register' && (
                <input
                  value={name}
                  onChange={e => setName(e.target.value)}
                  placeholder="Name"
                  className="w-full bg-black border border-white/10 rounded-lg py-3 px-4 text-white font-mono focus:outline-none focus:border-white/30"
                />
              )}
              <input
                type="email"
                value={email}
                onChange={e => setEmail(e.target.value)}
                placeholder="Email"
                required
                className="w-full bg-black border border-white/10 rounded-lg py-3 px-4 text-white font-mono focus:outline-none focus:border-white/30"
                data-testid="auth-modal-email"
              />
              <div className="relative">
                <input
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={e => setPassword(e.target.value)}
                  placeholder="Password"
                  required
                  minLength={8}
                  className="w-full bg-black border border-white/10 rounded-lg py-3 px-4 pr-12 text-white font-mono focus:outline-none focus:border-white/30"
                  data-testid="auth-modal-password"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(v => !v)}
                  aria-label={showPassword ? 'Hide password' : 'Show password'}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-white/40 hover:text-white/70 transition-colors"
                >
                  {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
                </button>
              </div>
              {error && <p className="text-red-400 text-sm font-mono">{error}</p>}
              <button
                type="submit"
                disabled={loading}
                className="w-full bg-white text-black py-3 rounded-lg font-mono font-bold text-sm hover:bg-white/90 transition-colors disabled:opacity-50"
                data-testid="auth-modal-submit"
              >
                {loading ? 'Loading...' : mode === 'login' ? 'Sign In' : 'Create Account'}
              </button>
            </form>
            <button
              onClick={() => {
                setMode(mode === 'login' ? 'register' : 'login');
                setError('');
              }}
              className="mt-4 text-sm font-mono text-white/40 hover:text-white transition-colors"
              data-testid="auth-modal-toggle"
            >
              {mode === 'login' ? 'Need an account? Register' : 'Already have an account? Sign In'}
            </button>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
