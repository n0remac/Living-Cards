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
  onControl: (control: ControlDescriptor, value: unknown) => void;
  onClose: () => void;
}

export function openComponentOverlay(options: ComponentOverlayOptions): void {
  const root = options.root;
  if (!root) return;
  closeComponentOverlay(root);

  const slots = edgeControlSlots(root);
  if (!slots) return;

  document.body.classList.add("stage-controls-open");
  renderHeader(slots.top, options.overlay, () => {
    closeComponentOverlay(root);
    options.onClose();
  });

  const leftControls: ControlDescriptor[] = [];
  const rightControls: ControlDescriptor[] = [];
  options.overlay.controls.forEach((control) => {
    if (control.kind === "range" || control.kind === "color") {
      rightControls.push(control);
      return;
    }
    leftControls.push(control);
  });

  renderControlRail(slots.left, leftControls, options.onControl);
  renderControlRail(slots.right, rightControls, options.onControl);
  renderStatus(slots.bottom, "Controls stay fixed while you edit " + options.overlay.title.toLowerCase() + ".");
}

export function closeComponentOverlay(root: HTMLElement | null): void {
  document.body.classList.remove("stage-controls-open");
  const slots = root ? edgeControlSlots(root) : null;
  slots?.all.forEach((slot) => {
    slot.innerHTML = "";
  });
  if (slots?.bottom) {
    renderStatus(slots.bottom, "");
  }
}

export function isComponentOverlayOpen(): boolean {
  return document.body.classList.contains("stage-controls-open");
}

function edgeControlSlots(root: HTMLElement): {
  top: HTMLElement;
  left: HTMLElement;
  right: HTMLElement;
  bottom: HTMLElement;
  all: HTMLElement[];
} | null {
  const top = root.querySelector<HTMLElement>("#stage-edge-controls-top");
  const left = root.querySelector<HTMLElement>("#stage-edge-controls-left");
  const right = root.querySelector<HTMLElement>("#stage-edge-controls-right");
  const bottom = root.querySelector<HTMLElement>("#stage-edge-controls-bottom");
  if (!top || !left || !right || !bottom) return null;
  return { top, left, right, bottom, all: [top, left, right, bottom] };
}

function renderHeader(root: HTMLElement, overlay: ComponentOverlay, onClose: () => void): void {
  root.innerHTML = "";
  stopCardTapEvents(root);

  const panel = document.createElement("div");
  panel.className = "stage-edge-panel stage-edge-header";

  const text = document.createElement("div");
  text.className = "min-w-0";

  const title = document.createElement("div");
  title.className = "stage-edge-title";
  title.textContent = overlay.title;

  const subtitle = document.createElement("div");
  subtitle.className = "stage-edge-subtitle";
  subtitle.textContent = overlay.componentKind + " controls";

  text.append(title, subtitle);

  const close = document.createElement("button");
  close.type = "button";
  close.className = "h-8 rounded-md border border-[var(--app-border)] bg-[var(--app-panel)] px-2 text-xs font-semibold text-[var(--app-fg-soft)]";
  close.textContent = "Close";
  close.addEventListener("click", onClose);

  panel.append(text, close);
  root.appendChild(panel);
}

function renderControlRail(root: HTMLElement, controls: ControlDescriptor[], onControl: (control: ControlDescriptor, value: unknown) => void): void {
  root.innerHTML = "";
  stopCardTapEvents(root);

  const panel = document.createElement("div");
  panel.className = "stage-edge-panel stage-edge-controls-group";
  if (!controls.length) {
    const empty = document.createElement("div");
    empty.className = "stage-edge-controls-empty";
    empty.textContent = "No controls here.";
    panel.appendChild(empty);
  } else {
    controls.forEach((control) => {
      panel.appendChild(renderControl(control, (value) => onControl(control, value)));
    });
  }
  root.appendChild(panel);
}

function renderStatus(root: HTMLElement, message: string, tone = "info"): void {
  root.innerHTML = "";
  stopCardTapEvents(root);

  const panel = document.createElement("div");
  panel.className = "stage-edge-panel";

  const status = document.createElement("div");
  status.id = "stage-edge-controls-status";
  status.className = "stage-edge-controls-status";
  status.dataset.tone = tone;
  status.textContent = message;

  panel.appendChild(status);
  root.appendChild(panel);
}

function renderControl(control: ControlDescriptor, onValue: (value: unknown) => void): HTMLElement {
  switch (control.kind) {
    case "checkbox":
      return renderCheckboxControl(control, onValue);
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

function renderCheckboxControl(control: ControlDescriptor, onValue: (value: unknown) => void): HTMLElement {
  const wrapper = document.createElement("label");
  wrapper.className = "flex items-center justify-between gap-3 rounded-md border border-[var(--app-border)] bg-[var(--app-panel)] px-2 py-2 text-sm font-semibold text-[var(--app-fg)]";

  const text = document.createElement("span");
  text.textContent = control.label;

  const input = document.createElement("input");
  input.type = "checkbox";
  input.className = "h-4 w-4 accent-emerald-300";
  input.checked = Boolean(control.value);
  input.addEventListener("change", () => onValue(input.checked));

  wrapper.append(text, input);
  return wrapper;
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

function hexOrFallback(value: string): string {
  const trimmed = String(value || "").trim();
  return /^#[0-9a-fA-F]{6}$/.test(trimmed) ? trimmed : "#22c55e";
}
