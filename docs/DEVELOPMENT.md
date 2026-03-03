# Documentation Site — Development Guide

The documentation site is a **Jekyll-powered static site** deployed to GitHub Pages. Content is written in **Markdown** and automatically converted to HTML by Jekyll.

GitHub Pages is configured to serve from the `docs/` directory on the `main` branch.

---

## Architecture Overview

### Jekyll Structure

The site uses GitHub Pages' **native Jekyll support** — no CI/CD build step required. Push Markdown files and GitHub builds the HTML automatically.

| File/Directory | Purpose |
|----------------|---------|
| `_config.yml` | Jekyll configuration: plugins, excludes, defaults |
| `_layouts/default.html` | Page layout: header, sidebar, content area, search overlay |
| `_layouts/home.html` | Home page layout (no prev/next nav) |
| `_includes/header.html` | Shared header: logo, search, theme toggle, GitHub link |
| `_includes/page-nav.html` | Previous/next page navigation (auto-generated from nav data) |
| `_data/nav.yml` | **Source of truth** for sidebar navigation, page ordering, and prev/next |
| `css/style.css` | Full stylesheet: light + dark themes, responsive breakpoints, Rouge syntax highlighting |
| `js/app.js` | Client-side JS: theme toggle, search, TOC generation, copy buttons |
| `index.md` | Home page content |
| `404.md` | Custom 404 page |
| `Gemfile` | Ruby dependencies for local preview |
| `pages/` | **Legacy** — old SPA HTML fragments (excluded from build) |

### Content Pages

Each page is an `index.md` file inside a directory matching its URL path:

```
docs/
├── getting-started/
│   ├── introduction/index.md   → /getting-started/introduction/
│   ├── installation/index.md   → /getting-started/installation/
│   └── quickstart/index.md     → /getting-started/quickstart/
├── concepts/
│   ├── agents/index.md         → /concepts/agents/
│   └── ...
└── ...
```

Each page has YAML front matter:

```yaml
---
title: "Page Title"
description: "Brief description for meta tags"
permalink: /section/slug/
---

# Page Title
*Brief description for meta tags*

Content in Markdown...
```

### Navigation

Navigation is defined in `_data/nav.yml` — a flat list of sections and items:

```yaml
- section: Getting Started
  items:
    - title: Introduction
      url: /getting-started/introduction/
    - title: Installation
      url: /getting-started/installation/
```

This drives:
- **Sidebar navigation** — auto-rendered with active state highlighting
- **Previous/next links** — computed from the flat ordering of all pages
- **Search index** — built client-side from sidebar links

### How to Add a New Page

1. Create `section/slug/index.md` with front matter (see above).
2. Add an entry to `_data/nav.yml` in the appropriate section.
3. That's it. Sidebar, search, prev/next, and TOC are all automatic.

### Styling Conventions Used in Pages

Pages use these patterns (all defined in `css/style.css`):

| Element | Usage |
|---------|-------|
| `*subtitle text*` | Italic subtitle below `# Heading`, used on every page |
| `<div class="callout callout-info/tip/warning">` | Admonition boxes. Contain `<div class="callout-title">` + `<p>` |
| `<div class="cards">` + `<a class="card">` | Grid of clickable cards with `card-icon`, `h3`, `p` |
| `<div class="features-grid">` + `<div class="feature">` | Feature showcase grid (home + intro pages) |
| ` ```yaml ` / ` ```bash ` / ` ```go ` | Fenced code blocks with Rouge syntax highlighting |
| `<kbd>Ctrl+K</kbd>` | Keyboard shortcut badges |
| `<div class="hero">` | Hero banner (only on home page) |
| `<div class="demo-container"><img>` | Demo GIF container |

**Important:** Markdown inside HTML `<div>` blocks is **not processed** by kramdown. Use HTML tags (`<h3>`, `<p>`, `<a>`, `<code>`) inside callouts, cards, and feature grids.

Internal links use standard Markdown or HTML:
```markdown
See the [Quick Start](/getting-started/quickstart/) guide.
```

### Theme System

- Light is default. Dark activates via `[data-theme="dark"]` on `<html>`.
- Persisted in `localStorage` key `cagent-docs-theme`.
- Auto-detects `prefers-color-scheme: dark` on first visit.
- Toggle button in header calls `toggleTheme()`.

### Search

Search is **client-side** — the index is built from sidebar navigation links on page load. Supports title and section matching. Opens with ⌘K / Ctrl+K or by clicking the search input.

### Table of Contents

Long pages automatically get a sticky right-side TOC generated from `<h2>` and `<h3>` headings. The TOC:
- Appears on screens ≥1280px
- Uses scroll-spy to highlight the current section
- Supports click-to-scroll
- Auto-hides for short pages (<3 headings)

### Syntax Highlighting

Code blocks use **Rouge** (Jekyll's built-in highlighter) with a custom dark theme defined in `css/style.css`. The color scheme matches One Dark Pro:

```css
.highlight .na { color: #e06c75; }  /* YAML keys */
.highlight .s  { color: #98c379; }  /* strings */
.highlight .k  { color: #c678dd; }  /* keywords */
.highlight .nf { color: #61afef; }  /* functions */
```

---

## How to Preview Locally

### Option 1: Jekyll (recommended)

```bash
# Install Jekyll (macOS)
brew install ruby
gem install jekyll jekyll-relative-links jekyll-optional-front-matter jekyll-readme-index jekyll-seo-tag

# Serve with live reload
cd docs
jekyll serve --livereload
# Open http://localhost:4000
```

### Option 2: Static file server (quick preview, no Jekyll processing)

```bash
cd docs/_site  # After a jekyll build
python3 -m http.server 8000
# Open http://localhost:8000
```

## How to Deploy to GitHub Pages

Configure GitHub Pages to serve from the `docs/` directory on the `main` branch:

1. In repo Settings → Pages, set source to **Deploy from a branch**.
2. Select the `main` branch and `/docs` folder.

GitHub automatically runs Jekyll on every push — no CI/CD workflow needed.

## Re-recording the Demo GIF

The demo GIF is recorded using [VHS](https://github.com/charmbracelet/vhs). Tape files live in `recordings/`:

```bash
# Re-record the main demo
vhs ./docs/recordings/demo.tape

# The output goes to docs/demo.gif
```
