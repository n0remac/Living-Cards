import type { ComponentTarget } from "../types";

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

interface ColorControlsOptions {
  root: HTMLElement | null;
  target: ComponentTarget;
  anchorX: number;
  anchorY: number;
  currentColor: string;
  onColor: (control: ColorControlSelection) => void;
}

export interface ColorControlSelection {
  color: string;
  secondaryColor?: string;
  gradient?: boolean;
  angle?: number;
}

export function openColorControls(options: ColorControlsOptions): void {
  const root = options.root;
  if (!root) return;
  closeColorControls(root);

  const panel = document.createElement("div");
  panel.dataset.stageControlOverlay = "color";
  panel.className = "stage-control-panel pointer-events-auto fixed rounded-md border border-[var(--app-border-strong)] bg-[var(--app-surface-muted)] p-2 shadow-2xl backdrop-blur";
  panel.style.left = clampPX(options.anchorX, 12, window.innerWidth - 260) + "px";
  panel.style.top = clampPX(options.anchorY, 88, window.innerHeight - 270) + "px";

  const header = document.createElement("div");
  header.className = "flex items-center justify-between gap-2";

  const palette = document.createElement("button");
  palette.type = "button";
  palette.className = "flex h-9 items-center gap-2 rounded-md border border-[var(--app-border)] bg-[var(--app-panel)] px-2 text-sm font-semibold text-[var(--app-fg)]";
  palette.title = labelForTarget(options.target) + " color";
  palette.innerHTML = '<span class="block h-5 w-5 rounded-sm border border-white/35" style="background:' + escapeAttribute(hexOrFallback(options.currentColor)) + '"></span><span>Palette</span>';

  const close = document.createElement("button");
  close.type = "button";
  close.className = "h-9 rounded-md border border-[var(--app-border)] bg-[var(--app-panel)] px-2 text-sm text-[var(--app-fg-soft)]";
  close.textContent = "Close";
  close.addEventListener("click", () => panel.remove());

  const chooser = document.createElement("div");
  chooser.className = "mt-2 hidden grid w-56 gap-3";

  const primaryColor = hexOrFallback(options.currentColor);
  let secondaryColor = defaultSecondary(primaryColor);
  let angle = 135;
  let gradientEnabled = false;

  const swatchGrid = document.createElement("div");
  swatchGrid.className = "grid grid-cols-4 gap-1.5";
  swatches.forEach((color) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "h-8 rounded-md border border-white/25 shadow-sm";
    button.title = color;
    button.style.background = color;
    button.addEventListener("click", () => {
      if (gradientEnabled) {
        secondaryColor = color;
        secondaryInput.value = color;
        preview.style.background = gradientCSS(input.value, secondaryColor, angle);
        return;
      }
      options.onColor({ color });
    });
    swatchGrid.appendChild(button);
  });

  const input = document.createElement("input");
  input.type = "color";
  input.className = "h-9 w-full cursor-pointer rounded-md border border-[var(--app-border)] bg-[var(--app-panel)]";
  input.value = primaryColor;
  input.addEventListener("change", () => {
    preview.style.background = gradientEnabled ? gradientCSS(input.value, secondaryColor, angle) : input.value;
    if (!gradientEnabled) {
      options.onColor({ color: input.value });
    }
  });

  const gradientRow = document.createElement("label");
  gradientRow.className = "flex items-center justify-between gap-3 rounded-md border border-[var(--app-border)] bg-[var(--app-panel)] px-2 py-2 text-sm text-[var(--app-fg)]";
  gradientRow.innerHTML = "<span>Gradient</span>";

  const gradientToggle = document.createElement("input");
  gradientToggle.type = "checkbox";
  gradientToggle.className = "h-4 w-4 accent-emerald-300";

  const gradientControls = document.createElement("div");
  gradientControls.className = "hidden grid gap-2";

  const secondaryInput = document.createElement("input");
  secondaryInput.type = "color";
  secondaryInput.className = "h-9 w-full cursor-pointer rounded-md border border-[var(--app-border)] bg-[var(--app-panel)]";
  secondaryInput.value = secondaryColor;
  secondaryInput.addEventListener("input", () => {
    secondaryColor = secondaryInput.value;
    preview.style.background = gradientCSS(input.value, secondaryColor, angle);
  });

  const angleInput = document.createElement("input");
  angleInput.type = "range";
  angleInput.min = "0";
  angleInput.max = "315";
  angleInput.step = "45";
  angleInput.value = String(angle);
  angleInput.className = "w-full accent-emerald-300";
  angleInput.addEventListener("input", () => {
    angle = Number(angleInput.value) || 135;
    angleValue.textContent = angle + "deg";
    preview.style.background = gradientCSS(input.value, secondaryColor, angle);
  });

  const angleValue = document.createElement("span");
  angleValue.className = "text-xs font-semibold text-[var(--app-fg-soft)]";
  angleValue.textContent = angle + "deg";

  const preview = document.createElement("div");
  preview.className = "h-8 rounded-md border border-white/20";
  preview.style.background = primaryColor;

  const applyGradient = document.createElement("button");
  applyGradient.type = "button";
  applyGradient.className = "h-9 rounded-md border border-emerald-300/30 bg-emerald-300 px-3 text-sm font-semibold text-zinc-950";
  applyGradient.textContent = "Apply Gradient";
  applyGradient.addEventListener("click", () => {
    options.onColor({
      color: input.value,
      secondaryColor,
      gradient: true,
      angle,
    });
  });

  gradientToggle.addEventListener("change", () => {
    gradientEnabled = gradientToggle.checked;
    gradientControls.classList.toggle("hidden", !gradientEnabled);
    preview.style.background = gradientEnabled ? gradientCSS(input.value, secondaryColor, angle) : input.value;
  });

  palette.addEventListener("click", () => {
    chooser.classList.toggle("hidden");
  });

  gradientRow.appendChild(gradientToggle);
  gradientControls.append(
    labeledControl("Second Color", secondaryInput),
    labeledControl("Angle", angleInput, angleValue),
    preview,
    applyGradient,
  );
  header.append(palette, close);
  chooser.append(swatchGrid, labeledControl("Color", input), gradientRow, gradientControls);
  panel.append(header, chooser);
  root.appendChild(panel);
}

export function closeColorControls(root: HTMLElement | null): void {
  root?.querySelectorAll("[data-stage-control-overlay]").forEach((element) => element.remove());
}

function clampPX(value: number, min: number, max: number): number {
  if (!Number.isFinite(value)) return min;
  return Math.max(min, Math.min(max, value));
}

function hexOrFallback(value: string): string {
  const trimmed = String(value || "").trim();
  return /^#[0-9a-fA-F]{6}$/.test(trimmed) ? trimmed : "#22c55e";
}

function defaultSecondary(primary: string): string {
  return primary.toLowerCase() === "#38bdf8" ? "#22c55e" : "#38bdf8";
}

function gradientCSS(primary: string, secondary: string, angle: number): string {
  return "linear-gradient(" + angle + "deg, " + primary + " 0%, " + secondary + " 100%)";
}

function labeledControl(label: string, control: HTMLElement, aside?: HTMLElement): HTMLElement {
  const wrapper = document.createElement("label");
  wrapper.className = "grid gap-1 text-xs font-semibold uppercase text-[var(--app-fg-soft)]";
  const row = document.createElement("span");
  row.className = "flex items-center justify-between gap-2";
  const text = document.createElement("span");
  text.textContent = label;
  row.appendChild(text);
  if (aside) row.appendChild(aside);
  wrapper.append(row, control);
  return wrapper;
}

function labelForTarget(target: ComponentTarget): string {
  switch (target) {
    case "background":
      return "Background";
    case "border":
      return "Border";
    default:
      return "Card";
  }
}

function escapeAttribute(value: string): string {
  return value.replace(/&/g, "&amp;").replace(/"/g, "&quot;");
}
