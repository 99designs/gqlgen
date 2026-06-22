# Citations & Bibliography for `RULES.md`

This file records the public sources behind the design principles in
[`RULES.md`](./RULES.md). Each source is defined once in **Part A** with a stable citation key,
and **Part B** maps each section of `RULES.md` to those keys.

## How to use it

1. Read a section in `RULES.md` (e.g. *§2 Interface Design*).
2. Look that section up in **Part B** to get its citation key(s) (e.g. `Ousterhout-PoSD-2018`).
3. Find the key in **Part A** for the full reference and links.

## Citation-key convention

`Author-ShortTitle-Year` — e.g. `Ousterhout-PoSD-2018`, `Johnson-StdPkgLayout-2016`,
`Ryer-HTTPServices-2024`. Books, talks/videos, and articles all use the same scheme.

## Scope

Covers **only** the sources `RULES.md` synthesizes — the six authors named in its header: Ben
Johnson, Gary Bernhardt, Mat Ryer, Michael Feathers, Mitchell Hashimoto, and John Ousterhout.

`RULES.md` is a gqlgen-specific adaptation of a broader Go-guidelines synthesis. The topics that
do **not** apply to a GraphQL code-generation library were deliberately dropped from `RULES.md`
and so are **not** cited here: distributed-systems and data-intensive design, enterprise
messaging/integration patterns, site-reliability engineering, observability metric methods, and
CLI command-pattern frameworks (gqlgen's CLI is built on `urfave/cli/v3`).

______________________________________________________________________

## Part A — Bibliography

### John Ousterhout

- **`Ousterhout-PoSD-2018`** — John Ousterhout, *A Philosophy of Software Design*. Yaknyam
  Press; 1st ed. 2018 (2nd ed. 2021). The source of `RULES.md`'s interface-design,
  naming/comments, and design-philosophy guidance (deep modules, the red-flag list, "make code
  obvious").

### Gary Bernhardt

- **`Bernhardt-FCIS-2012`** — Gary Bernhardt, "Functional Core, Imperative Shell," *Destroy All
  Software* screencast **DAS-0072** (2012). Public companion talk: "Boundaries" (2012). Source
  of the functional-core/imperative-shell and immutability-under-concurrency guidance.
  - Screencast: `https://www.destroyallsoftware.com/screencasts/catalog` (members-only);
    "Boundaries" talk available publicly (e.g. InfoQ/YouTube).

### Michael Feathers

- **`Feathers-WELC-2004`** — Michael Feathers, *Working Effectively with Legacy Code*. Prentice
  Hall, 2004. Backs the testing-as-design-pressure, naming, and design-philosophy points.
- **`Feathers-TestingPatience-2016`** — Michael Feathers, "Testing Patience" (talk; YOW! 2016,
  and a GeekFest delivery). Backs the "know which goal a test serves / don't test for coverage"
  framing.

### Ben Johnson — Go application architecture (blog series)

The project-layout and error-handling principles draw on Ben Johnson's Go architecture posts
(originally `medium.com/@benbjohnson`, later collected on `gobeyond.dev`).

- **`Johnson-StdPkgLayout-2016`** — "Standard Package Layout" (2016).
  `https://www.gobeyond.dev/standard-package-layout/`
- **`Johnson-StructuringApps-2014`** — "Structuring Applications in Go" (2014). Medium
  (`medium.com/@benbjohnson`).
- **`Johnson-PackagesAsLayers-2021`** — "Packages as Layers, Not Groups" (Jan 2021).
  `https://www.gobeyond.dev/packages-as-layers/`
- **`Johnson-FailureDomain-2018`** — "Failure Is Your Domain" (2018).
  `https://www.gobeyond.dev/failure-is-your-domain/`

### Mat Ryer

- **`Ryer-HTTPServices-2024`** — Mat Ryer, "How I Write HTTP Services in Go After 13 Years,"
  Grafana blog (2023/2024). Lineage: "…after eight years" (pace.dev, 2018) and Go Time #278.
  Backs the testing-mechanics notes (`getenv` injection, careful use of `t.Parallel()`).
  - `https://grafana.com/blog/how-i-write-http-services-in-go-after-13-years/`
  - earlier: `https://pace.dev/blog/2018/05/09/how-I-write-http-services-after-eight-years.html`

### Mitchell Hashimoto

- **`Hashimoto-AdvTesting-2017`** — Mitchell Hashimoto, "Advanced Testing with Go" (talk;
  GopherCon 2017). Backs the table-driven-test mechanics.
  - `https://www.youtube.com/watch?v=8hQG7QlcLBk`

______________________________________________________________________

## Part B — Section → source map

`RULES.md` tags most principle sections with an inline `*(Author)*` attribution; the keys below
follow those tags. Sections marked "—" are gqlgen-specific or process guidance with no external
design-principle source.

| `RULES.md` section | Citation key(s) |
| --- | --- |
| §0 What gqlgen Is | — (project context) |
| §1 Before You Write Code | `Johnson-StdPkgLayout-2016`, `Johnson-StructuringApps-2014`, `Johnson-PackagesAsLayers-2021` |
| §2 Interface Design | `Ousterhout-PoSD-2018` |
| §3 Functional Core / Imperative Shell | `Bernhardt-FCIS-2012`, `Feathers-WELC-2004` |
| §4 Error Handling | `Johnson-FailureDomain-2018` (gqlgen's `gqlerror` flow is project-specific) |
| §5 Context Discipline | — (general Go context conventions; gqlgen-specific) |
| §6 Concurrency | `Bernhardt-FCIS-2012` |
| §7 Testing | `Feathers-TestingPatience-2016`, `Feathers-WELC-2004`; mechanics — `Hashimoto-AdvTesting-2017`, `Ryer-HTTPServices-2024` |
| §8 Naming, Comments & Making Code Obvious | `Ousterhout-PoSD-2018`, `Feathers-WELC-2004` |
| §9 Design Philosophy | `Ousterhout-PoSD-2018`, `Feathers-WELC-2004` |
| §10 The CLI | — (built on `urfave/cli/v3`; no external design-principle source) |
| §11 Submitting Your Change — Checklist | — (process; cross-references the sections above) |
