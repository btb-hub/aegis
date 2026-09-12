# Habr Article About Aegis and AI Harness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Produce a publication-ready Russian Habr article about building Aegis with AI, explain the value of a mature harness and BMAD, and provide four consistent illustrations.

**Architecture:** Keep research notes, the article, and visual assets as separate deliverables under `docs/habr/`. Build the narrative from verified project facts and primary external sources, then run a dedicated editorial and visual verification pass before finalizing the article.

**Tech Stack:** Markdown, Git, built-in image generation, deterministic SVG, shell verification commands

## Global Constraints

- Article language: Russian.
- Article length: 2500 to 3500 words, targeting 12 to 15 minutes of reading.
- Article title: `Opsgenie ушёл, JSM не пришёл: как я собрал собственный incident management с AI`.
- Aegis positioning: working open-source beta with explicit limitations.
- Audience: experienced developers, team leads, Heads of Development, and CTOs.
- BMAD should occupy about one third of the article and remain grounded in the Aegis case.
- Career positioning must remain implicit and emerge from decisions, scope control, delivery, and honesty.
- Use only the ordinary hyphen character `-` in the final article. Do not use en dash or em dash characters.
- Use primary sources for changing external facts and place links next to the claims they support.
- Treat negative experience with OpsKnight as the author's experience, not a universal factual verdict.
- Separate verified GSD events from interpretation and avoid unverified accusations.
- Visual direction: technical editorial cartoon, warm off-white paper, black ink, coral-red accent.
- Generated illustrations must not contain logos, long embedded text, watermarks, or product UI replicas.

---

### Task 1: Build the factual source sheet

**Files:**
- Create: `docs/habr/aegis-ai-harness-sources.md`
- Read: `README.md`
- Read: `docs/00-product-brief.md`
- Read: `docs/01-prd.md`
- Read: `docs/02-architecture.md`
- Read: `docs/07-setup-deployment.md`
- Read: `docs/10-agent-loop.md`
- Read: `docs/features/incident-management.md`
- Read: `docs/features/alerting.md`

**Interfaces:**
- Consumes: approved article design and current repository state
- Produces: verified claims, current metrics, source URLs, and wording constraints for the article draft

- [ ] **Step 1: Record product facts**

Create sections for product purpose, target users, core incident flow, integrations, deployment, architecture, agent loop, current gaps, and post-MVP scope. Every statement must point to a local source path.

- [ ] **Step 2: Refresh repository metrics**

Run:

```bash
git rev-list --count HEAD
git ls-files | awk 'END { print NR }'
git ls-files '*_test.go' '*.spec.ts' '*.test.ts' '*.test.tsx' | awk 'END { print NR }'
git log --reverse --format='%ad %h %s' --date=short | sed -n '1p;$p'
```

Record the results with the exact command and date. Do not reuse the earlier `98 commits` value if the new article-spec commits changed it.

- [ ] **Step 3: Record primary external sources**

Include direct links and a one-sentence supported claim for:

```text
Atlassian Opsgenie licensing and end-of-support dates
Atlassian Opsgenie migration to Jira Service Management
Official OpsKnight site or repository
Official BMAD Method repository and documentation
Official GSD project materials
Community-maintained GSD fork announcement, clearly labeled as a community source
```

- [ ] **Step 4: Mark claim risk**

Tag each external claim as one of:

```text
VERIFIED_PRIMARY
VERIFIED_COMMUNITY
AUTHOR_EXPERIENCE
AUTHOR_INTERPRETATION
```

Rewrite any claim that cannot be placed in one category.

- [ ] **Step 5: Verify the source sheet**

Run:

```bash
rg -n 'TBD|TODO|FIXME|XXX' docs/habr/aegis-ai-harness-sources.md
rg -n 'VERIFIED_PRIMARY|VERIFIED_COMMUNITY|AUTHOR_EXPERIENCE|AUTHOR_INTERPRETATION' docs/habr/aegis-ai-harness-sources.md
```

Expected: the first command returns no matches; the second returns every external claim.

- [ ] **Step 6: Commit**

```bash
git add docs/habr/aegis-ai-harness-sources.md
git commit -m "docs: collect sources for Aegis Habr article"
```

---

### Task 2: Write the complete article draft

**Files:**
- Create: `docs/habr/aegis-ai-harness.md`
- Read: `docs/habr/aegis-ai-harness-sources.md`
- Read: `docs/superpowers/specs/2026-07-26-habr-aegis-ai-harness-article-design.md`

**Interfaces:**
- Consumes: verified source sheet and approved editorial design
- Produces: complete Russian article without visual assets

- [ ] **Step 1: Write the hook and problem**

Open with Opsgenie, the JSM/self-hosted Jira constraint, and the failed search for a suitable alternative. Introduce the one-week promise and the actual two-calendar-week evening schedule.

- [ ] **Step 2: Explain the product scope through one alert**

Trace a single alert through:

```text
webhook -> storage -> deduplication -> routing -> incident -> Jira -> page -> acknowledge or escalation -> timeline and analytics
```

Use this flow to introduce Aegis modules without copying the README feature list.

- [ ] **Step 3: Introduce the harness problem**

Explain why one-shot prompting produces locally plausible code but does not maintain product intent. Use specific Aegis artifacts: product brief, PRD, architecture, data model, API contract, stories, design system, gates, and Definition of Done.

- [ ] **Step 4: Explain the delivery loop**

Show the exact loop:

```text
story -> short plan -> vertical slice with tests -> lint/type/test -> self-review -> PR -> merge
```

Connect each gate to a concrete failure it prevents.

- [ ] **Step 5: Explain BMAD**

Cover specialized roles, discovery, planning, architecture, implementation, verification, blocked states, and shared artifacts. Emphasize that BMAD helps the developer operate across a wider responsibility surface. State explicitly that the manager benefits from a developer who uses BMAD, not from BMAD as a management dashboard.

- [ ] **Step 6: Add the honest beta status**

Describe working backend flows, integrations, worker jobs, deployment, and UI areas. Name the incident UI fixture/API-wiring gap and avoid language that implies full production readiness.

- [ ] **Step 7: Add the GSD and OpsKnight lessons**

Keep OpsKnight framed as a mismatch with the author's requirements and experience. Credit GSD's context-engineering idea, then describe the later crypto association and trust loss using the risk labels from the source sheet.

- [ ] **Step 8: Close with lessons and what would change**

End with three practical lessons and three changes for a second attempt: BMAD from project start, earlier end-to-end UI wiring, and time/token cost tracking.

- [ ] **Step 9: Verify structure and length**

Run:

```bash
awk 'NF { words += NF } END { print words }' docs/habr/aegis-ai-harness.md
rg -n '^## ' docs/habr/aegis-ai-harness.md
rg -n '[–—]' docs/habr/aegis-ai-harness.md
```

Expected: 2500 to 3500 words, all planned sections present, and no en dash or em dash matches.

- [ ] **Step 10: Commit**

```bash
git add docs/habr/aegis-ai-harness.md
git commit -m "docs: draft Aegis AI harness Habr article"
```

---

### Task 3: Perform the technical and editorial review

**Files:**
- Modify: `docs/habr/aegis-ai-harness.md`
- Read: `docs/habr/aegis-ai-harness-sources.md`

**Interfaces:**
- Consumes: complete article draft and source sheet
- Produces: fact-checked, stylistically consistent article text ready for visuals

- [ ] **Step 1: Check every number and changing fact**

Compare dates, commit counts, test-file counts, coverage claims, supported integrations, deployment model, and current limitations against the source sheet and current repository.

- [ ] **Step 2: Check source placement**

Ensure every Atlassian, BMAD, OpsKnight, and GSD claim has an adjacent Markdown link. Do not collect changing facts into an unreferenced bibliography only.

- [ ] **Step 3: Run the self-promotion test**

Remove sentences that describe the author as strategic, senior, visionary, leadership-oriented, or CTO-ready. Keep evidence of those qualities through decisions and outcomes.

- [ ] **Step 4: Run the humor test**

Keep humor aimed at the author, the process, and general AI behavior. Remove jokes aimed at individual maintainers or users of competing projects.

- [ ] **Step 5: Run the honesty test**

Confirm the article contains all three:

```text
The two weeks were calendar time with 1 to 2 hours per day.
Aegis is an open-source beta.
The incident UI still has an API-wiring gap.
```

- [ ] **Step 6: Run mechanical checks**

```bash
rg -n '[–—]' docs/habr/aegis-ai-harness.md
rg -n 'TBD|TODO|FIXME|XXX' docs/habr/aegis-ai-harness.md
awk 'NF { words += NF } END { print words }' docs/habr/aegis-ai-harness.md
git diff --check
```

Expected: no forbidden dash characters, no placeholders, 2500 to 3500 words, and no whitespace errors.

- [ ] **Step 7: Commit**

```bash
git add docs/habr/aegis-ai-harness.md
git commit -m "docs: review Aegis Habr article"
```

---

### Task 4: Generate three editorial illustrations

**Files:**
- Create: `docs/habr/assets/aegis-cover.png`
- Create: `docs/habr/assets/vibe-coding-vs-harness.png`
- Create: `docs/habr/assets/two-week-build.png`

**Interfaces:**
- Consumes: approved technical-felieton visual direction
- Produces: three consistent raster illustrations without embedded text

- [ ] **Step 1: Generate the cover**

Use the built-in image generation tool with this prompt:

```text
Use case: illustration-story
Asset type: wide editorial cover illustration for a Russian technology article
Primary request: An engineer is assembling a compact incident-management shield machine from precise mechanical parts while a cheerful AI robot offers one obviously incompatible gear. The engineer looks amused rather than frustrated.
Style/medium: sophisticated European newspaper editorial cartoon, hand-drawn black ink, subtle warm paper grain, flat coral-red accent, limited palette
Composition/framing: wide 16:9 composition, engineer and machine on the right two thirds, calm negative space on the left for the publication title
Lighting/mood: warm, intelligent, lightly self-ironic
Constraints: no text, no logos, no recognizable product interfaces, no watermark, no cyberpunk neon, no humanoid glamour robot
```

- [ ] **Step 2: Generate the harness comparison**

Use the built-in image generation tool with this prompt:

```text
Use case: illustration-story
Asset type: horizontal in-article editorial illustration
Primary request: A split scene with the same small AI robot. On the left it sprints chaotically while dropping loose code files, tangled cables, and mismatched parts. On the right it moves through an orderly engineering workshop with labeled-looking but text-free stations represented by a blueprint, architecture blocks, story cards, tests, and a green verification lamp.
Style/medium: same sophisticated newspaper cartoon, hand-drawn black ink, warm paper grain, coral-red accent
Composition/framing: balanced wide split composition with a clear visual transformation from chaos to controlled flow
Constraints: no readable text, no logos, no watermark, no product UI, no mocking facial expression
```

- [ ] **Step 3: Generate the two-week build illustration**

Use the built-in image generation tool with this prompt:

```text
Use case: illustration-story
Asset type: compact horizontal editorial spot illustration
Primary request: An engineer works calmly at a desk for a short evening session under a small lamp while two weekly calendar sheets turn overhead. A tiny one-week estimate note has stretched into two weeks, shown only through visual calendar structure, not readable words.
Style/medium: same sophisticated newspaper cartoon, hand-drawn black ink, warm paper grain, coral-red accent
Composition/framing: simple readable silhouette, generous whitespace, understated humor
Constraints: no readable text, no logos, no watermark, no exhausted or unhealthy work imagery
```

- [ ] **Step 4: Save all selected outputs**

Copy the selected generated files from the tool output location to the exact workspace paths above. Do not overwrite unrelated existing assets.

- [ ] **Step 5: Inspect each output**

Use image inspection on all three files. Confirm:

```text
Consistent ink and paper style
No accidental readable text
No logos or watermarks
Correct wide composition
Cover has usable negative space
No extra hands, tools, or malformed mechanical parts that distract from the metaphor
```

- [ ] **Step 6: Commit**

```bash
git add docs/habr/assets/aegis-cover.png docs/habr/assets/vibe-coding-vs-harness.png docs/habr/assets/two-week-build.png
git commit -m "docs: add editorial illustrations for Habr article"
```

---

### Task 5: Create the deterministic BMAD and Aegis flow diagram

**Files:**
- Create: `docs/habr/assets/bmad-aegis-flow.svg`

**Interfaces:**
- Consumes: approved process `idea -> PRD -> architecture -> stories -> implementation -> verification -> working beta`
- Produces: readable SVG diagram matching the article's visual system

- [ ] **Step 1: Create the SVG**

Build a 1400 by 500 SVG with seven rounded cards, directional arrows, Russian labels, and a small feedback loop from verification to stories. Use warm off-white background, black lines, coral-red emphasis for human decision points, and green only for the final verified state.

- [ ] **Step 2: Validate XML**

Run:

```bash
xmllint --noout docs/habr/assets/bmad-aegis-flow.svg
```

Expected: exit code 0 and no output.

- [ ] **Step 3: Inspect the rendered diagram**

Open or render the SVG and verify labels, arrow direction, spacing, and readability at article width. Confirm that the feedback loop does not cross labels.

- [ ] **Step 4: Commit**

```bash
git add docs/habr/assets/bmad-aegis-flow.svg
git commit -m "docs: add BMAD and Aegis workflow diagram"
```

---

### Task 6: Integrate visuals and perform final publication checks

**Files:**
- Modify: `docs/habr/aegis-ai-harness.md`
- Read: `docs/habr/assets/aegis-cover.png`
- Read: `docs/habr/assets/vibe-coding-vs-harness.png`
- Read: `docs/habr/assets/two-week-build.png`
- Read: `docs/habr/assets/bmad-aegis-flow.svg`

**Interfaces:**
- Consumes: reviewed article and four approved assets
- Produces: final Habr-ready Markdown package

- [ ] **Step 1: Place illustrations**

Add relative Markdown image links:

```markdown
![Обложка статьи об Aegis и AI-harness](assets/aegis-cover.png)
![Vibe coding и разработка с harness](assets/vibe-coding-vs-harness.png)
![Две календарные недели вечерней разработки](assets/two-week-build.png)
![Путь от идеи до проверенной beta](assets/bmad-aegis-flow.svg)
```

Place the cover after the introductory lead, the comparison after the harness problem, the
two-week illustration after the time clarification, and the diagram in the BMAD section.

- [ ] **Step 2: Run final text checks**

```bash
rg -n '[–—]' docs/habr/aegis-ai-harness.md
rg -n 'TBD|TODO|FIXME|XXX' docs/habr/aegis-ai-harness.md
awk 'NF { words += NF } END { print words }' docs/habr/aegis-ai-harness.md
git diff --check
```

Expected: no forbidden dash characters, no placeholders, 2500 to 3500 words, and no whitespace errors.

- [ ] **Step 3: Run final asset checks**

```bash
test -s docs/habr/assets/aegis-cover.png
test -s docs/habr/assets/vibe-coding-vs-harness.png
test -s docs/habr/assets/two-week-build.png
xmllint --noout docs/habr/assets/bmad-aegis-flow.svg
```

Expected: all commands exit 0.

- [ ] **Step 4: Review the complete diff**

Run:

```bash
git diff --stat
git status --short
```

Confirm only the planned article package and any explicitly approved source notes are included.

- [ ] **Step 5: Commit**

```bash
git add docs/habr/aegis-ai-harness.md docs/habr/assets
git commit -m "docs: finalize Aegis Habr article package"
```
