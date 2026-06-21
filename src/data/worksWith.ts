// Apps in the broader OpenRouter-compatible ecosystem that work with OpenPaths.
// OpenPaths is OpenAI- and Anthropic-compatible, so anything that speaks OpenRouter
// speaks OpenPaths by pointing at https://openpaths.io/v1 with an OpenPaths key.
//
// status:
//   native-merged  - first-class OpenPaths provider merged upstream
//   native-pr      - PR open to add a first-class OpenPaths provider
//   compatible     - works today via a custom OpenAI-compatible base URL + key
//   listed         - proprietary / no public repo; works with a BYO OpenPaths key

export type WorksWithStatus = 'native-merged' | 'native-pr' | 'compatible' | 'listed';

export type WorksWithApp = {
  slug: string;
  name: string;
  url: string;
  repo?: string;
  description: string;
  category: string;
  oss: boolean;
  status: WorksWithStatus;
  prUrl?: string;
  setup?: string;
};

export const worksWithFavicon = (url: string) =>
  `https://t0.gstatic.com/faviconV2?client=SOCIAL&type=FAVICON&fallback_opts=TYPE,SIZE,URL&url=${encodeURIComponent(url)}&size=128`;

export const WORKS_WITH_CATEGORIES = [
  'Coding agents',
  'Chat & desktop',
  'Agents & assistants',
  'Frameworks & SDKs',
  'Creative & writing',
  'Observability & infra',
  'Productivity',
] as const;

const OPENAI_COMPAT = 'OpenAI-compatible provider → base URL https://openpaths.io/v1, OpenPaths key.';

// Repos + statuses reconciled from research; prUrl/status flip to native-pr as PRs land.
export const worksWithApps: WorksWithApp[] = [
  // ---- Coding agents ----
  {
    slug: 'aider', name: 'Aider', url: 'https://aider.chat/', repo: 'https://github.com/Aider-AI/aider',
    description: 'AI pair programming in your terminal, working with LLMs on your existing codebase.',
    category: 'Coding agents', oss: true, status: 'native-pr',
    prUrl: 'https://github.com/Aider-AI/aider/pull/5270',
    setup: 'OPENAI_API_BASE=https://openpaths.io/v1, OPENAI_API_KEY=<key>, --model openai/openpaths/auto-code.',
  },
  {
    slug: 'cline', name: 'Cline', url: 'https://cline.bot/', repo: 'https://github.com/cline/cline',
    description: 'Autonomous coding agent in your IDE.',
    category: 'Coding agents', oss: true, status: 'compatible',
    setup: 'Pick "OpenAI Compatible", ' + OPENAI_COMPAT,
  },
  {
    slug: 'roo-code', name: 'Roo Code', url: 'https://roocode.com/', repo: 'https://github.com/RooCodeInc/Roo-Code',
    description: 'AI-powered autonomous coding agent that lives in your editor.',
    category: 'Coding agents', oss: true, status: 'compatible',
    setup: 'Pick "OpenAI Compatible", ' + OPENAI_COMPAT,
  },
  {
    slug: 'kilo-code', name: 'Kilo Code', url: 'https://kilocode.ai/', repo: 'https://github.com/Kilo-Org/kilocode',
    description: 'AI coding assistant for VS Code supporting many providers and models.',
    category: 'Coding agents', oss: true, status: 'native-pr',
    prUrl: 'https://github.com/Kilo-Org/kilocode/pull/11283',
    setup: OPENAI_COMPAT,
  },
  {
    slug: 'vt-code', name: 'VT Code', url: 'https://github.com/vinhnx/vtcode', repo: 'https://github.com/vinhnx/vtcode',
    description: 'Semantic coding agent in the terminal.',
    category: 'Coding agents', oss: true, status: 'compatible',
    setup: 'Add a [[custom_providers]] block in vtcode.toml: base_url https://openpaths.io/v1, OPENPATHS_API_KEY.',
  },
  {
    slug: 'autohand-code', name: 'Autohand Code CLI', url: 'https://github.com/autohandai/code-cli', repo: 'https://github.com/autohandai/code-cli',
    description: 'Fast open-source coding CLI agent.',
    category: 'Coding agents', oss: true, status: 'native-pr',
    prUrl: 'https://github.com/autohandai/code-cli/pull/244',
  },
  {
    slug: 'soulforge', name: 'SoulForge', url: 'https://github.com/proxysoul/soulforge', repo: 'https://github.com/proxysoul/soulforge',
    description: 'Graph-powered terminal coding agent that edits code by symbol via AST.',
    category: 'Coding agents', oss: true, status: 'compatible',
    setup: 'Register a custom OpenAI-compatible provider (base URL https://openpaths.io/v1).',
  },
  {
    slug: 'nanocode', name: 'nanocode', url: 'https://github.com/1rgs/nanocode', repo: 'https://github.com/1rgs/nanocode',
    description: 'Minimal Claude Code alternative — single Python file, full agentic loop.',
    category: 'Coding agents', oss: true, status: 'compatible',
    setup: 'Uses the Anthropic Messages API — point it at https://openpaths.io/v1/messages with OPENPATHS_API_KEY.',
  },
  {
    slug: 'github-copilot', name: 'GitHub Copilot', url: 'https://github.com/features/copilot', repo: undefined,
    description: 'Agent mode in VS Code; bring your own OpenRouter-style key.',
    category: 'Coding agents', oss: false, status: 'listed',
  },
  {
    slug: 'gitbug', name: 'GitBug', url: 'https://gitbug.dev/', repo: undefined,
    description: 'AI-powered code review for GitHub PRs with your own key.',
    category: 'Coding agents', oss: false, status: 'listed',
  },

  // ---- Chat & desktop ----
  {
    slug: 'sillytavern', name: 'SillyTavern', url: 'https://sillytavern.app/', repo: 'https://github.com/SillyTavern/SillyTavern',
    description: 'Local-first LLM frontend for power users: chat, character cards, many providers.',
    category: 'Chat & desktop', oss: true, status: 'compatible',
    setup: 'Chat Completion → Custom (OpenAI-compatible), endpoint https://openpaths.io/v1, OpenPaths key.',
  },
  {
    slug: 'librechat', name: 'LibreChat', url: 'https://librechat.ai/', repo: 'https://github.com/danny-avila/LibreChat',
    description: 'Open-source chat app compatible with any AI provider.',
    category: 'Chat & desktop', oss: true, status: 'native-pr',
    prUrl: 'https://github.com/danny-avila/LibreChat/pull/13781',
    setup: 'Add a custom endpoint in librechat.yaml: baseURL https://openpaths.io/v1, apiKey ${OPENPATHS_API_KEY}.',
  },
  {
    slug: 'chatbox', name: 'Chatbox', url: 'https://chatboxai.app/', repo: 'https://github.com/chatboxai/chatbox',
    description: 'Desktop client for ChatGPT, Claude and other LLMs across all platforms.',
    category: 'Chat & desktop', oss: true, status: 'native-pr',
    prUrl: 'https://github.com/chatboxai/chatbox/pull/3761',
    setup: 'Add provider → OpenAI compatible, API host https://openpaths.io/v1, OpenPaths key.',
  },
  {
    slug: 'chorus', name: 'Chorus', url: 'https://chorus.sh/', repo: 'https://github.com/meltylabs/chorus',
    description: 'macOS app for unified access to many AI models.',
    category: 'Chat & desktop', oss: true, status: 'native-pr',
    prUrl: 'https://github.com/meltylabs/chorus/pull/72',
  },
  {
    slug: 'warden', name: 'Warden', url: 'https://github.com/SidhuK/WardenApp', repo: 'https://github.com/SidhuK/WardenApp',
    description: 'Native Swift macOS app supporting many providers via your own keys.',
    category: 'Chat & desktop', oss: true, status: 'native-pr',
    prUrl: 'https://github.com/SidhuK/WardenApp/pull/56',
  },
  {
    slug: 'skales', name: 'Skales', url: 'https://skales.app/', repo: 'https://github.com/skalesapp/skales',
    description: 'Local AI desktop agent for Windows, macOS & Linux with 15+ providers.',
    category: 'Chat & desktop', oss: true, status: 'compatible',
    setup: 'Use the built-in Custom (OpenAI-compatible) provider with base URL https://openpaths.io.',
  },
  {
    slug: 'chatlima', name: 'ChatLima', url: 'https://chatlima.com/', repo: undefined,
    description: 'MCP-powered AI chatbot with multi-model support and key management.',
    category: 'Chat & desktop', oss: false, status: 'listed',
  },
  {
    slug: 'boltai', name: 'BoltAI', url: 'https://boltai.com/', repo: undefined,
    description: 'Native Mac AI app with access to 300+ models.',
    category: 'Chat & desktop', oss: false, status: 'listed',
  },

  // ---- Agents & assistants ----
  {
    slug: 'agent-zero', name: 'Agent Zero', url: 'https://github.com/agent0ai/agent-zero', repo: 'https://github.com/agent0ai/agent-zero',
    description: 'Build autonomous AI agents effortlessly.',
    category: 'Agents & assistants', oss: true, status: 'native-pr',
    prUrl: 'https://github.com/agent0ai/agent-zero/pull/1707',
  },
  {
    slug: 'browser-use', name: 'Browser Use', url: 'https://browser-use.com/', repo: 'https://github.com/browser-use/browser-use',
    description: 'Open-source browser agent driven via CDP as the action layer for any LLM.',
    category: 'Agents & assistants', oss: true, status: 'compatible',
    setup: 'ChatOpenAI(base_url="https://openpaths.io/v1", api_key=OPENPATHS_API_KEY, model="openpaths/auto").',
  },
  {
    slug: 'openclaw', name: 'OpenClaw (Moltbot)', url: 'https://openclaw.ai/', repo: 'https://github.com/openclaw/openclaw',
    description: 'Personal AI assistant connecting to WhatsApp, Telegram, Discord, Slack and more.',
    category: 'Agents & assistants', oss: true, status: 'compatible',
  },
  {
    slug: 'nanoclaw', name: 'NanoClaw', url: 'https://github.com/nanocoai/nanoclaw', repo: 'https://github.com/nanocoai/nanoclaw',
    description: 'Lightweight agent runner in isolated containers across messaging channels.',
    category: 'Agents & assistants', oss: true, status: 'compatible',
    setup: 'Runs on the Claude Agent SDK — set ANTHROPIC_BASE_URL=https://openpaths.io, token=OPENPATHS_API_KEY.',
  },
  {
    slug: 'agent-swarm', name: 'Agent Swarm', url: 'https://github.com/desplega-ai/agent-swarm', repo: 'https://github.com/desplega-ai/agent-swarm',
    description: 'Coordination intelligence for AI coding agents; bring your own key.',
    category: 'Agents & assistants', oss: true, status: 'compatible',
  },
  {
    slug: 'dexto', name: 'Dexto', url: 'https://github.com/truffle-ai/dexto', repo: 'https://github.com/truffle-ai/dexto',
    description: 'Open agent harness with a coding agent, CLI, and Web UI.',
    category: 'Agents & assistants', oss: true, status: 'compatible',
    setup: 'Use the openai-compatible provider with baseURL https://openpaths.io/v1, OPENPATHS_API_KEY.',
  },
  {
    slug: 'project-airi', name: 'Project AIRI', url: 'https://airi.moeru.ai/', repo: 'https://github.com/moeru-ai/airi',
    description: 'Open-source virtual companion that can chat, listen, speak, and play games.',
    category: 'Agents & assistants', oss: true, status: 'native-pr',
    prUrl: 'https://github.com/moeru-ai/airi/pull/1980',
  },
  {
    slug: 'space-agent', name: 'Space Agent', url: 'https://github.com/agent0ai/space-agent', repo: 'https://github.com/agent0ai/space-agent',
    description: 'Free, open-source AI agent that builds your space in the browser.',
    category: 'Agents & assistants', oss: true, status: 'compatible',
    setup: 'Set apiEndpoint https://openpaths.io/v1/chat/completions, your key, model openpaths/auto.',
  },

  // ---- Frameworks & SDKs ----
  {
    slug: 'llamaindex', name: 'LlamaIndex', url: 'https://www.llamaindex.ai/', repo: 'https://github.com/run-llama/llama_index',
    description: 'Framework for building knowledge assistants over your data.',
    category: 'Frameworks & SDKs', oss: true, status: 'native-pr',
    prUrl: 'https://github.com/run-llama/llama_index/pull/21993',
    setup: 'OpenAILike(api_base="https://openpaths.io/v1", api_key=OPENPATHS_API_KEY, model="openpaths/auto").',
  },
  {
    slug: 'mastra', name: 'Mastra', url: 'https://mastra.ai/', repo: 'https://github.com/mastra-ai/mastra',
    description: 'TypeScript framework for building AI apps and agents.',
    category: 'Frameworks & SDKs', oss: true, status: 'compatible',
    setup: 'Use a custom OpenAI-compatible model with baseURL https://openpaths.io/v1.',
  },
  {
    slug: 'stirrup', name: 'Stirrup', url: 'https://github.com/ArtificialAnalysis/Stirrup', repo: 'https://github.com/ArtificialAnalysis/Stirrup',
    description: 'Lightweight framework for building agents with any OpenRouter-style model.',
    category: 'Frameworks & SDKs', oss: true, status: 'native-pr',
    prUrl: 'https://github.com/ArtificialAnalysis/Stirrup/pull/56',
    setup: 'ChatCompletionsClient(base_url="https://openpaths.io/v1", api_key=OPENPATHS_API_KEY).',
  },
  {
    slug: 'openrouter-rs', name: 'openrouter-rs', url: 'https://github.com/realmorrisliu/openrouter-rs', repo: 'https://github.com/realmorrisliu/openrouter-rs',
    description: 'Community Rust SDK + CLI for OpenRouter-compatible APIs.',
    category: 'Frameworks & SDKs', oss: true, status: 'compatible',
    setup: 'builder().base_url("https://openpaths.io/v1").api_key(OPENPATHS_API_KEY).',
  },
  {
    slug: 'or-mcp-multimodal', name: 'OpenRouter MCP Multimodal', url: 'https://github.com/stabgan/openrouter-mcp-multimodal', repo: 'https://github.com/stabgan/openrouter-mcp-multimodal',
    description: 'MCP server exposing many models for chat and multimodal analysis/generation.',
    category: 'Frameworks & SDKs', oss: true, status: 'compatible',
  },
  {
    slug: 'quests', name: 'Quests', url: 'https://quests.dev/', repo: 'https://github.com/quests-org/quests',
    description: 'Open-source app builder.',
    category: 'Frameworks & SDKs', oss: true, status: 'native-pr',
    prUrl: 'https://github.com/quests-org/quests/pull/46',
  },
  {
    slug: 'shakespeare', name: 'Shakespeare', url: 'https://gitlab.com/soapbox-pub/shakespeare', repo: 'https://gitlab.com/soapbox-pub/shakespeare',
    description: 'Browser-based AI app builder that runs entirely in-browser.',
    category: 'Frameworks & SDKs', oss: true, status: 'compatible',
    setup: 'Add an AI provider preset with base URL https://openpaths.io/v1.',
  },

  // ---- Creative & writing ----
  {
    slug: 'aventura', name: 'Aventura', url: 'https://github.com/unkarelian/Aventura', repo: 'https://github.com/unkarelian/Aventura',
    description: 'Free, open-source AI adventure and creative writing app.',
    category: 'Creative & writing', oss: true, status: 'compatible',
    setup: 'Add an API profile in Settings: base URL https://openpaths.io/v1, OpenPaths key.',
  },
  {
    slug: 'novelcrafter', name: 'Novelcrafter', url: 'https://novelcrafter.com/', repo: undefined,
    description: 'All-in-one writing workspace with model-agnostic AI assistance.',
    category: 'Creative & writing', oss: false, status: 'listed',
  },
  {
    slug: 'nikke-db-rp', name: 'Nikke-DB RP Generator', url: 'https://nikke-db.pages.dev/', repo: 'https://github.com/Nikke-db/nikke-db-vue',
    description: 'Interactive roleplay and story generation with synced Live2D animation.',
    category: 'Creative & writing', oss: true, status: 'compatible',
  },

  // ---- Observability & infra ----
  {
    slug: 'helicone', name: 'Helicone', url: 'https://helicone.ai/', repo: 'https://github.com/Helicone/ai-gateway',
    description: 'Open-source LLM observability, evals, and AI gateway; proxy your OpenPaths traffic.',
    category: 'Observability & infra', oss: true, status: 'native-pr',
    prUrl: 'https://github.com/Helicone/ai-gateway/pull/305',
  },
  {
    slug: 'posthog', name: 'PostHog', url: 'https://posthog.com/', repo: 'https://github.com/PostHog/posthog',
    description: 'Product analytics and LLM observability for product engineers.',
    category: 'Observability & infra', oss: true, status: 'compatible',
    setup: 'Instrument the OpenAI SDK pointed at https://openpaths.io/v1; PostHog auto-captures $ai events.',
  },
  {
    slug: 'bifrost', name: 'Bifrost (Maxim AI)', url: 'https://github.com/maximhq/bifrost', repo: 'https://github.com/maximhq/bifrost',
    description: 'Open-source LLM gateway behind Maxim AI; add OpenPaths as a provider in config.',
    category: 'Observability & infra', oss: true, status: 'compatible',
  },
  {
    slug: 'cloudflare-ai-gateway', name: 'Cloudflare AI Gateway', url: 'https://developers.cloudflare.com/ai-gateway/', repo: undefined,
    description: 'Monitor, control, and optimize AI apps; add OpenPaths as a custom provider.',
    category: 'Observability & infra', oss: false, status: 'listed',
  },

  // ---- Productivity ----
  {
    slug: 'analystos', name: 'analystOS', url: 'https://github.com/sheeki03/analystOS', repo: 'https://github.com/sheeki03/analystOS',
    description: 'AI research workspace with RAG over your docs and Notion automation.',
    category: 'Productivity', oss: true, status: 'native-pr',
    prUrl: 'https://github.com/sheeki03/analystOS/pull/4',
    setup: 'Set OPENROUTER_BASE_URL=https://openpaths.io/v1 and the key in config.',
  },
  {
    slug: 'roboflow-workflows', name: 'Roboflow Workflows', url: 'https://roboflow.com/workflows', repo: 'https://github.com/roboflow/inference',
    description: 'Visual builder for computer-vision pipelines; run any VLM via a model block.',
    category: 'Productivity', oss: true, status: 'native-pr',
    prUrl: 'https://github.com/roboflow/inference/pull/2458',
    setup: 'Use the OpenAI-compatible block with base URL https://openpaths.io/v1.',
  },
  {
    slug: 'aiassistworks', name: 'AiAssistWorks', url: 'https://aiassistworks.com/', repo: undefined,
    description: 'AI for Google Sheets, Slides & Docs with 100+ models.',
    category: 'Productivity', oss: false, status: 'listed',
  },
  {
    slug: 'octomind', name: 'Octomind', url: 'https://octomind.dev/', repo: undefined,
    description: 'Session-based AI dev assistant CLI with MCP tool execution.',
    category: 'Productivity', oss: false, status: 'listed',
  },
  {
    slug: 'postqode', name: 'PostQode', url: 'https://postqode.com/', repo: undefined,
    description: 'AI-powered SDLC platform with specialized testing agents.',
    category: 'Productivity', oss: false, status: 'listed',
  },
  {
    slug: 'ottex', name: 'Ottex', url: 'https://ottex.ai/', repo: undefined,
    description: 'Dictation + AI shortcuts with a custom OpenAI-compatible endpoint option.',
    category: 'Productivity', oss: false, status: 'listed',
  },
  {
    slug: 'spokenly', name: 'Spokenly', url: 'https://spokenly.app/', repo: undefined,
    description: 'Dictation app for macOS/Windows/iOS, free with your own keys.',
    category: 'Productivity', oss: false, status: 'listed',
  },
  {
    slug: 'maxim-ai', name: 'Maxim AI', url: 'https://getmaxim.ai/', repo: undefined,
    description: 'Agent simulation, evaluation, and observability platform.',
    category: 'Productivity', oss: false, status: 'listed',
  },
];
