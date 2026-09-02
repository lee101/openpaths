import React from 'react';
import { Link } from 'react-router-dom';
import { ExternalLink, ImageIcon, Sparkles } from 'lucide-react';
import { Seo } from '../components/Seo';
import {
  OPENPATHS_IMAGE_MODELS,
  artificialAnalysisImageSnapshot,
  displayElo,
  displayWinRate,
  formatPricePer1k,
  imageEditingLeaderboard,
  matchOpenPathsModel,
  topImageModels,
  type ImageLeaderboardModel,
} from '../lib/artificialAnalysisImages';

const BLOG_SLUG = 'image-generator-quality-compared-2026';

function crawledOn(): string {
  try {
    return new Date(artificialAnalysisImageSnapshot.crawledAt).toLocaleDateString('en-US', {
      year: 'numeric', month: 'short', day: 'numeric',
    });
  } catch {
    return artificialAnalysisImageSnapshot.crawledAt;
  }
}

function LeaderboardTable({ rows, limit }: { rows: ImageLeaderboardModel[]; limit: number }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[820px] text-left text-sm">
        <thead className="font-mono text-xs uppercase tracking-[0.14em] text-white/50">
          <tr>
            <th className="px-4 py-3">#</th>
            <th className="px-4 py-3">Model</th>
            <th className="px-4 py-3">Creator</th>
            <th className="px-4 py-3">Arena Elo</th>
            <th className="px-4 py-3">Win rate</th>
            <th className="px-4 py-3">$/1k imgs</th>
            <th className="px-4 py-3">On OpenPaths</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-white/10">
          {rows.slice(0, limit).map(model => {
            const hosted = matchOpenPathsModel(model.name);
            return (
              <tr key={model.id} className="text-white/70">
                <td className="px-4 py-4 font-mono text-white/55">{model.rank}</td>
                <td className="px-4 py-4">
                  <div className="font-mono text-white">{model.name}</div>
                  {model.openWeights && <div className="text-xs text-emerald-300/70">open weights</div>}
                </td>
                <td className="px-4 py-4">{model.creator.name}</td>
                <td className="px-4 py-4 font-mono text-cyan-200">{displayElo(model.elo)}</td>
                <td className="px-4 py-4 font-mono">{displayWinRate(model.winRate)}</td>
                <td className="px-4 py-4 font-mono">{formatPricePer1k(model.pricePer1kImages)}</td>
                <td className="px-4 py-4">
                  {hosted ? (
                    <Link
                      to="/text-to-image"
                      className="inline-flex items-center gap-1 rounded-full border border-cyan-400/40 bg-cyan-400/10 px-2.5 py-1 font-mono text-xs text-cyan-200 hover:border-cyan-300/70"
                    >
                      {hosted.openpathsId}
                    </Link>
                  ) : (
                    <span className="font-mono text-xs text-white/40">-</span>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

export function ImageEvals() {
  const leaders = topImageModels(20);
  const editing = imageEditingLeaderboard;

  return (
    <>
      <Seo
        title="Image Generator Evals - Arena Elo, Price & Example Images | OpenPaths"
        description="Artificial Analysis Text-to-Image and Image Editing Arena Elo leaderboards, plus a real example image from every generator OpenPaths hosts - RA1, GPT Image 2, FLUX, and more."
        path="/image-evals"
      />
      <div className="mx-auto max-w-7xl px-6 py-12">
        <section className="mb-10">
          <div className="mb-4 inline-flex items-center gap-2 font-mono text-xs uppercase tracking-[0.22em] text-cyan-300">
            <ImageIcon className="h-4 w-4" />
            Image generation evals
          </div>
          <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <h1 className="max-w-4xl text-4xl font-semibold tracking-tight md:text-6xl">Image generator quality map</h1>
              <p className="mt-5 max-w-3xl text-lg leading-relaxed text-white/60">
                The Artificial Analysis Text-to-Image and Image Editing Arena Elo leaderboards, snapshotted locally and
                normalized for OpenPaths - plus a real example image from every generator we host, all from the same prompt.
              </p>
            </div>
            <a
              href={artificialAnalysisImageSnapshot.sourceUrl}
              target="_blank"
              rel="noreferrer noopener"
              className="inline-flex items-center justify-center gap-2 rounded-lg border border-white/20 px-4 py-3 font-mono text-sm text-white/65 transition-colors hover:border-white/50 hover:text-white"
            >
              Source <ExternalLink className="h-4 w-4" />
            </a>
          </div>
          <p className="mt-3 font-mono text-xs text-white/50">Crawled {crawledOn()} · {leaders.length} of {artificialAnalysisImageSnapshot.textToImage.length} ranked models shown</p>
        </section>

        {/* Example gallery - one real generation per OpenPaths generator */}
        <section className="mb-12">
          <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
            <h2 className="inline-flex items-center gap-2 font-mono text-sm uppercase tracking-[0.16em] text-white/65">
              <Sparkles className="h-4 w-4 text-cyan-300" /> Same prompt, every generator
            </h2>
            <Link to={`/blog/${BLOG_SLUG}`} className="font-mono text-xs text-cyan-200 hover:text-white">
              Read the full quality comparison →
            </Link>
          </div>
          <p className="mb-6 max-w-3xl text-sm text-white/50">
            Prompt: <span className="italic text-white/70">“A cinematic photograph of a red fox wearing a tiny astronaut helmet,
            sitting on a mossy rock in a foggy pine forest at golden hour, shallow depth of field, ultra detailed.”</span>
          </p>
          <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
            {OPENPATHS_IMAGE_MODELS.map(model => (
              <figure key={model.slug} className="overflow-hidden rounded-2xl border border-white/20 bg-white/[0.05]">
                <div className="aspect-square w-full bg-black/40">
                  <img src={model.image} alt={`${model.label} example generation`} loading="lazy" className="h-full w-full object-cover" />
                </div>
                <figcaption className="p-4">
                  <div className="flex items-center justify-between gap-2">
                    <span className="font-mono text-sm text-white">{model.label}</span>
                    {!model.onLeaderboard && (
                      <span className="rounded-full border border-fuchsia-400/40 bg-fuchsia-400/10 px-2 py-0.5 font-mono text-[10px] text-fuchsia-200">
                        OpenPaths exclusive
                      </span>
                    )}
                  </div>
                  <div className="mt-0.5 text-xs text-white/55">{model.provider}</div>
                  <p className="mt-2 text-xs leading-relaxed text-white/55">{model.blurb}</p>
                  <Link to="/text-to-image" className="mt-3 inline-flex font-mono text-xs text-cyan-200 hover:text-white">
                    Try {model.openpathsId} →
                  </Link>
                </figcaption>
              </figure>
            ))}
          </div>
        </section>

        {/* Text-to-Image leaderboard */}
        <section className="mb-12 overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
          <div className="flex flex-wrap items-center justify-between gap-4 border-b border-white/20 px-4 py-3">
            <h2 className="font-mono text-sm uppercase tracking-[0.16em] text-white/65">Text-to-Image Arena</h2>
            <span className="font-mono text-xs text-white/50">{artificialAnalysisImageSnapshot.textToImage.length} models · blind preference Elo</span>
          </div>
          <LeaderboardTable rows={leaders} limit={20} />
        </section>

        {/* Image editing leaderboard */}
        {editing.length > 0 && (
          <section className="overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
            <div className="flex flex-wrap items-center justify-between gap-4 border-b border-white/20 px-4 py-3">
              <h2 className="font-mono text-sm uppercase tracking-[0.16em] text-white/65">Image Editing Arena</h2>
              <span className="font-mono text-xs text-white/50">{editing.length} models</span>
            </div>
            <LeaderboardTable rows={editing} limit={12} />
          </section>
        )}
      </div>
    </>
  );
}
