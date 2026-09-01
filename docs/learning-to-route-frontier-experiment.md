# Frontier-aware learning-to-route experiment

Status: research only. Nothing in this plan should change production routing until it beats the current route on held-out tasks.

## Question

Can a small, cheap model sweep learn a useful task frontier from real generated outcomes, then transfer the embedding and routing policy to larger models without assuming that model quality follows price?

The existing `learning-to-route` work gives us the starting point: embed tasks, retrieve nearby anchors, record per-model outcomes, and choose the cheapest model above a quality floor. The new experiment adds current catalog capability and price metadata as a weak prior, then lets verified task results do the work.

## First sweep

Start with models that are inexpensive enough to run repeatedly:

- `glm-5.3-flash`
- `deepseek-v4-flash`
- `gemini-3.1-flash-lite`
- `gemini-3.7-flash`

Treat `deepseek-v4-flash-vision-exp` as a separate multimodal lane rather than mixing image tasks into the text/coding table. Add a larger escalation arm only for a small calibration and holdout sample, for example `gpt-5.6-sol` or `deepseek-v4-pro`.

The model catalog supplies the version, availability, context, price, and capability metadata. If a price or model version changes, start a new arm or decay the old observations; never silently combine incompatible runs.

## What to record

Each generated task/model attempt should retain:

- task and task-family IDs, prompt hash, suite version, and verifier version
- model ID, provider route, model version, and catalog snapshot
- pass/fail or graded score from a deterministic verifier
- input/output tokens, cost, time to first token, total latency, retries, and escalation depth
- whether the result was chosen by the router, exploration, or a fixed baseline

The router should learn from the outcome fields, not from a leaderboard label. Capability data can initialize a prior or filter an impossible route, but it should not override repeated task-level evidence.

## Routing policy

1. Embed the task and retrieve its nearest anchors.
2. Estimate each model's pass probability or expected score from similarity-weighted outcomes, with time decay and a small prior for new models.
3. Choose the cheapest model above the quality floor.
4. Run the verifier. Escalate only on a failed or insufficient result.
5. Fold the verified outcome back into the anchor table.

Reserve a small exploration budget for new models and uncertain neighborhoods. Without exploration, a model that is never selected can never earn evidence.

## Transfer test

Transfer the embedding space and task-family routing policy, not the cheap model's success labels. A larger model gets a new model arm with its own prior and must be calibrated on held-out tasks. Compare:

- fixed cheapest model
- fixed best single model
- current hand-authored OpenPaths route
- embedding router without frontier metadata
- embedding router with metadata prior
- verify-and-escalate cascade

Use leave-task-family-out or time-ordered evaluation, not a random split that puts near-duplicate prompts in both train and test. Report pass rate, mean score, cost per solved task, escalation rate, TTFT, p95 latency, and calibration. Include confidence intervals once the suite is large enough.

## Stop conditions and write-up

Do not promote a rule because it wins on the 27-task development suite. Promote it only if it improves the cost/quality frontier on held-out generated tasks, remains stable after a model or price update, and does not increase verifier failures or tail latency.

After the holdout run, publish the task generator, model/version snapshot, raw outcome schema, routing policy, and negative results in a blog post. Until then, describe this as an experiment and keep the landing page metrics labeled as a research baseline.
