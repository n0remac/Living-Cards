import type { ComponentOverlay, ControlDescriptor } from "../types";

const swatches = [
  "#22c55e",
  "#38bdf8",
  "#f59e0b",
  "#f43f5e",
  "#a78bfa",
  "#f8fafc",
  "#111827",
  "#f5e6c8",
];

interface ComponentOverlayOptions {
  root: HTMLElement | null;
  overlay: ComponentOverlay;
  anchorX: number;
  anchorY: number;
  onControl: (control: ControlDescriptor, value: unknown) => void;
  onRandomize: () => void;
}

export function openComponentOverlay(options: ComponentOverlayOptions): void {
  const root = options.root;
  if (!root) return;
  closeComponentOverlay(root);

  const panel = document.createElement("div");
  panel.dataset.stageControlOverlay = "component";
  panel.className = "stage-component-panel pointer-events-auto fixed grid max-h-[min(28rem,calc(100dvh-6rem))] w-72 gap-3 overflow-y-auto rounded-md border border-[var(--app-border-strong)] bg-[var(--app-surface-muted)] p-3 shadow-2xl backdrop-blur";
  panel.style.left = clampPX(options.anchorX, 12, window.innerWidth - 300) + "px";
  panel.style.top = clampPX(options.anchorY, 84, window.innerHeight - 360) + "px";
  stopCardTapEvents(panel);

  const header = document.createElement("div");
  header.className = "flex items-center justify-between gap-3";

  const title = document.createElement("div");
  title.className = "min-w-0 text-sm font-semibold text-[var(--app-fg)]";
  title.textContent = options.overlay.title;

  const close = document.createElement("button");
  close.type = "button";
  close.className = "h-8 rounded-md border border-[var(--app-border)] bg-[var(--app-panel)] px-2 text-xs font-semibold text-[var(--app-fg-soft)]";
  close.textContent = "Close";
  close.addEventListener("click", () => panel.remove());

  header.append(title, close);
  panel.appendChild(header);

  if (options.overlay.randomizeEnabled) {
    const randomize = document.createElement("button");
    randomize.type = "button";
    randomize.className = "h-9 rounded-md border border-emerald-300/30 bg-emerald-300 px-3 text-sm font-semibold text-zinc-950";
    randomize.textContent = "Randomize";
    randomize.addEventListener("click", options.onRandomize);
    panel.appendChild(randomize);
  }

  const controls = document.createElement("div");
  controls.className = "grid gap-3";
  options.overlay.controls.forEach((control) => {
    controls.appendChild(renderControl(control, (value) => options.onControl(control, value)));
  });
  panel.appendChild(controls);
  root.appendChild(panel);
}

export function closeComponentOverlay(root: HTMLElement | null): void {
  root?.querySelectorAll("[data-stage-control-overlay]").forEach((element) => element.remove());
}

function renderControl(control: ControlDescriptor, onValue: (value: unknown) => void): HTMLElement {
  switch (control.kind) {
    case "color":
      return renderColorControl(control, onValue);
    case "range":
      return renderRangeControl(control, onValue);
    case "select":
      return renderSelectControl(control, onValue);
    case "text":
      return renderTextControl(control, onValue);
    default:
      return document.createElement("div");
  }
}

function renderColorControl(control: ControlDescriptor, onValue: (value: unknown) => void): HTMLElement {
  const wrapper = controlWrapper(control.label);
  const current = hexOrFallback(String(control.value || "#22c55e"));
  const row = document.createElement("div");
  row.className = "grid grid-cols-[1fr_auto] gap-2";

  const input = document.createElement("input");
  input.type = "color";
  input.value = current;
  input.className = "h-9 w-full cursor-pointer rounded-md border border-[var(--app-border)] bg-[var(--app-panel)]";
  input.addEventListener("change", () => onValue(input.value));

  const preview = document.createElement("span");
  preview.className = "block h-9 w-9 rounded-md border border-white/25";
  preview.style.background = current;
  input.addEventListener("input", () => {
    preview.style.background = input.value;
  });

  row.append(input, preview);

  const grid = document.createElement("div");
  grid.className = "grid grid-cols-8 gap-1";
  swatches.forEach((color) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "h-6 rounded-sm border border-white/25";
    button.title = color;
    button.style.background = color;
    button.addEventListener("click", () => {
      input.value = color;
      preview.style.background = color;
      onValue(color);
    });
    grid.appendChild(button);
  });

  wrapper.append(row, grid);
  return wrapper;
}

function renderRangeControl(control: ControlDescriptor, onValue: (value: unknown) => void): HTMLElement {
  const wrapper = controlWrapper(control.label);
  const value = document.createElement("span");
  value.className = "text-xs font-semibold text-[var(--app-fg-soft)]";
  value.textContent = String(control.value ?? control.min ?? 0);

  const input = document.createElement("input");
  input.type = "range";
  input.min = String(control.min ?? 0);
  input.max = String(control.max ?? 100);
  input.step = String(control.step ?? 1);
  input.value = String(control.value ?? control.min ?? 0);
  input.className = "w-full accent-emerald-300";
  input.addEventListener("input", () => {
    value.textContent = input.value;
  });
  input.addEventListener("change", () => onValue(Number(input.value)));

  wrapper.firstElementChild?.appendChild(value);
  wrapper.appendChild(input);
  return wrapper;
}

function renderSelectControl(control: ControlDescriptor, onValue: (value: unknown) => void): HTMLElement {
  const wrapper = controlWrapper(control.label);
  const select = document.createElement("select");
  select.className = "h-9 rounded-md border border-[var(--app-border)] bg-[var(--app-panel)] px-2 text-sm text-[var(--app-fg)] outline-none";
  (control.options || []).forEach((option) => {
    const item = document.createElement("option");
    item.value = option.value;
    item.textContent = option.label;
    select.appendChild(item);
  });
  select.value = String(control.value ?? "");
  select.addEventListener("change", () => onValue(select.value));
  wrapper.appendChild(select);
  return wrapper;
}

function renderTextControl(control: ControlDescriptor, onValue: (value: unknown) => void): HTMLElement {
  const wrapper = controlWrapper(control.label);
  const input = document.createElement("input");
  input.type = "text";
  input.value = String(control.value ?? "");
  input.className = "h-9 rounded-md border border-[var(--app-border)] bg-[var(--app-panel)] px-2 text-sm text-[var(--app-fg)] outline-none";
  input.addEventListener("change", () => onValue(input.value));
  input.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      event.preventDefault();
      input.blur();
      onValue(input.value);
    }
  });
  wrapper.appendChild(input);
  return wrapper;
}

function controlWrapper(label: string): HTMLElement {
  const wrapper = document.createElement("label");
  wrapper.className = "grid gap-1 text-xs font-semibold uppercase text-[var(--app-fg-soft)]";
  const text = document.createElement("span");
  text.textContent = label;
  wrapper.appendChild(text);
  return wrapper;
}

function stopCardTapEvents(element: HTMLElement): void {
  for (const eventName of ["pointerdown", "pointermove", "pointerup", "pointercancel", "contextmenu"]) {
    element.addEventListener(eventName, (event) => {
      event.stopPropagation();
    });
  }
}

function clampPX(value: number, min: number, max: number): number {
  if (!Number.isFinite(value)) return min;
  return Math.max(min, Math.min(max, value));
}

function hexOrFallback(value: string): string {
  const trimmed = String(value || "").trim();
  return /^#[0-9a-fA-F]{6}$/.test(trimmed) ? trimmed : "#22c55e";
}
