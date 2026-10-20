import type { BlogPost } from './blog';

export const routingExperimentChecklist: BlogPost = {
  slug: 'llm-routing-cost-latency-quality-experiment-checklist',
  title: 'LLM Routing Experiments: A Cost, Latency and Quality Checklist',
  excerpt: 'Compare fixed-model and routed AI requests with a reproducible workload, quality gates, tail-latency measurements and cost per accepted task before increasing traffic.',
  date: '2026-10-14',
  author: 'OpenPaths Team',
  readTime: '5 min',
  tags: ['routing', 'evaluation', 'latency', 'cost', 'production'],
  content: `A model router should earn its place in your application. Comparing one inexpensive answer with one impressive answer is not an experiment: the prompts, output lengths, retries and acceptance criteria may all differ. Use this checklist to compare a fixed-model baseline with candidate routes on the same tasks. It describes a testing method, not measured OpenPaths performance or a promise of savings.

## 1. Write an acceptance rule before choosing a model

Start with the product outcome. For extraction, require the expected fields and correct values. For generated code, run the same tests in an isolated environment. For support drafts, use a review rubric covering accuracy, policy compliance and whether the answer resolves the question.

Separate hard failures from subjective preferences. A pleasant response that invents a refund policy is still a failure. Keep graders independent of route selection, and record changes to the rubric so a new score does not quietly mean a new standard.

## 2. Freeze a representative workload

Create a versioned set of tasks drawn from permitted, redacted examples. Include ordinary requests, long inputs, structured output, ambiguous instructions and cases that should refuse or ask a question. Keep a separate holdout set that you do not use to tune routing rules.

Record the prompt version, expected outcome, input size, output limit and required capabilities. Tool support, image input and schema compliance are eligibility constraints; a route that does not meet them should not win merely because it is inexpensive.

Use repeated trials rather than treating a single response as representative. Anthropic's [evaluation guide](https://www.anthropic.com/engineering/demystifying-evals-for-ai-agents) distinguishes task outcomes from transcripts and describes combining deterministic checks with calibrated review. Apply those distinctions to your own workload rather than copying someone else's benchmark score.

## 3. Compare routes under the same conditions

Run a fixed-model baseline and each candidate on the frozen workload. Keep generation settings, token ceilings and concurrency comparable, and record any settings a provider cannot support. Label cold and warm requests separately instead of mixing them into one average.

Start with serial checks for correctness, then run bounded concurrency tests on infrastructure you control. Decide the request count, maximum spend and stop conditions before starting. A routing experiment must not become an unbounded load test against an upstream service.

## 4. Measure the complete task

Record time to first useful output and time to completion separately. Summarize the distribution, including median and tail latency, and retain failures and timeouts in the report. Do not report only the requests that finished quickly.

For cost, include every attempt: retries, fallback calls and outputs rejected by your application. Calculate cost per accepted task as total measured spend divided by accepted tasks. When none pass, report no accepted tasks rather than manufacturing a favorable ratio. Label estimates separately from provider-reported or billed usage.

Keep quality, cost and latency visible as separate columns. A single blended score can hide an unacceptable regression in one dimension.

## 5. Test degraded conditions

Use mocked or controlled fixtures for rate limits, upstream timeouts, malformed responses and unavailable routes. Define whether to retry, fail or use a fallback for each case. Set explicit retry, token, spend and wall-clock budgets in your application; do not assume every router or provider enforces your desired limits.

Capture the route actually served when the response exposes it. If fallback changes model behavior or cost, show that in your experiment report. Never describe an unavailable route as successful because a different route eventually answered.

## 6. Roll out with a holdout and a rollback rule

Keep a small, predefined slice of eligible traffic on the baseline while evaluating a candidate. Set the quality floor, cost ceiling, latency threshold and minimum observation window in advance. Monitor user-visible outcomes as well as provider response codes.

Write down what triggers rollback and who owns it. Re-run the frozen workload after changing prompts, models, provider settings or routing rules. Offline results are evidence for a rollout decision, not proof that production traffic will behave identically.

## Try the checklist with OpenPaths

[Explore the model catalog](/models) to identify candidates with the capabilities your tasks require. [Read the API docs](/docs) before wiring a reproducible harness, or [try a prompt in the Playground](/playground) for an initial manual check.

When you are ready for a budgeted comparison, [review current pricing](/pricing) and [open your account](/account) to manage access and prepaid credits. Start with a bounded experiment; buy only the credits your planned workload needs. Prices, availability and results can change, so record the date and settings of every comparison.

## Before increasing traffic

- Acceptance rules and the holdout workload are versioned.
- Candidate routes satisfy the required modalities and output contracts.
- Reports include failed requests, tail latency and every paid attempt.
- Retry, fallback and spend budgets have controlled tests.
- Production monitoring and rollback ownership are ready.

The useful winner is the route that meets your quality bar within your application's budget and latency limits. Measure that trade-off on your own tasks before expanding usage.`,
};
