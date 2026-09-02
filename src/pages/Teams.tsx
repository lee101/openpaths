import React from 'react';
import { Link } from 'react-router-dom';
import { ArrowRight, Building2, KeyRound, Mail, ShieldAlert, Users } from 'lucide-react';
import { Seo } from '../components/Seo';

const faqJsonLd = {
  '@context': 'https://schema.org',
  '@type': 'FAQPage',
  mainEntity: [
    {
      '@type': 'Question',
      name: 'What are spend controls on OpenPaths?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Billing guards: a spend alert threshold plus a cap on top-ups, set per person or org-wide by owners and admins.',
      },
    },
    {
      '@type': 'Question',
      name: 'Can I restrict which models my team members can call?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Yes. Organization model rules use allow/deny patterns over model names and apply to every member. Rules are enforced at request time on both the OpenAI-compatible and Anthropic-native endpoints.',
      },
    },
    {
      '@type': 'Question',
      name: 'How do members join an organization?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Owners and admins invite people by email or share a join link of the form /orgs/your-org-slug/join. Invitations expire after 14 days, and new users can accept right after signing up.',
      },
    },
    {
      '@type': 'Question',
      name: 'Does OpenPaths charge extra for organizations?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'There is no separate organization fee. Any signed-in user can create an org, add members, and set rules; you pay only your normal usage credits.',
      },
    },
  ],
};

const features = [
  {
    icon: Users,
    title: 'Member invites and join links',
    body: 'Invite teammates by email or share a join link (/orgs/your-org/join). Invites expire after 14 days, roles cover owner, admin, member, and viewer, and admins can remove members at any time.',
  },
  {
    icon: ShieldAlert,
    title: 'Org-wide model rules',
    body: 'Allow or deny models by pattern for everyone in the org. Rules are checked on every request, so a denied model returns a clear permission error instead of a surprise line item.',
  },
  {
    icon: Building2,
    title: 'Billing guards',
    body: 'Set a spend alert threshold and a top-up cap for yourself or the whole org. Owners and admins control team-level guards from one place.',
  },
  {
    icon: KeyRound,
    title: 'Bring your own keys',
    body: 'Members can add their own provider keys for 18 providers, including OpenAI, Anthropic, Google, and OpenRouter. BYOK requests bypass the OpenPaths balance entirely, and org model rules still apply to them.',
  },
];

const audiences = [
  {
    name: 'Startups',
    body: 'One org, one set of spend controls. Everyone ships against the same catalog while founders keep alert thresholds and top-up caps under their control.',
  },
  {
    name: 'Agencies',
    body: 'Run a separate org per client. Restrict each org to the models the contract allows, so client A never calls what client B did not approve.',
  },
  {
    name: 'Enterprises',
    body: 'Hard top-up caps plus org-wide deny rules enforce policy in the request path, not in a document. Usage search over recorded responses covers audit trails.',
  },
];

const faqs = [
  {
    q: 'What are spend controls?',
    a: 'A spend alert threshold and a top-up cap, configurable per person or per org. When spend crosses your threshold you get notified, and caps bound how much credit can be added.',
  },
  {
    q: 'Can I restrict which models members use?',
    a: 'Yes. Org model rules accept allow/deny patterns over model names and are enforced on every request across both API formats.',
  },
  {
    q: 'How do joins work?',
    a: 'Owners and admins send email invites or share a join link (/orgs/slug/join). Recipients sign in with the invited email and accept; invites expire after 14 days.',
  },
  {
    q: 'Does it cost extra?',
    a: 'No separate org fee. Creating organizations, inviting members, and setting rules is part of a normal OpenPaths account; you pay only standard usage credits.',
  },
];

export function Teams() {
  return (
    <>
      <Seo
        title="Teams and Spend Controls | OpenPaths"
        description="OpenPaths organizations: pooled management for AI spend with member invites, org-wide model rules, billing guards, BYOK support, and per-member usage visibility."
        path="/teams"
        jsonLd={faqJsonLd}
      />
      <div className="mx-auto max-w-6xl px-6 py-16">
        <section className="mb-16">
          <div className="mb-4 inline-flex items-center gap-2 font-mono text-xs uppercase tracking-[0.22em] text-cyan-300">
            <Users className="h-4 w-4" />
            Teams
          </div>
          <h1 className="max-w-4xl text-4xl font-semibold tracking-tight md:text-6xl">
            One balance. Every model. Real controls.
          </h1>
          <p className="mt-5 max-w-3xl text-lg leading-relaxed text-white/60">
            Create an organization, invite your team, and manage AI spend from one place: org-wide
            model rules restrict which models members may call, billing guards put alert thresholds
            and top-up caps ahead of invoice surprises, and usage stays searchable per member.
          </p>
          <div className="mt-8 flex flex-wrap gap-3">
            <Link
              to="/account"
              className="inline-flex items-center gap-2 rounded-lg bg-white px-5 py-3 font-mono text-sm font-bold text-black hover:bg-white/90"
            >
              Set up your org <ArrowRight className="h-4 w-4" />
            </Link>
            <Link
              to="/docs"
              className="inline-flex items-center gap-2 rounded-lg border border-white/20 px-5 py-3 font-mono text-sm text-white/70 transition-colors hover:border-white/50 hover:text-white"
            >
              Read the docs
            </Link>
          </div>
        </section>

        <section className="mb-16">
          <h2 className="mb-8 font-mono text-sm uppercase tracking-[0.16em] text-white/65">
            What you get
          </h2>
          <div className="grid gap-5 md:grid-cols-2">
            {features.map(({ icon: Icon, title, body }) => (
              <div key={title} className="rounded-2xl border border-white/15 bg-white/[0.05] p-6">
                <Icon className="mb-4 h-5 w-5 text-cyan-300" />
                <h3 className="mb-2 text-lg font-semibold">{title}</h3>
                <p className="text-sm leading-relaxed text-white/60">{body}</p>
              </div>
            ))}
          </div>
          <p className="mt-6 text-sm leading-relaxed text-white/50">
            Members can also bring their own provider keys from Account settings. Keys are stored
            server-side and shown only as masked previews; requests made on them record zero cost
            against the OpenPaths balance while org model rules continue to apply.
          </p>
        </section>

        <section className="mb-16">
          <h2 className="mb-8 font-mono text-sm uppercase tracking-[0.16em] text-white/65">
            Built for how teams actually run
          </h2>
          <div className="divide-y divide-white/10 overflow-hidden rounded-2xl border border-white/15 bg-white/[0.04]">
            {audiences.map(({ name, body }) => (
              <div key={name} className="grid gap-2 p-6 md:grid-cols-[180px_1fr] md:gap-6">
                <div className="font-mono text-sm uppercase tracking-[0.14em] text-cyan-300">{name}</div>
                <p className="text-sm leading-relaxed text-white/60">{body}</p>
              </div>
            ))}
          </div>
        </section>

        <section className="mb-16">
          <h2 className="mb-8 font-mono text-sm uppercase tracking-[0.16em] text-white/65">FAQ</h2>
          <div className="space-y-6">
            {faqs.map(({ q, a }) => (
              <div key={q}>
                <h3 className="mb-1 font-semibold">{q}</h3>
                <p className="max-w-3xl text-sm leading-relaxed text-white/60">{a}</p>
              </div>
            ))}
          </div>
        </section>

        <section className="rounded-2xl border border-white/15 bg-white/[0.06] p-8">
          <h2 className="mb-3 text-2xl font-semibold">Roll out controls this afternoon</h2>
          <p className="mb-6 max-w-2xl text-sm leading-relaxed text-white/60">
            Create your org in Account settings, invite your team, and set model rules and billing
            guards before the next invoice cycle.
          </p>
          <div className="flex flex-wrap gap-3">
            <Link
              to="/account"
              className="inline-flex items-center gap-2 rounded-lg bg-white px-5 py-3 font-mono text-sm font-bold text-black hover:bg-white/90"
            >
              Open Account
            </Link>
            <Link
              to="/docs"
              className="inline-flex items-center gap-2 rounded-lg border border-white/20 px-5 py-3 font-mono text-sm text-white/70 transition-colors hover:border-white/50 hover:text-white"
            >
              Documentation
            </Link>
            <a
              href="mailto:support@openpaths.io"
              className="inline-flex items-center gap-2 rounded-lg border border-white/20 px-5 py-3 font-mono text-sm text-white/70 transition-colors hover:border-white/50 hover:text-white"
            >
              <Mail className="h-4 w-4" /> Contact us
            </a>
          </div>
        </section>
      </div>
    </>
  );
}
