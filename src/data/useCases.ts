export interface UseCaseRoute {
  need: string;
  model: string;
}

export interface UseCaseFaq {
  question: string;
  answer: string;
}

export interface UseCase {
  slug: string;
  title: string;
  metaTitle: string;
  metaDescription: string;
  hero: string;
  pains: string[];
  why: string[];
  routes: UseCaseRoute[];
  code: string;
  faq: UseCaseFaq[];
}

const CURL = (model: string, prompt: string) => `curl https://openpaths.io/v1/chat/completions \\
  -H "Authorization: Bearer $OPENPATHS_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "${model}",
    "messages": [
      {"role": "user", "content": "${prompt}"}
    ]
  }'`;

export const useCases: UseCase[] = [
  {
    slug: 'coding-agents',
    title: 'AI gateway for coding agents',
    metaTitle: 'AI Gateway for Coding Agents | OpenPaths',
    metaDescription:
      'One API key for coding agents across Claude, GPT, Codex, Kimi, and Qwen models. Auto-routing, automatic fallbacks, BYOK at $0 recorded cost, public latency stats.',
    hero: 'Run coding agents on one gateway with automatic provider fallback.',
    pains: [
      'Agents break when a single provider rate-limits or goes down mid-task.',
      'Comparing coding models means juggling separate keys, SDKs, and invoices per provider.',
      'No visibility into which model actually completes agentic tasks fastest.',
    ],
    why: [
      'One key reaches Claude, OpenAI, Moonshot, Qwen, and more through an OpenAI-compatible POST /v1/chat/completions.',
      'openpaths/auto-code routes each request to a coding-capable model without manual selection.',
      'Automatic fallback chains retry on another provider when one fails, so long agent sessions survive outages.',
      'Bring your own provider keys and requests bypass your OpenPaths balance entirely — cost is recorded as $0.',
      'Public stats at /stats show real per-model latency, TTFT, and throughput before you commit.',
    ],
    routes: [
      { need: 'Balanced default for agent loops', model: 'openpaths/auto-code' },
      { need: 'Frontier agentic coding', model: 'claude-sonnet-5' },
      { need: 'OpenAI codex-style tasks', model: 'gpt5.3-codex' },
      { need: 'Long-context repo work', model: 'moonshotai/kimi-k2.7-code' },
    ],
    code: CURL('openpaths/auto-code', 'Refactor the failing test in src/api/handler.ts'),
    faq: [
      {
        question: 'Can I point Claude Code at OpenPaths?',
        answer:
          'Yes. OpenPaths serves an Anthropic-native POST /v1/messages endpoint, so Claude Code works by setting ANTHROPIC_BASE_URL=https://openpaths.io.',
      },
      {
        question: 'What happens when my primary model is down?',
        answer:
          'OpenPaths maintains automatic fallback chains across providers. When one provider fails or rate-limits, the request retries on another route without changes to your code.',
      },
      {
        question: 'Do I have to give up my own Anthropic or OpenAI keys?',
        answer:
          'No. Add your own provider keys under Account → Provider keys. Requests on those keys bypass your OpenPaths balance entirely and are recorded as $0.',
      },
      {
        question: 'How do I pick between coding models?',
        answer:
          'Use openpaths/auto-code to let the gateway route automatically, or pin a specific id like claude-sonnet-5. Per-model latency and throughput from real usage are published at /stats.',
      },
    ],
  },
  {
    slug: 'customer-support-automation',
    title: 'AI gateway for customer support automation',
    metaTitle: 'AI Gateway for Customer Support Automation | OpenPaths',
    metaDescription:
      'Power support bots with one API key across Gemini, DeepSeek, Llama, and more. Fast auto-routing, fallbacks when providers fail, BYOK at $0, public latency stats.',
    hero: 'Answer every ticket with the right model, routed automatically.',
    pains: [
      'Support traffic spikes overwhelm a single provider’s rate limits.',
      'Cheap models handle volume but escalate quality; premium models waste money on simple replies.',
      'Switching providers to cut costs means rewriting integration code.',
    ],
    why: [
      'One key covers Google, DeepSeek, Meta, Mistral, xAI, and others through a single OpenAI-compatible chat endpoint.',
      'openpaths/auto-fast routes high-volume conversations to quick responses without pinning a vendor.',
      'routing_strategy supports price (default), config, or fastest — tune spend versus latency per workload.',
      'Automatic fallbacks keep bots responding when a provider degrades.',
      'BYOK lets you run volume on your existing provider contracts while OpenPaths records $0 against your balance.',
    ],
    routes: [
      { need: 'Fast conversational replies', model: 'openpaths/auto-fast' },
      { need: 'High-quality complex tickets', model: 'gemini-3.5-flash' },
      { need: 'Low-cost general assistance', model: 'deepseek-chat' },
      { need: 'Ultra-low-latency triage', model: 'llama-3.1-8b-instant' },
    ],
    code: CURL('openpaths/auto-fast', 'Draft a reply to this refund request within policy.'),
    faq: [
      {
        question: 'How do I control cost versus response speed?',
        answer:
          'Set routing_strategy to price for cheapest-first routing, config for explicit configuration, or fastest when latency matters most. price is the default.',
      },
      {
        question: 'Will my bot go dark if one provider has an outage?',
        answer:
          'No. Automatic fallback chains retry failed requests on another provider behind the same endpoint.',
      },
      {
        question: 'Which models work well for support volume?',
        answer:
          'Teams typically pair a fast small model like llama-3.1-8b-instant for triage with a stronger model like gemini-3.5-flash for escalations — all on the same key.',
      },
      {
        question: 'Is there a dashboard for model performance?',
        answer:
          'Yes. /stats publishes per-model average latency, time-to-first-token, and throughput computed from real usage logs, plus periodic live probes at /stats/model-probes.',
      },
    ],
  },
  {
    slug: 'content-pipelines',
    title: 'AI gateway for content pipelines',
    metaTitle: 'AI Gateway for Content Pipelines | OpenPaths',
    metaDescription:
      'Build writing and content pipelines on one API key across GPT, Claude, Grok, and open models. Auto-routing by price or speed, provider fallbacks, BYOK support.',
    hero: 'Draft, edit, and localize with any frontier model from one endpoint.',
    pains: [
      'Pipelines mix drafting, editing, and translation — no single model excels at all stages.',
      'Provider lock-in makes A/B testing writers slow and expensive.',
      'A single-provider outage halts scheduled publishing runs.',
    ],
    why: [
      'One key reaches OpenAI, Anthropic, xAI, MiniMax, and open-weight models via OpenAI-compatible POST /v1/chat/completions.',
      'openpaths/auto routes each stage to a suitable model; override with pinned ids where quality demands it.',
      'routing_strategy=price keeps bulk generation on budget; fastest prioritizes turnaround.',
      'Automatic fallback chains keep batch jobs running through provider incidents.',
      'BYOK applies your own provider keys with costs recorded as $0 against your OpenPaths balance.',
    ],
    routes: [
      { need: 'Hands-off stage routing', model: 'openpaths/auto' },
      { need: 'Long-form drafting', model: 'gpt-5.6' },
      { need: 'Editing and analysis', model: 'claude-sonnet-5' },
      { need: 'Bulk cheap generation', model: 'minimax-m2.7' },
    ],
    code: CURL('openpaths/auto', 'Turn these release notes into a blog outline and tweet thread.'),
    faq: [
      {
        question: 'Can I use different models for different pipeline stages?',
        answer:
          'Yes. Every call names its model explicitly, so drafting, editing, and localization can each hit different providers from the same key and base URL.',
      },
      {
        question: 'Does switching models require code changes?',
        answer:
          'Only the model field changes. The endpoint, auth header, and payload shape stay identical across all providers.',
      },
      {
        question: 'How do I keep batch generation affordable?',
        answer:
          'Use openpaths/auto with routing_strategy=price to prefer cheaper capable routes, or pin low-cost ids like minimax-m2.7 directly.',
      },
      {
        question: 'What if a provider fails mid-batch?',
        answer:
          'OpenPaths falls back automatically across providers, so scheduled runs continue instead of crashing on one vendor’s incident.',
      },
    ],
  },
  {
    slug: 'data-extraction',
    title: 'AI gateway for data extraction',
    metaTitle: 'AI Gateway for Structured Data Extraction | OpenPaths',
    metaDescription:
      'Extract structured data at scale with schema-first extraction models and cheap flash routes on one API key. Auto-routing, fallbacks, BYOK at $0, public speed stats.',
    hero: 'Turn documents into JSON without stitching together five vendors.',
    pains: [
      'Extraction volume makes frontier-model pricing unsustainable.',
      'Noisy HTML and messy documents break generic chat models.',
      'Provider-specific quirks mean rework whenever you switch extraction backends.',
    ],
    why: [
      'Specialized routes like inference-net/schematron-v2-turbo convert noisy HTML and documents into strict JSON.',
      'Cost-efficient models such as gemini-3.1-flash-lite and deepseek-v4-flash handle high-volume processing with JSON output.',
      'openpaths/auto-cheap routes bulk jobs toward inexpensive capable models automatically.',
      'Automatic fallbacks reroute around provider failures during long crawl jobs.',
      'Per-model latency and throughput from real usage are public at /stats, so you can size extraction batches on evidence.',
      'BYOK runs volume on your own provider accounts with $0 recorded against your OpenPaths balance.',
    ],
    routes: [
      { need: 'Schema-first document parsing', model: 'inference-net/schematron-v2-turbo' },
      { need: 'Cheap high-volume processing', model: 'openpaths/auto-cheap' },
      { need: 'Simple classification and tagging', model: 'gemini-3.1-flash-lite' },
      { need: 'Long documents with JSON output', model: 'deepseek-v4-flash' },
    ],
    code: CURL('inference-net/schematron-v2-turbo', 'Extract invoice_number, vendor, total, and due_date as strict JSON.'),
    faq: [
      {
        question: 'Which model should I use for messy HTML?',
        answer:
          'inference-net/schematron-v2-turbo is purpose-built for converting noisy HTML and documents into strict JSON.',
      },
      {
        question: 'How do I minimize extraction costs?',
        answer:
          'Route with openpaths/auto-cheap or pin efficient ids like gemini-3.1-flash-lite and deepseek-v4-flash. routing_strategy=price also biases toward cheaper routes.',
      },
      {
        question: 'Do extraction calls survive provider outages?',
        answer:
          'Yes. Automatic fallback chains retry failed requests on alternative providers behind the same endpoint.',
      },
      {
        question: 'Can I benchmark extraction speed before committing?',
        answer:
          '/stats publishes average latency, TTFT, and throughput per model from real usage logs, and /stats/model-probes runs periodic live probes.',
      },
    ],
  },
  {
    slug: 'education-tutoring',
    title: 'AI gateway for education and tutoring',
    metaTitle: 'AI Gateway for Education and Tutoring Apps | OpenPaths',
    metaDescription:
      'Build tutoring features on reasoning models from DeepSeek, OpenAI, Gemini, and Qwen with one API key. Step-by-step explanations, auto-routing, fallbacks, BYOK.',
    hero: 'Give students step-by-step tutoring on whichever model reasons best.',
    pains: [
      'Tutoring quality depends on reasoning strength, but frontier reasoning models sit behind separate vendor APIs.',
      'Interactive sessions die when a provider rate-limits mid-explanation.',
      'Student-facing budgets leave no room for paying premium prices on routine questions.',
    ],
    why: [
      'Dedicated reasoning routes like deepseek-reasoner deliver extended thinking for math and science help.',
      'openpaths/auto-reasoning sends hard problems to strong reasoners without manual model choice.',
      'One key covers OpenAI, Google, DeepSeek, Qwen, and more through a single OpenAI-compatible endpoint.',
      'routing_strategy=price keeps routine Q&A cheap while pinned ids handle hard derivations.',
      'Automatic fallback chains keep live tutoring sessions alive across provider incidents.',
      'BYOK lets schools run on their own provider keys with $0 recorded against the OpenPaths balance.',
    ],
    routes: [
      { need: 'Math and science reasoning', model: 'deepseek-reasoner' },
      { need: 'Auto-routed hard problems', model: 'openpaths/auto-reasoning' },
      { need: 'Fast everyday homework help', model: 'o4-mini' },
      { need: 'Multilingual tutoring', model: 'qwen/qwen3.7-plus' },
    ],
    code: CURL('deepseek-reasoner', 'Explain solving 2x^2 - 8x = 0 step by step.'),
    faq: [
      {
        question: 'Which model is best for step-by-step math?',
        answer:
          'Reasoning-focused routes like deepseek-reasoner produce extended thinking for derivations, and openpaths/auto-reasoning picks a strong reasoner automatically.',
      },
      {
        question: 'How do I keep student-facing costs low?',
        answer:
          'Send routine questions to fast small models like o4-mini and reserve reasoning routes for hard problems. routing_strategy=price biases routing toward cheaper options.',
      },
      {
        question: 'What happens if the reasoning provider goes down during a session?',
        answer:
          'OpenPaths retries on fallback providers automatically, so the session continues on the same API call shape.',
      },
      {
        question: 'Do we need separate accounts for each subject’s best model?',
        answer:
          'No. One OpenPaths key reaches DeepSeek, OpenAI, Google, Qwen, and other providers from a single endpoint.',
      },
    ],
  },
  {
    slug: 'creative-media',
    title: 'AI gateway for creative media',
    metaTitle: 'AI Gateway for Creative Media Generation | OpenPaths',
    metaDescription:
      'Generate images and video through one API key: Flux image editing, anime styles, Seedance text-to-video, and auto-image routing. Fallbacks, BYOK, public benchmarks.',
    hero: 'Images and video from multiple labs, billed through one account.',
    pains: [
      'Every image and video lab wants its own account, pricing page, and SDK.',
      'Iterating on styles means comparing generators manually with no shared billing.',
      'Render jobs fail silently when a single media provider has downtime.',
    ],
    why: [
      'One key covers images, video, audio, music, and more across providers including fal.ai and Black Forest Labs.',
      'flux-kontext-pro handles production image generation with instruction-based editing for iterative workflows.',
      'seedance-2.0-fast-text-to-video produces 720p cinematic clips with optional synchronized audio.',
      'openpaths/auto-image routes generation requests to suitable media models automatically.',
      'Automatic fallback chains rerender on another provider when one fails mid-job.',
      'Independent model comparisons are published at /evals and /image-evals; live speed probes run at /stats/model-probes.',
    ],
    routes: [
      { need: 'Instruction-based image editing', model: 'flux-kontext-pro' },
      { need: 'Text-to-video drafts', model: 'seedance-2.0-fast-text-to-video' },
      { need: 'Low-cost stylized art', model: 'zimage' },
      { need: 'Auto-routed image generation', model: 'openpaths/auto-image' },
    ],
    code: CURL('flux-kontext-pro', 'A neon-lit city street at night, cinematic wide shot'),
    faq: [
      {
        question: 'Do image and video calls use the chat completions endpoint?',
        answer:
          'Chat-shaped requests use POST /v1/chat/completions. Media generation models are reachable with the same single OpenPaths key across modalities.',
      },
      {
        question: 'Can I bring my fal.ai or Black Forest Labs keys?',
        answer:
          'Yes. Both fal (fal.ai) and bfl (Black Forest Labs) are supported BYOK providers. Requests on your keys bypass the OpenPaths balance and record $0.',
      },
      {
        question: 'Where can I compare creative models objectively?',
        answer:
          '/evals hosts the text leaderboard from Artificial Analysis data, /image-evals covers image generators, and /compare/* pages compare models head to head.',
      },
      {
        question: 'What if a video render fails because the provider is down?',
        answer:
          'Automatic fallback chains retry the request on an alternative provider, reducing lost render jobs during outages.',
      },
    ],
  },
];
