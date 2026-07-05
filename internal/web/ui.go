package web

import . "github.com/n0remac/GoDom/html"

func pageHead(title, extraCSS string, extraNodes ...*Node) *Node {
	nodes := []*Node{
		Meta(Charset("UTF-8")),
		Meta(Name("viewport"), Content("width=device-width, initial-scale=1.0")),
		Title(T(title)),
		Script(Src("https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4")),
		Style(Raw(sharedPageCSS() + extraCSS)),
	}
	nodes = append(nodes, extraNodes...)
	return Head(nodes...)
}

func uiPrimaryButtonClass(size string) string {
	return "inline-flex items-center justify-center rounded-md border border-emerald-300/30 bg-emerald-300 px-4 font-semibold text-zinc-950 shadow-sm transition duration-150 hover:-translate-y-0.5 hover:bg-emerald-200 focus:outline-none focus:ring-4 focus:ring-emerald-300/25 disabled:cursor-not-allowed disabled:opacity-60 " + uiButtonSizeClass(size)
}

func uiSecondaryButtonClass(size string) string {
	return "inline-flex items-center justify-center rounded-md border border-[var(--app-border-strong)] bg-[var(--app-panel)] px-4 font-medium text-[var(--app-fg)] shadow-sm transition duration-150 hover:-translate-y-0.5 hover:border-emerald-300/50 hover:bg-emerald-300/10 focus:outline-none focus:ring-4 focus:ring-emerald-300/15 disabled:cursor-not-allowed disabled:opacity-60 " + uiButtonSizeClass(size)
}

func uiInputClass() string {
	return "w-full rounded-md border border-[var(--app-border-strong)] bg-[var(--app-panel)] px-3 py-2.5 text-sm text-[var(--app-fg)] shadow-sm outline-none transition placeholder:text-[var(--app-fg-soft)] focus:border-emerald-300/70 focus:ring-4 focus:ring-emerald-300/15"
}

func uiButtonSizeClass(size string) string {
	switch size {
	case "xs":
		return "h-8 px-3 text-[0.68rem] uppercase tracking-[0.18em]"
	case "sm":
		return "h-9 px-3.5 text-sm"
	default:
		return "h-10 px-4 text-sm"
	}
}

func sharedPageCSS() string {
	return `
:root {
  color-scheme: dark;
  --app-bg: rgb(17 17 17);
  --app-bg-depth: rgb(31 41 51);
  --app-bg-glow: rgba(16, 185, 129, 0.16);
  --app-surface: rgba(24, 24, 27, 0.76);
  --app-surface-muted: rgba(24, 24, 27, 0.96);
  --app-panel: rgb(39 39 42);
  --app-border: rgba(212, 212, 216, 0.18);
  --app-border-strong: rgba(212, 212, 216, 0.34);
  --app-fg: rgb(244 244 245);
  --app-fg-muted: rgb(212 212 216);
  --app-fg-soft: rgb(161 161 170);
}

* {
  box-sizing: border-box;
}

html {
  height: 100%;
  overflow: hidden;
  background: var(--app-bg);
}

body {
  margin: 0;
  height: 100%;
  overflow: hidden;
  background:
    radial-gradient(circle at top, var(--app-bg-glow), transparent 34%),
    linear-gradient(180deg, var(--app-bg) 0%, var(--app-bg-depth) 100%);
  color: var(--app-fg);
}
`
}
