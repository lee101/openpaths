import React, { useEffect, useMemo, useState } from 'react';
import { Check, Copy, Network, PlugZap } from 'lucide-react';
import { Link } from 'react-router-dom';
import { CodeBlock } from '../components/CodeBlock';
import { getAPIBaseURL, getStoredAPIKey } from '../lib/session';

type Integration = {
  id: string;
  name: string;
  install: string;
  summary: string;
  language: string;
  code: (apiBase: string, apiKey: string) => string;
  notes: string[];
};

const INTEGRATIONS: Integration[] = [
  {
    id: 'hermes-agent',
    name: 'Hermes Agent',
    install: 'git clone https://github.com/lee101/hermes-agent && cd hermes-agent && pip install -e .',
    summary: 'Hermes auto-detects OPENPATHS_API_KEY and can route its CLI or gateway agents through OpenPaths task-tier models.',
    language: 'bash',
    code: (apiBase, apiKey) => `export OPENPATHS_API_KEY="${apiKey}"
export OPENPATHS_BASE_URL="${apiBase}"

# Choose OpenPaths once, then chat normally.
hermes model openpaths:auto-medium-task
hermes

# Harder coding/research sessions can switch tiers without changing keys.
hermes model openpaths:auto-hard-task`,
    notes: [
      'OPENPATHS_API_KEY is detected automatically when Hermes resolves provider auth.',
      'Use auto-easy-task, auto-medium-task, auto-hard-task, auto-think, or autothink depending on task size.',
    ],
  },
  {
    id: 'openclaw',
    name: 'OpenClaw',
    install: 'curl -fsSL https://openclaw.ai/install.sh | bash -s -- --install-method git',
    summary: 'OpenClaw can onboard OpenPaths as a bundled provider, then expose OpenPaths models to agents and slash-command model switching.',
    language: 'bash',
    code: (apiBase, apiKey) => `export OPENPATHS_API_KEY="${apiKey}"
# OpenClaw uses the OpenPaths OpenAI-compatible base URL by default.
# Base URL: ${apiBase}

openclaw onboard --auth-choice openpaths-api-key \\
  --openpaths-api-key "$OPENPATHS_API_KEY"

openclaw models list --all --provider openpaths
openclaw models set openpaths/auto-medium-task

# In a chat, use /think medium or /think xhigh with OpenPaths auto models.`,
    notes: [
      'The OpenPaths provider uses the OpenAI-compatible endpoint at https://openpaths.io/v1.',
      'OpenClaw lists openpaths/auto, auto-easy-task, auto-medium-task, auto-hard-task, auto-think, and autothink.',
    ],
  },
  {
    id: 'langchain',
    name: 'LangChain',
    install: 'pip install langchain-openai',
    summary: 'Use ChatOpenAI with the OpenPaths base URL for agents, chains, tools, and structured output.',
    language: 'python',
    code: (apiBase, apiKey) => `from langchain_openai import ChatOpenAI

llm = ChatOpenAI(
    model="openai-chat-latest",
    api_key="${apiKey}",
    base_url="${apiBase}",
    temperature=0,
)

reply = llm.invoke("say hi and nothing else")
print(reply.content)`,
    notes: [
      'Works anywhere LangChain accepts a chat model.',
      'Use OpenPaths model IDs directly, including aliases like openai-chat-latest, grok-latest, and auto.',
    ],
  },
  {
    id: 'vercel-ai-sdk',
    name: 'Vercel AI SDK',
    install: 'npm install ai @ai-sdk/openai-compatible',
    summary: 'Create an OpenAI-compatible provider and use it with generateText, streamText, tools, and UI routes.',
    language: 'typescript',
    code: (apiBase, apiKey) => `import { generateText, streamText } from 'ai';
import { createOpenAICompatible } from '@ai-sdk/openai-compatible';

const openpaths = createOpenAICompatible({
  name: 'openpaths',
  apiKey: '${apiKey}',
  baseURL: '${apiBase}',
});

const { text } = await generateText({
  model: openpaths('openai-chat-latest'),
  prompt: 'say hi and nothing else',
});

console.log(text);

const stream = streamText({
  model: openpaths('auto'),
  prompt: 'write one sentence about model routing',
});

for await (const chunk of stream.textStream) {
  process.stdout.write(chunk);
}`,
    notes: [
      'Use the same provider object for generateText and streamText.',
      'For Next.js route handlers, return stream.toUIMessageStreamResponse() as usual.',
    ],
  },
  {
    id: 'pydantic-ai',
    name: 'PydanticAI',
    install: 'pip install "pydantic-ai-slim[openai]"',
    summary: 'Attach PydanticAI agents to OpenPaths with OpenAIProvider and OpenAIChatModel.',
    language: 'python',
    code: (apiBase, apiKey) => `from pydantic_ai import Agent
from pydantic_ai.models.openai import OpenAIChatModel
from pydantic_ai.providers.openai import OpenAIProvider

model = OpenAIChatModel(
    'openai-chat-latest',
    provider=OpenAIProvider(
        base_url='${apiBase}',
        api_key='${apiKey}',
    ),
)

agent = Agent(model, instructions='Be concise.')
result = agent.run_sync('say hi and nothing else')
print(result.output)`,
    notes: [
      'Structured outputs and tools keep using PydanticAI primitives.',
      'Use OpenPaths aliases to move between providers without changing agent code.',
    ],
  },
  {
    id: 'mastra',
    name: 'Mastra',
    install: 'npm install @mastra/core ai @ai-sdk/openai-compatible',
    summary: 'Mastra agents accept AI SDK language models, so the OpenPaths provider drops in as the model.',
    language: 'typescript',
    code: (apiBase, apiKey) => `import { Agent } from '@mastra/core/agent';
import { createOpenAICompatible } from '@ai-sdk/openai-compatible';

const openpaths = createOpenAICompatible({
  name: 'openpaths',
  apiKey: '${apiKey}',
  baseURL: '${apiBase}',
});

export const supportAgent = new Agent({
  id: 'support-agent',
  name: 'Support Agent',
  instructions: 'Answer in one short paragraph.',
  model: openpaths('auto-medium-task'),
});

const result = await supportAgent.generate('say hi and nothing else');
console.log(result.text);`,
    notes: [
      'Mastra workflows, memory, and tools stay unchanged.',
      'Switch models by changing only the OpenPaths model ID.',
    ],
  },
  {
    id: 'langfuse',
    name: 'Langfuse',
    install: 'pip install langfuse openai',
    summary: 'Trace OpenPaths calls with Langfuse by pointing the Langfuse OpenAI wrapper at OpenPaths.',
    language: 'python',
    code: (apiBase, apiKey) => `from langfuse import observe
from langfuse.openai import openai

openai.api_key = '${apiKey}'
openai.base_url = '${apiBase}'

@observe()
def run():
    response = openai.chat.completions.create(
        model='openai-chat-latest',
        messages=[{'role': 'user', 'content': 'say hi and nothing else'}],
    )
    return response.choices[0].message.content

print(run())`,
    notes: [
      'Set LANGFUSE_PUBLIC_KEY, LANGFUSE_SECRET_KEY, and LANGFUSE_BASE_URL in your environment.',
      'Langfuse captures the model name, latency, usage, and application trace around the OpenPaths call.',
    ],
  },
  {
    id: 'livekit',
    name: 'LiveKit Agents',
    install: 'pip install "livekit-agents[openai]~=1.4"',
    summary: 'Use the LiveKit OpenAI plugin with OpenPaths for LLM calls inside realtime voice agents.',
    language: 'python',
    code: (apiBase, apiKey) => `from livekit import agents
from livekit.agents import Agent, AgentSession
from livekit.plugins import openai

async def entrypoint(ctx: agents.JobContext):
    await ctx.connect()

    session = AgentSession(
        llm=openai.LLM(
            model='openai-chat-latest',
            api_key='${apiKey}',
            base_url='${apiBase}',
        ),
    )

    await session.start(
        room=ctx.room,
        agent=Agent(instructions='Say hi and nothing else.'),
    )`,
    notes: [
      'This routes LiveKit LLM turns through OpenPaths while keeping LiveKit room/session handling unchanged.',
      'For provider-native realtime speech APIs, use that provider plugin directly when a WebSocket bridge is required.',
    ],
  },
];

export function Integrations() {
  const [apiKey, setApiKey] = useState(() => getStoredAPIKey());
  const [copied, setCopied] = useState<string | null>(null);
  const apiBase = getAPIBaseURL();
  const exampleKey = apiKey || 'OPENPATHS_API_KEY';
  const integrations = useMemo(() => INTEGRATIONS, []);

  useEffect(() => {
    const sync = () => setApiKey(getStoredAPIKey());
    window.addEventListener('storage', sync);
    window.addEventListener('auth-change', sync);
    return () => {
      window.removeEventListener('storage', sync);
      window.removeEventListener('auth-change', sync);
    };
  }, []);

  const copyCode = async (id: string, code: string) => {
    await navigator.clipboard.writeText(code);
    setCopied(id);
    window.setTimeout(() => setCopied(null), 1500);
  };

  return (
    <section className="max-w-6xl mx-auto px-6 py-16">
      <div className="mb-12">
        <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-white/10 bg-white/[0.04] text-xs font-mono text-white/60 mb-6">
          <PlugZap className="w-3.5 h-3.5" />
          SDK Integrations
        </div>
        <h1 className="text-4xl md:text-5xl font-bold tracking-tight mb-4">Integrate OpenPaths With Your Agent Stack</h1>
        <p className="text-white/60 max-w-3xl font-light leading-relaxed">
          OpenPaths speaks OpenAI-compatible chat, image, audio, and embedding APIs. These examples cover agent runtimes like Hermes Agent
          and OpenClaw plus SDK patterns from LangChain, Vercel AI SDK, PydanticAI, Mastra, Langfuse, and LiveKit.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-10">
        <div className="border border-white/10 bg-white/[0.02] rounded-2xl p-5">
          <div className="text-xs font-mono text-white/40 mb-2">Base URL</div>
          <code className="text-sm text-white/80 break-all" data-testid="integrations-base-url">{apiBase}</code>
        </div>
        <div className="border border-white/10 bg-white/[0.02] rounded-2xl p-5">
          <div className="text-xs font-mono text-white/40 mb-2">API Key</div>
          <code className="text-sm text-white/80 break-all" data-testid="integrations-api-key">{exampleKey}</code>
        </div>
        <div className="border border-white/10 bg-white/[0.02] rounded-2xl p-5">
          <div className="text-xs font-mono text-white/40 mb-2">Model IDs</div>
          <div className="text-sm text-white/70 font-mono">auto-medium-task · auto-hard-task · openai-chat-latest</div>
        </div>
      </div>

      {!apiKey && (
        <div className="border border-white/10 bg-white/[0.02] rounded-2xl p-5 mb-10 text-sm text-white/60">
          Sign in on the <Link to="/account" className="text-white underline underline-offset-4">account page</Link> to auto-fill these examples with your real OpenPaths API key.
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {integrations.map(integration => {
          const code = integration.code(apiBase, exampleKey);
          return (
            <article key={integration.id} className="border border-white/10 bg-white/[0.02] rounded-2xl overflow-hidden" data-testid={`integration-${integration.id}`}>
              <div className="p-6 border-b border-white/10">
                <div className="flex items-start justify-between gap-4 mb-3">
                  <div>
                    <h2 className="text-2xl font-bold tracking-tight mb-2">{integration.name}</h2>
                    <p className="text-sm text-white/60 leading-relaxed">{integration.summary}</p>
                  </div>
                  <Network className="w-5 h-5 text-white/35 shrink-0 mt-1" />
                </div>
                <code className="block rounded-lg bg-black/50 border border-white/10 px-3 py-2 text-xs text-white/65 overflow-x-auto">
                  {integration.install}
                </code>
              </div>

              <div className="p-6">
                <div className="flex items-center justify-between mb-3">
                  <div className="text-xs font-mono text-white/40">Copy-paste example</div>
                  <button
                    onClick={() => copyCode(integration.id, code)}
                    className="inline-flex items-center gap-2 border border-white/10 px-3 py-1.5 rounded-lg text-xs font-mono text-white/70 hover:text-white hover:border-white/20 transition-colors"
                    data-testid={`copy-${integration.id}`}
                  >
                    {copied === integration.id ? <Check className="w-3.5 h-3.5 text-green-400" /> : <Copy className="w-3.5 h-3.5" />}
                    {copied === integration.id ? 'Copied' : 'Copy'}
                  </button>
                </div>
                <CodeBlock
                  code={code}
                  language={integration.language}
                  preClassName="rounded-xl border border-white/10 bg-black/60 p-4 text-xs leading-6 max-h-[420px]"
                  testId={`code-${integration.id}`}
                />
                <ul className="mt-4 space-y-2">
                  {integration.notes.map(note => (
                    <li key={note} className="text-sm text-white/50 leading-relaxed">{note}</li>
                  ))}
                </ul>
              </div>
            </article>
          );
        })}
      </div>
    </section>
  );
}
