import type { BlogPost } from './blog';

export const openpathsBrowse: BlogPost = {
  slug: 'openpaths-browse-agent-in-your-browser',
  title: 'OpenPaths Browse: a browser agent that works on a free model',
  date: '2026-10-16',
  excerpt:
    'We rebuilt Sitegeist as OpenPaths Browse and tested it on 14 small pages that hide a code behind a click, a form, a popup or a canvas. Muse Spark 1.3 passes all 14; openpaths/free passes the 12 text tasks it ran, with one upstream error.',
  author: 'OpenPaths',
  readTime: '6 min',
  tags: ['agents', 'browser', 'evals', 'open-source'],
  content: `OpenPaths Browse is a Chromium side panel that lets an LLM read and use the page you are on, in its own tab group, in the background. It is the Sitegeist extension, renamed and tightened, and it ships with OpenPaths as a first-class provider: the free \`openpaths/free\` route works with one key, and every other OpenPaths model is the same key.

Source: [github.com/lee101/sitegeist](https://github.com/lee101/sitegeist) (the repository keeps its old name for now; the CLI is \`openpaths-browse\`, the old \`sitegeist\` command still works).

## It did nothing on pages. Why.

We started from a bug report that sounds vague: with Muse Spark selected, the agent "doesn't seem to do anything on a page". Reproducing it end to end (the real side panel, a real model, a local test site) found five separate causes, each of which would have been enough on its own:

| Symptom | Cause | Fix |
| --- | --- | --- |
| The model answered "I can't open that page" | The system prompt and all tool declarations never reached the model: the message transformer dropped the leading \`system\` message that the agent library now carries them in | Let \`system\` through |
| Every click and script failed with "Debugger is not attached" | The extension skipped \`chrome.debugger.attach\` whenever *any* client (DevTools, a test driver) was attached | Always attach, re-attach once on a stale attachment |
| The agent froze for two minutes after one typo | A syntax error in a \`repl\` snippet stops the inlined module script from parsing, so the sandbox never reports back | Report the parse error to the model immediately |
| "This session has no tab yet" right after opening a page | The first save renamed the browser session by a freshly generated id and orphaned the tab | Keep one stable id per panel |
| Tool calls with empty names on the OpenPaths free route | The gateway streams tool calls with no \`index\` and sends argument fragments with an empty id and name; each fragment became its own call | Attach index-less continuations to the last tool call |

Two further findings were build problems: the bundle contained two copies of the model library (one stale), and the stale one was the copy that ran.

## Tools that cost fewer tokens

The side panel used to have one real browsing tool: a JavaScript REPL. Small models write broken JavaScript. It now has four plain tools, built on the same handlers the MCP bridge uses:

- \`read_page\`: a snapshot with element refs and readable text, **footer left out** (it reports how much it dropped).
- \`find_text\`: grep for the page's visible text, with context. \`jump\` scrolls a match to the middle of the viewport and highlights it; \`screenshot\` returns that view.
- \`act\`: click, type, fill, press, scroll, hover, wait, by ref, selector or pixel.
- \`screenshot\`: JPEG of the tab, background tabs included.

Page content is also cleaned before the model sees it. Data URIs and long base64 runs become a short marker, HTML drops scripts, styles and inline SVG, and text is capped with a hint about how to get the rest. One of our eval pages carries a 300 KB base64 image; the requests stay at about 43 KB, system prompt and tool definitions included.

## The evals

\`scripts/evals/\` serves 14 small pages and drives the real side panel agent headless in a throwaway profile, so it never takes your window focus. Each page hides a code that is only reachable by using it:

| Task | What it needs |
| --- | --- |
| click, form, dialog | real input, form fields, a modal |
| scroll, findtext | a code at the bottom, or deep in a long page ("do not read the whole page") |
| dynamic | wait for content that appears after 2.5 s |
| nav, hidden | follow links, open a menu |
| popup, tabs | links that open new tabs, read two tabs |
| canvas | click a shape drawn on a canvas |
| datauri | answer from a page with a 300 KB inline image; fails if any request grows past 150 KB |
| footer | the footer holds a decoy; fails if it reaches the model |
| jump | the answer is drawn on a canvas below a text match; only readable from the jump screenshot |

Results from the final run:

| Model | Result |
| --- | --- |
| \`meta/muse-spark-1.3-contributor\` | 14 / 14 |
| \`openpaths/openpaths/free\` | 12 / 13 (text-only tasks; the one miss was the gateway returning a Cloudflare error page mid-run) |

Before the fixes, Muse passed 0 of 10 and the free route 0 of 10. Typical tasks take 9 to 30 seconds and 3 to 14 model calls.

## How it compares

We read the code of the other open-source browser agents to see what to copy and what not to.

| Project | Good idea | Take it? |
| --- | --- | --- |
| browser-use (MIT) | zero-LLM-cost \`search_page\` and \`find_elements\` | yes, that is \`find_text\` |
| Nanobrowser (Apache-2.0) | staged context compaction: elide old tool output before summarizing | worth porting next |
| WebBrain (GPL) | batched browser actions, accessibility-tree clicks, WebMCP | batching already exists (\`actions\`); GPL means ideas only |
| OpenSidekick (MIT) | \`read_console\` / \`read_network\` tools | already in the bridge |
| OpenBUA | per-site workflows (Docs, Reddit, X) | site skills, not a core tool |

None of them is a better base than this one for what we want: a side panel *and* a bridge that lets Claude Code, Codex or omp use the same browser in a background tab group, with one free model route that works.

## Try it

\`\`\`bash
git clone https://github.com/lee101/sitegeist && cd sitegeist
npm ci && node scripts/build.mjs        # unpacked extension in dist-chrome/
# chrome://extensions -> Developer mode -> Load unpacked -> dist-chrome
# side panel -> pick OpenPaths -> openpaths/free

node scripts/evals/run-sidepanel.mjs --provider openpaths --model openpaths/free
\`\`\`

Driving it from a coding agent: \`openpaths-browse mcp\` is an MCP server (\`claude mcp add openpaths-browse -- openpaths-browse mcp\`). Each harness gets its own tab group, works in the background and cannot see your other tabs.

## Limits

- The eval pages are small and local. They test that the plumbing and the tools work, not that a model survives the real web.
- One run per task; model variance is real, which is why the runner keeps the raw model streams of every task for diagnosis.
- Vision was only tested with Muse. The free route is text-only.`,
};
