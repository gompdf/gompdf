# gompdf — Roadmap (English)

> Neutral, production-minded plan for a fully functional HTML/CSS → PDF engine in Go (no Chromium), inspired by [dompdf](https://github.com/dompdf/dompdf).

## 1) Vision

Build a native renderer that converts HTML + CSS to PDF through a full pipeline: **HTML → DOM → CSS → Cascade → Layout → Pagination → PDF paint**. Target a well-defined, testable subset of CSS 2.1 plus practical extensions (fonts, images, page headers/footers, tables) with strong typography (shaping + bidi) and server safety. Publish as a reusable Go SDK and a CLI.

## 2) Functional Scope (v1.0 “fully functional”)

**HTML elements:** `html/head/body`, headings `h1..h3`, `div/p/span/br/hr`, `ul/ol/li`, `a`, `img`, `table/thead/tbody/tfoot/tr/th/td`, `blockquote/pre/code`, `small/sup/sub`, `strong/em/b/i/u`, `figure/figcaption`.

**CSS (CSS 2.1 + practical bits)**

* **Box model:** `display: block|inline|inline-block|table|table-row|table-cell`; margin/padding/border; margin collapse; `box-sizing: content-box` (v1.0), `border` styles (solid/dashed/dotted), `border-radius` (all corners). Shadows/filters **out-of-scope for v1.0**.
* **Sizing/positioning:** `width/height` (px, %, auto), `min/max-*`, `position: static|relative|absolute|fixed`, `top/right/bottom/left`, `z-index` with basic stacking contexts.
* **Typography:** `font-family` (fallback chain), `font-style`, `font-weight` (100–900), `font-size` (px, pt, em, rem), `line-height`, `letter/word-spacing`, `text-align`, `text-indent`, `text-decoration` (underline/line-through), `white-space: normal|pre|pre-wrap`, `direction` + `unicode-bidi`.
* **Color/backgrounds:** `color`, `opacity` (paint only), `background-color`, `background-image` (url/data URI), `background-repeat`, `background-position`, `background-size: cover|contain|auto`.
* **Lists:** `list-style-type` (disc/circle/square/decimal/roman), `list-style-position`.
* **Tables:** auto & fixed layout, `border-collapse`, `vertical-align`, `rowspan/colspan`, repeat `thead` across page breaks.
* **Pagination:** `@page` margins, `page-break-before/after/inside` (and `break-*`), basic orphans/widows.
* **Fonts:** `@font-face` (TTF/OTF/WOFF/WOFF2; embed + subset). Local and remote.
* **Links:** internal anchors and external URLs with PDF annotations.
* **Explicitly not in v1.0:** flexbox/grid, filters, text-shadow/box-shadow, transforms, JavaScript, native SVG vector painting (SVG rasterized), audio/video.

**Resources:** PNG/JPEG/WebP (where feasible), SVG rasterized; external stylesheets; local/remote fonts; data URIs; optional resource cache.

**PDF features:** PDF 1.7, flate compression, font subsetting (TrueType/CFF), ToUnicode maps, hyperlinks/bookmarks, minimal XMP metadata.

## 3) Architecture

**Modules**

1. `parser/html` — DOM via x/net/html.
2. `parser/css` — CSS lexer/parser (tdewolff/parse); units, colors.
3. `style/cascade` — selectors (cascadia), specificity, inheritance, computed values.
4. `layout/tree` — box tree (block/inline/anonymous/table/replaced/positioned) & formatting contexts.
5. `layout/pagination` — content fragmentation, `@page`, breaks, orphans/widows.
6. `text/typography` — shaping (Harfbuzz), bidi (x/text/bidi), line breaking (UAX #14).
7. `render/pdf` — Canvas API + PDF backend (objects, resources, xref, compression, links).
8. `res` — resource loaders (file/http/data), limits, cache, safety.
9. `api` — public Go API + CLI plumbing.
10. `testkit` — fixtures, golden tests, raster diff utilities.

**Core data structures**

* `Node`, `StyledNode{node, computed}`, `ComputedStyle`.
* `Box` (BlockBox, InlineBox, AnonymousBlock, TableBox, TableRow, TableCell, ReplacedBox, AbsoluteBox, FixedBox), `LineBox`, `Page`.
* `FontFace` (family, weight, style), `GlyphRun` (font, size, dir, range, metrics), `Image`.

## 4) Detailed design

**4.1 CSS cascade** — Parse stylesheets, resolve selectors (specificity, order), inheritance; compute absolute units (px/pt), resolve percentages against containing box; register `@font-face` entries (lazy load, fallback chains).

**4.2 Layout algorithms**

* **Block formatting:** normal flow, vertical margin collapse, auto widths/heights, percentage resolution. BFC simplifications (full floats after v1.0).
* **Inline formatting:** segment into script/direction runs; shape with Harfbuzz; measure + line breaking via UAX #14; basic justify (space expansion); baseline alignment and `line-height` handling.
* **Positioned:** `relative` offsets; `absolute/fixed` with containing block resolution and simple stacking contexts.
* **Tables:** width algorithm (auto/fixed), cell height by content, simple border collapse, grid painting, repeat `thead` on page breaks.
* **Images:** intrinsic sizing, fallback DPI=96; `object-fit` equivalents via background sizing when applicable.

**4.3 Pagination** — Partition content respecting available page height, breaks (`page-break-*`), orphans/widows=2 default; header/footer callbacks with page numbers (Page X of Y).

**4.4 Typography** — Load TTF/OTF/WOFF/WOFF2, choose fonts by family/weight/style with fallback; per-document font subsetting; ToUnicode CMap; bidi ordering; ligatures & kerning where available.

**4.5 PDF painting** — Canvas primitives (text at (x,y), paths, fills/strokes, images, link rectangles); PDF object model (catalog, pages, resources, content streams), compression, metadata.

**4.6 Resources & safety** — Allow `file://`, `https://`, `data:` (remote disabled by default in server contexts). Max sizes, timeouts, whitelist/blacklist, controlled User-Agent, optional on-disk cache.

## 5) Public API & CLI

**Go SDK**

```go
package api

type Options struct {
    PageSize    string        // "A4", "Letter" (custom W×H later)
    MarginMm    [4]float64    // top,right,bottom,left
    BaseURL     string        // resolve relative URLs
    PrintBG     bool
    Header, Footer func(c Canvas, page, total int)
    Timeout     time.Duration
}

type Canvas interface {
    Text(x, y float64, run GlyphRun)
    Rect(x, y, w, h float64, stroke, fill bool)
    Image(x, y, w, h float64, img Image)
    LinkRect(x, y, w, h float64, url string)
}

func RenderHTMLToPDF(ctx context.Context, html []byte, opt Options) ([]byte, error)
```

**CLI**

```
gompdf render -in input.html -out out.pdf \
  --page A4 --margin 10,10,12,10 --base file:///path/ \
  --print-bg --header header.html --footer footer.html \
  --assets-allow-remote=false
```

## 6) CSS Compatibility Matrix (target)

Legend: ✅ supported in target, ⚠️ partial/limited, ⏩ planned (version), ❌ not planned.

| Area        | Property / Feature       | v0.1 | v0.3 | v0.4 | v1.0 |
| ----------- | ------------------------ | :--: | :--: | :--: | :--: |
| Box model   | display: block/inline    |   ✅  |   ✅  |   ✅  |   ✅  |
|             | display: inline-block    |   ⏩  |   ✅  |   ✅  |   ✅  |
|             | display: table/row/cell  |   ❌  |   ⏩  |   ✅  |   ✅  |
|             | margin/padding/border    |   ✅  |   ✅  |   ✅  |   ✅  |
|             | border-radius            |   ❌  |   ⏩  |   ✅  |   ✅  |
| Positioning | position: relative       |   ⏩  |   ✅  |   ✅  |   ✅  |
|             | position: absolute/fixed |   ❌  |   ⏩  |   ✅  |   ✅  |
| Sizing      | width/height px/%/auto   |   ✅  |   ✅  |   ✅  |   ✅  |
| Typography  | font-\*, line-height     |   ✅  |   ✅  |   ✅  |   ✅  |
|             | letter/word-spacing      |   ⏩  |   ✅  |   ✅  |   ✅  |
|             | text-decoration          |   ⏩  |   ✅  |   ✅  |   ✅  |
| Backgrounds | background-color         |   ✅  |   ✅  |   ✅  |   ✅  |
|             | background-image/repeat  |   ❌  |   ⏩  |   ✅  |   ✅  |
|             | background-position/size |   ❌  |   ⏩  |   ✅  |   ✅  |
| Lists       | list-style-type/position |   ⏩  |   ✅  |   ✅  |   ✅  |
| Tables      | auto/fixed layout        |   ❌  |   ⏩  |   ✅  |   ✅  |
|             | rowspan/colspan          |   ❌  |   ❌  |   ⏩  |   ✅  |
| Pagination  | @page margins            |   ⏩  |   ✅  |   ✅  |   ✅  |
|             | page-break-\* / break-\* |   ❌  |   ⏩  |   ✅  |   ✅  |
| Fonts       | @font-face (TTF/OTF)     |   ⏩  |   ✅  |   ✅  |   ✅  |
|             | WOFF/WOFF2               |   ❌  |   ⏩  |   ✅  |   ✅  |
| Links       | internal/external        |   ⏩  |   ✅  |   ✅  |   ✅  |

*Adjust matrix as implementation progresses. v0.1 aims for a printable subset; v1.0 covers the full table above.*

## 7) Quality & Testing

* **Unit tests:** parsers (HTML/CSS), specificity, unit conversions, block/inline layout, line-height, pagination rules.
* **Typography tests:** shaping (Latin/Arabic/Hebrew), bidi, ligatures/kerning; golden PDFs.
* **Pagination tests:** orphans/widows, page breaks, header/footer, repeating `thead`.
* **Table tests:** auto/fixed widths, rowspan/colspan, border collapse.
* **Golden testing:** render → PDF → rasterize to PNG (fixed DPI) → structural/visual diff (SSIM/percentage).
* **Fuzzing:** HTML/CSS parser packages.
* **Performance budgets:** ≤400 ms/page (A4, text+simple images) on typical x86\_64 server; ≤150 MB for a 20‑page document with moderate images.

## 8) Security

* Remote resources disabled by default (server mode). Allowlist protocols: `file:`, `https:`, `data:`.
* Size/time limits per resource; path sanitization with `BaseURL`.
* No JavaScript execution. SVG rasterized only.

## 9) Performance principles

* Two‑phase layout (measure → place) limited to changed regions.
* Glyph/metrics caches; per-font and per-run caching.
* Page streaming: emit PDF page-by-page to keep memory low.
* Buffer/object pools for allocations.

## 10) Project management (GitHub‑grade)

* **License:** Apache‑2.0; SPDX headers in `.go` files.
* **Contributions:** DCO (Signed‑off‑by required). Conventional Commits.
* **Branching:** trunk-based on `main`; protected branch; CI gates.
* **CI:** format (`gofumpt`), lint (`golangci-lint`), tests, `govulncheck`, optional CodeQL; Dependabot weekly.
* **Docs:** `README` (badges, quickstart), `docs/` (design, roadmap, getting-started), CSS compatibility matrix (this table).
* **Releases:** SemVer; tags `vX.Y.Z`; optional GoReleaser for binaries.

## 11) Delivery plan & milestones

**Phase A (Weeks 1–12) → v0.1**

* A1 Repo/CI/docs scaffold (Week 1)
* A2 HTML/CSS parsers + local loader (Weeks 2–3)
* A3 Cascade + computed styles (Week 4)
* A4 Block layout + inline basics (Weeks 5–7)
* A5 Typography (shaping+bidi) (Weeks 6–7)
* A6 Basic pagination + header/footer API (Week 8)
* A7 PDF backend (Week 9)
* A8 MVP CSS subset + tests/goldens (Weeks 10–11)
* A9 Release v0.1 (Week 12)

**Phase B (Weeks 13–24) → v0.3**

* B1 Positioning absolute/fixed + simple z-index (13–15)
* B2 Backgrounds image/repeat/size (16)
* B3 Tables I (no spans) + `thead` repeat (17–19)
* B4 Remote resources + cache + safety (20)
* B5 Stabilization, perf, bug fixes (21–24) → v0.3

**Phase C (Months 7–9) → v1.0**

* C1 Tables II (rowspan/colspan)
* C2 Full border-radius
* C3 Advanced page‑breaks, orphans/widows control
* C4 @font-face (WOFF/WOFF2) + subsetting
* C5 Optional hyphenation with dictionaries
* C6 Stabilization, docs, examples → v1.0

## 12) Definition of Done (per story)

* Unit + integration tests (golden when applicable).
* Documentation updated (README/docs/matrix).
* CI green (fmt/lint/test/vuln).
* No perf/memory regression beyond thresholds.

## 13) Roles

* **Core:** 1–2 engineers for layout/typography, 1 for PDF backend, 1 for parsers/cascade.
* **QA/Docs:** 1 person \~20–30% time on fixtures, goldens, docs.
* **Maintenance:** triage, reviews, releases, security.

## 14) Next steps (actionable)

1. Add this roadmap to `docs/roadmap.md` and link from README.
2. Open **milestone `v0.1`** with issues: parsers, cascade, block/inline layout, typography, pagination, PDF backend, testkit, CLI.
3. Create **project board** with Phase A columns; seed 20 real‑world templates (invoice/report/table) for goldens.
4. Enforce DCO + branch protection; set up CI (fmt/lint/test/vuln) and Dependabot.

5. UTF‑8 / Unicode font support (proper fix)
   - Embed a permissive default Unicode TTF (e.g., DejaVu Sans or Noto Sans) as the engine’s fallback family so UTF‑8 text (like • U+2022) renders correctly without user setup.
   - Register fonts via gofpdf’s UTF‑8 APIs and use the same family for both measurement and painting to keep layout consistent.
   - Map CSS `font-family`/`font-weight`/`font-style` to available faces (regular/bold/italic/bold‑italic), with fallback chaining to the default Unicode family, then core fonts as last resort.
   - Support user fonts through `@font-face` and `Options.FontDirectories`; load TTF/OTF (WOFF/WOFF2 in later phase) and embed subsets with ToUnicode maps.
   - Tests: golden fixtures covering bullets, accented characters, Cyrillic/Greek; assert `measureTextWidth` ≈ PDF width and visual correctness.
   - Docs/examples: show how to configure custom fonts and verify UTF‑8 output; update minimal example to demonstrate a bullet • and a non‑Latin sample.
