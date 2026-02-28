import React, { useState } from 'react';
import { CreditCard, Key, Wallet, Plus, Copy, Check, Activity, ArrowUpRight } from 'lucide-react';
import { motion } from 'motion/react';

export function Account() {
  const [activeTab, setActiveTab] = useState<'overview' | 'keys' | 'billing'>('overview');
  const [copied, setCopied] = useState(false);

  const copyKey = () => {
    navigator.clipboard.writeText('op_live_8f92a1b3c4d5e6f7g8h9i0j1k2l3m4n5');
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="max-w-6xl mx-auto px-6 py-12 flex flex-col md:flex-row gap-12">
      {/* Sidebar */}
      <aside className="w-full md:w-64 shrink-0">
        <div className="mb-8">
          <h2 className="text-xl font-bold tracking-tight mb-1">Account</h2>
          <p className="text-sm font-mono text-white/40">user@example.com</p>
        </div>
        
        <nav className="flex flex-col gap-2 font-mono text-sm">
          <button 
            onClick={() => setActiveTab('overview')}
            className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors text-left ${activeTab === 'overview' ? 'bg-white/10 text-white' : 'text-white/60 hover:bg-white/5 hover:text-white'}`}
          >
            <Activity className="w-4 h-4" /> Overview
          </button>
          <button 
            onClick={() => setActiveTab('keys')}
            className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors text-left ${activeTab === 'keys' ? 'bg-white/10 text-white' : 'text-white/60 hover:bg-white/5 hover:text-white'}`}
          >
            <Key className="w-4 h-4" /> API Keys
          </button>
          <button 
            onClick={() => setActiveTab('billing')}
            className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors text-left ${activeTab === 'billing' ? 'bg-white/10 text-white' : 'text-white/60 hover:bg-white/5 hover:text-white'}`}
          >
            <CreditCard className="w-4 h-4" /> Billing
          </button>
        </nav>
      </aside>

      {/* Main Content */}
      <main className="flex-1 min-w-0">
        {activeTab === 'overview' && (
          <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}>
            <h1 className="text-3xl font-bold tracking-tight mb-8">Overview</h1>
            
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-6 mb-12">
              <div className="border border-white/10 bg-white/[0.02] rounded-xl p-6">
                <div className="text-sm font-mono text-white/40 mb-2">Current Balance</div>
                <div className="text-4xl font-light tracking-tight mb-4">$42.50</div>
                <button onClick={() => setActiveTab('billing')} className="text-xs font-mono text-white border border-white/20 px-3 py-1.5 rounded hover:bg-white/10 transition-colors">
                  Add Funds
                </button>
              </div>
              <div className="border border-white/10 bg-white/[0.02] rounded-xl p-6">
                <div className="text-sm font-mono text-white/40 mb-2">Requests (30d)</div>
                <div className="text-4xl font-light tracking-tight mb-4">1.2M</div>
                <div className="text-xs font-mono text-green-400 flex items-center gap-1">
                  <ArrowUpRight className="w-3 h-3" /> +14% from last month
                </div>
              </div>
            </div>

            <h2 className="text-xl font-bold tracking-tight mb-4">Recent Activity</h2>
            <div className="border border-white/10 rounded-xl overflow-hidden">
              <table className="w-full text-left text-sm">
                <thead className="bg-white/5 font-mono text-xs text-white/40 border-b border-white/10">
                  <tr>
                    <th className="px-6 py-3 font-normal">Model</th>
                    <th className="px-6 py-3 font-normal">Requests</th>
                    <th className="px-6 py-3 font-normal">Cost</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-white/10 font-mono">
                  <tr>
                    <td className="px-6 py-4">anthropic/claude-3.5-sonnet</td>
                    <td className="px-6 py-4 text-white/60">45,210</td>
                    <td className="px-6 py-4">$12.40</td>
                  </tr>
                  <tr>
                    <td className="px-6 py-4">openai/gpt-4o</td>
                    <td className="px-6 py-4 text-white/60">12,050</td>
                    <td className="px-6 py-4">$8.20</td>
                  </tr>
                  <tr>
                    <td className="px-6 py-4">meta-llama/llama-3.1-70b-instruct</td>
                    <td className="px-6 py-4 text-white/60">89,400</td>
                    <td className="px-6 py-4">$2.15</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </motion.div>
        )}

        {activeTab === 'keys' && (
          <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}>
            <div className="flex justify-between items-center mb-8">
              <h1 className="text-3xl font-bold tracking-tight">API Keys</h1>
              <button className="bg-white text-black px-4 py-2 text-sm font-mono font-bold hover:bg-white/90 transition-colors flex items-center gap-2 rounded">
                <Plus className="w-4 h-4" /> Create Key
              </button>
            </div>

            <div className="border border-white/10 bg-white/[0.02] rounded-xl p-6 mb-8">
              <div className="flex justify-between items-start mb-4">
                <div>
                  <h3 className="font-bold mb-1">Default Project Key</h3>
                  <p className="text-xs font-mono text-white/40">Created on Oct 24, 2023</p>
                </div>
                <div className="px-2 py-1 bg-green-500/10 text-green-400 text-[10px] font-mono rounded border border-green-500/20">Active</div>
              </div>
              <div className="flex items-center gap-4 bg-black border border-white/10 rounded-lg p-3">
                <code className="flex-1 font-mono text-sm text-white/80">op_live_8f92a1b3c4d5e6f7g8h9i0j1k2l3m4n5</code>
                <button 
                  onClick={copyKey}
                  className="p-2 hover:bg-white/10 rounded transition-colors text-white/60 hover:text-white"
                  title="Copy to clipboard"
                >
                  {copied ? <Check className="w-4 h-4 text-green-400" /> : <Copy className="w-4 h-4" />}
                </button>
              </div>
            </div>

            <p className="text-sm text-white/40 font-light">
              Do not share your API key in publicly accessible areas such as GitHub, client-side code, and so forth.
            </p>
          </motion.div>
        )}

        {activeTab === 'billing' && (
          <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}>
            <h1 className="text-3xl font-bold tracking-tight mb-8">Billing & Payments</h1>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-12">
              {/* Stripe Card */}
              <div className="border border-white/10 bg-white/[0.02] rounded-xl p-6 flex flex-col">
                <div className="mb-6">
                  <CreditCard className="w-8 h-8 text-white/60 mb-4" />
                  <h3 className="text-xl font-bold mb-2">Credit Card</h3>
                  <p className="text-sm text-white/60 font-light">Add funds securely using Stripe. Supports all major credit cards.</p>
                </div>
                <div className="mt-auto">
                  <button className="w-full bg-white text-black px-4 py-3 text-sm font-mono font-bold hover:bg-white/90 transition-colors rounded">
                    Add Funds with Stripe
                  </button>
                </div>
              </div>

              {/* Solana Card */}
              <div className="border border-[#14F195]/30 bg-[#14F195]/5 rounded-xl p-6 flex flex-col relative overflow-hidden">
                <div className="absolute -right-10 -top-10 w-32 h-32 bg-[#9945FF]/20 blur-3xl rounded-full pointer-events-none" />
                <div className="mb-6 relative z-10">
                  <svg className="w-8 h-8 text-[#14F195] mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M14.5 3h-10l-2 5h10l2-5Z"/><path d="M11.5 11h-10l-2 5h10l2-5Z"/><path d="M14.5 19h-10l-2 5h10l2-5Z"/></svg>
                  <h3 className="text-xl font-bold mb-2">Solana Payment</h3>
                  <p className="text-sm text-white/60 font-light">Pay instantly with SOL or USDC on the Solana network. Zero wait times.</p>
                </div>
                <div className="mt-auto relative z-10">
                  <button className="w-full bg-gradient-to-r from-[#9945FF] to-[#14F195] text-black px-4 py-3 text-sm font-mono font-bold hover:opacity-90 transition-opacity rounded flex items-center justify-center gap-2">
                    <Wallet className="w-4 h-4" /> Connect Wallet
                  </button>
                </div>
              </div>
            </div>

            <h2 className="text-xl font-bold tracking-tight mb-4">Payment History</h2>
            <div className="border border-white/10 rounded-xl overflow-hidden">
              <table className="w-full text-left text-sm">
                <thead className="bg-white/5 font-mono text-xs text-white/40 border-b border-white/10">
                  <tr>
                    <th className="px-6 py-3 font-normal">Date</th>
                    <th className="px-6 py-3 font-normal">Method</th>
                    <th className="px-6 py-3 font-normal">Amount</th>
                    <th className="px-6 py-3 font-normal">Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-white/10 font-mono">
                  <tr>
                    <td className="px-6 py-4 text-white/60">Oct 24, 2023</td>
                    <td className="px-6 py-4">Solana (USDC)</td>
                    <td className="px-6 py-4">$50.00</td>
                    <td className="px-6 py-4 text-green-400">Completed</td>
                  </tr>
                  <tr>
                    <td className="px-6 py-4 text-white/60">Sep 12, 2023</td>
                    <td className="px-6 py-4">Stripe (...4242)</td>
                    <td className="px-6 py-4">$25.00</td>
                    <td className="px-6 py-4 text-green-400">Completed</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </motion.div>
        )}
      </main>
    </div>
  );
}
