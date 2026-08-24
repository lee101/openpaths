import React, { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { ArrowRight, Calculator as CalculatorIcon, KeyRound, Table2 } from 'lucide-react';
import { calculatorModels, defaultSelectionIds, formatUsd } from '../data/calculator';
import type { Model } from '../data/models';
import { Seo } from '../components/Seo';

const TOKEN_PRESETS = [
  { label: '1M', value: 1_000_000 },
  { label: '10M', value: 10_000_000 },
  { label: '100M', value: 100_000_000 },
];

const MAX_SELECTION = 4;

function parseTokens(raw: string): number {
  const parsed = Number.parseFloat(raw.replace(/[^0-9.]/g, ''));
  if (!Number.isFinite(parsed) || parsed < 0) return 0;
  return Math.min(parsed, 1_000_000_000);
}

export function Calculator() {
  const [inputTokens, setInputTokens] = useState('10,000,000');
  const [outputTokens, setOutputTokens] = useState('2,500,000');
  const [selectedIds, setSelectedIds] = useState<string[]>(defaultSelectionIds);

  const monthlyInput = parseTokens(inputTokens);
  const monthlyOutput = parseTokens(outputTokens);

  const selectedModels = useMemo(
    () => selectedIds.map(id => calculatorModels.find(model => model.id === id)).filter((m): m is Model => Boolean(m)),
    [selectedIds]
  );

  const rows = useMemo(
    () =>
      selectedModels.map(model => ({
        model,
        inputCost: (monthlyInput / 1_000_000) * model.priceInput,
        outputCost: (monthlyOutput / 1_000_000) * model.priceOutput,
      })),
    [selectedModels, monthlyInput, monthlyOutput]
  );

  const maxTotal = rows.reduce((max, row) => Math.max(max, row.inputCost + row.outputCost), 0);

  function toggleModel(id: string) {
    setSelectedIds(prev => {
      if (prev.includes(id)) return prev.filter(existing => existing !== id);
      if (prev.length >= MAX_SELECTION) return prev;
      return [...prev, id];
    });
  }

  const faqJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: [
      {
        '@type': 'Question',
        name: 'How does the OpenPaths cost calculator work?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'It multiplies your monthly input and output token volume by each model\'s published per-1M-token price from the OpenPaths catalog. No tier discounts, caching discounts, or promotional rates are estimated.',
        },
      },
      {
        '@type': 'Question',
        name: 'What does BYOK mean for my monthly cost?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'With bring-your-own-key (BYOK), requests run on your own provider key and bypass the OpenPaths balance entirely — the cost recorded on OpenPaths is $0. You pay the provider directly under its own pricing.',
        },
      },
      {
        '@type': 'Question',
        name: 'Why include openpaths/auto in the comparison?',
        acceptedAnswer: {
          '@type': 'Answer',
          text: 'openpaths/auto routes each prompt across frontier and efficient models automatically, so its blended price often lands below always-frontier usage without you picking models per request.',
        },
      },
    ],
  };

  return (
    <>
      <Seo
        title="LLM Cost Calculator | Estimate Monthly Token Spend | OpenPaths"
        description="Estimate monthly LLM spend from your input and output token volume. Compare published per-token prices side by side, see deltas between frontier and efficient models, and learn how BYOK changes the math."
        path="/calculator"
        jsonLd={faqJsonLd}
      />
      <div className="mx-auto max-w-6xl px-6 py-16">
        <section className="mb-12">
          <div className="mb-4 inline-flex items-center gap-2 font-mono text-xs uppercase tracking-[0.22em] text-cyan-300">
            <CalculatorIcon className="h-4 w-4" />
            Cost calculator
          </div>
          <h1 className="text-4xl font-semibold tracking-tight md:text-5xl">Price your monthly token volume.</h1>
          <p className="mt-5 max-w-3xl text-lg leading-relaxed text-white/58">
            Enter what your workload sends and receives each month. The table applies each model's published
            per-1M-token price from our catalog — nothing estimated, no invented competitor pricing. Compare up to four
            models side by side.
          </p>
        </section>

        <section className="mb-10 grid gap-6 md:grid-cols-2">
          <TokenField
            label="Monthly input tokens"
            value={inputTokens}
            onChange={setInputTokens}
            hint="Prompt and system tokens sent to the model"
          />
          <TokenField
            label="Monthly output tokens"
            value={outputTokens}
            onChange={setOutputTokens}
            hint="Completion tokens returned by the model"
          />
        </section>

        <section className="mb-10">
          <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
            <h2 className="font-mono text-xs uppercase tracking-[0.22em] text-white/45">
              Models ({selectedIds.length}/{MAX_SELECTION} selected)
            </h2>
            <span className="font-mono text-xs text-white/40">
              {calculatorModels.length} token-priced chat models in catalog
            </span>
          </div>
          <div className="flex flex-wrap gap-2">
            {calculatorModels.map(model => {
              const active = selectedIds.includes(model.id);
              const disabled = !active && selectedIds.length >= MAX_SELECTION;
              return (
                <button
                  key={model.id}
                  type="button"
                  onClick={() => toggleModel(model.id)}
                  disabled={disabled}
                  className={`rounded-full border px-3 py-1.5 font-mono text-xs transition ${
                    active
                      ? 'border-cyan-300/60 bg-cyan-300/10 text-cyan-200'
                      : disabled
                        ? 'cursor-not-allowed border-white/10 bg-white/[0.03] text-white/25'
                        : 'border-white/20 bg-white/[0.05] text-white/60 hover:border-white/40 hover:text-white'
                  }`}
                >
                  {model.name}
                  <span className="ml-2 text-white/35">
                    ${model.priceInput}/${model.priceOutput} /1M
                  </span>
                </button>
              );
            })}
          </div>
        </section>

        <section className="overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
          <div className="flex flex-wrap items-center justify-between gap-4 border-b border-white/20 px-4 py-3">
            <div className="flex items-center gap-2 font-mono text-xs uppercase tracking-[0.22em] text-white/45">
              <Table2 className="h-4 w-4" />
              Monthly cost estimate
            </div>
            <span className="font-mono text-xs text-white/40">
              {monthlyInput.toLocaleString()} in / {monthlyOutput.toLocaleString()} out per month
            </span>
          </div>
          {rows.length === 0 ? (
            <p className="px-4 py-8 text-sm text-white/50">Select at least one model above.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full min-w-[760px] text-left">
                <thead>
                  <tr className="border-b border-white/15 font-mono text-xs uppercase tracking-wider text-white/45">
                    <th className="px-4 py-3 font-normal">Model</th>
                    <th className="px-4 py-3 text-right font-normal">Input $</th>
                    <th className="px-4 py-3 text-right font-normal">Output $</th>
                    <th className="px-4 py-3 text-right font-normal">Total / mo</th>
                    <th className="px-4 py-3 text-right font-normal">vs priciest</th>
                    <th className="px-4 py-3 font-normal">With BYOK</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map(row => {
                    const total = row.inputCost + row.outputCost;
                    const delta = total - maxTotal;
                    const isAuto = row.model.id === 'openpaths/auto';
                    return (
                      <tr key={row.model.id} className="border-b border-white/10 last:border-b-0 align-top">
                        <td className="px-4 py-4">
                          <Link to={`/models/${row.model.id}`} className="text-sm text-white hover:text-cyan-200">
                            {row.model.name}
                          </Link>
                          <div className="mt-1 font-mono text-xs text-white/40">
                            ${row.model.priceInput} / 1M in · ${row.model.priceOutput} / 1M out
                          </div>
                        </td>
                        <td className="px-4 py-4 text-right font-mono text-sm text-white/70">{formatUsd(row.inputCost)}</td>
                        <td className="px-4 py-4 text-right font-mono text-sm text-white/70">{formatUsd(row.outputCost)}</td>
                        <td className="px-4 py-4 text-right font-mono text-sm font-semibold text-white">{formatUsd(total)}</td>
                        <td className="px-4 py-4 text-right font-mono text-sm">
                          {delta === 0 ? (
                            <span className="text-white/40">baseline</span>
                          ) : (
                            <span className="text-emerald-300">
                              saves {formatUsd(Math.abs(delta))}/mo
                            </span>
                          )}
                        </td>
                        <td className="px-4 py-4 text-sm leading-relaxed text-white/55">
                          {isAuto ? (
                            <>Not applicable — auto-routing runs on the OpenPaths balance.</>
                          ) : (
                            <>
                              $0 recorded on OpenPaths with{' '}
                              <Link to="/byok" className="text-cyan-300 hover:text-cyan-200">
                                your own provider key
                              </Link>
                              ; you pay the provider directly.
                            </>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </section>

        <section className="mt-8 rounded-lg border border-white/10 bg-white/[0.03] px-5 py-4">
          <h2 className="font-mono text-xs uppercase tracking-[0.22em] text-white/45">Methodology</h2>
          <p className="mt-2 max-w-4xl text-sm leading-relaxed text-white/55">
            Estimates use published list prices for token-priced text models from the OpenPaths catalog:
            monthly&nbsp;$ = tokens ÷ 1M × price per 1M tokens, computed separately for input and output. No tier or
            committed-use discounts, prompt-caching discounts, or promotional rates are estimated. Image, video,
            audio, and per-request models are excluded because their units differ. Prices shown are the catalog's
            current rates; check{' '}
            <Link to="/pricing" className="text-cyan-300 hover:text-cyan-200">
              /pricing
            </Link>{' '}
            before committing a budget.
          </p>
        </section>

        <section className="mt-14 grid gap-4 md:grid-cols-3">
          <CtaCard
            to="/pricing"
            title="Full price list"
            copy="Every model's published rate, filterable by provider and modality."
          />
          <CtaCard
            to="/byok"
            title="Bring your own key"
            copy="Run requests on your provider keys with zero balance recording on OpenPaths."
          />
          <CtaCard
            to="/models"
            title="Browse models"
            copy="Context windows, tags, and specs for the full catalog."
          />
        </section>
      </div>
    </>
  );
}

function TokenField({
  label,
  value,
  onChange,
  hint,
}: {
  label: string;
  value: string;
  onChange: (next: string) => void;
  hint: string;
}) {
  const numeric = Number.parseFloat(value.replace(/[^0-9.]/g, '')) || 0;
  return (
    <div className="rounded-lg border border-white/15 bg-white/[0.04] p-5">
      <label className="block font-mono text-xs uppercase tracking-[0.18em] text-white/45">{label}</label>
      <input
        type="number"
        min={0}
        value={numeric}
        onChange={event => onChange(event.target.value)}
        className="mt-3 w-full rounded-md border border-white/20 bg-black/30 px-3 py-2 font-mono text-lg text-white outline-none focus:border-cyan-300/60"
      />
      <p className="mt-2 text-xs text-white/40">{hint}</p>
      <div className="mt-3 flex flex-wrap gap-2">
        {TOKEN_PRESETS.map(preset => (
          <button
            key={preset.label}
            type="button"
            onClick={() => onChange(String(preset.value))}
            className={`rounded border px-2.5 py-1 font-mono text-xs transition ${
              numeric === preset.value
                ? 'border-cyan-300/60 bg-cyan-300/10 text-cyan-200'
                : 'border-white/20 bg-white/[0.05] text-white/55 hover:border-white/40 hover:text-white'
            }`}
          >
            {preset.label}
          </button>
        ))}
      </div>
    </div>
  );
}

function CtaCard({ to, title, copy }: { to: string; title: string; copy: string }) {
  return (
    <Link
      to={to}
      className="group rounded-lg border border-white/15 bg-white/[0.04] p-5 transition hover:border-cyan-300/50 hover:bg-white/[0.07]"
    >
      <div className="flex items-center gap-2">
        <KeyRound className="h-4 w-4 text-cyan-300" />
        <span className="text-base font-medium text-white">{title}</span>
      </div>
      <p className="mt-2 text-sm leading-relaxed text-white/55">{copy}</p>
      <ArrowRight className="mt-3 h-4 w-4 text-white/40 transition group-hover:translate-x-1 group-hover:text-cyan-300" />
    </Link>
  );
}
