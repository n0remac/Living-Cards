import type { CardEvent } from "../types";

type NotificationTone = "info" | "error";

interface NotificationItem {
  message: string;
  tone: NotificationTone;
}

const visibleMS = 2600;
const notificationQueue: NotificationItem[] = [];
const notificationHistory: NotificationItem[] = [];
let notificationTimer = 0;
let activeNotification = false;

export function initNotifications(): void {
  const section = notificationSection();
  if (!section) return;
  stopCardTapEvents(section);
  const history = notificationHistoryPanel();
  if (history) {
    stopCardTapEvents(history);
  }
  section.addEventListener("click", (event) => {
    event.stopPropagation();
    toggleHistory();
  });
  document.addEventListener("click", (event) => {
    const history = notificationHistoryPanel();
    if (!history || history.classList.contains("hidden")) return;
    const target = event.target instanceof Node ? event.target : null;
    if (target && (history.contains(target) || section.contains(target))) return;
    history.classList.add("hidden");
  });
  renderHistory();
}

export function renderEvents(root: HTMLElement | null, events: CardEvent[]): void {
  if (!root) return;
  (events || []).forEach((event) => showEvent(root, event));
}

export function showMessage(root: HTMLElement | null, message: string, tone: NotificationTone = "info"): void {
  if (!root) return;
  enqueueNotification({ message, tone });
}

function showEvent(root: HTMLElement | null, event: CardEvent): void {
  switch (event.type) {
    case "configApplied":
      return;
    case "controlChanged":
      return;
    case "xpGained":
      return;
    case "levelUp":
      return;
    case "componentLevelUp":
      showMessage(root, labelForComponent(event.componentKind) + " level " + event.level);
      return;
    case "componentUnlocked":
      showMessage(root, event.message || labelForComponent(event.componentKind) + " unlocked");
      return;
    case "componentSelected":
      return;
    case "overlayOpened":
      return;
    case "configKindUnlocked":
      showMessage(root, labelForTarget(event.componentKind) + " unlocked");
      return;
    case "modeUnlocked":
      showMessage(root, labelForTarget(event.componentKind) + " " + event.mode + " unlocked");
      return;
    case "invalidAction":
      showMessage(root, event.message || "Locked", "error");
      return;
  }
}

function enqueueNotification(item: NotificationItem): void {
  notificationQueue.push(item);
  notificationHistory.unshift(item);
  if (notificationHistory.length > 30) {
    notificationHistory.length = 30;
  }
  renderHistory();
  showNextNotification();
}

function showNextNotification(): void {
  if (activeNotification) return;
  const current = notificationCurrent();
  const section = notificationSection();
  const edgeStatus = edgeControlStatus();
  if (!current || !section) return;
  const item = notificationQueue.shift();
  if (!item) {
    current.textContent = "No notifications";
    section.dataset.tone = "empty";
    if (edgeStatus) {
      edgeStatus.textContent = "Controls stay fixed while you edit.";
      edgeStatus.dataset.tone = "info";
    }
    return;
  }
  activeNotification = true;
  current.textContent = item.message;
  section.dataset.tone = item.tone;
  if (edgeStatus) {
    edgeStatus.textContent = item.message;
    edgeStatus.dataset.tone = item.tone;
  }
  window.clearTimeout(notificationTimer);
  notificationTimer = window.setTimeout(() => {
    activeNotification = false;
    showNextNotification();
  }, visibleMS);
}

function toggleHistory(): void {
  const history = notificationHistoryPanel();
  if (!history) return;
  history.classList.toggle("hidden");
  renderHistory();
}

function stopCardTapEvents(element: HTMLElement): void {
  for (const eventName of ["pointerdown", "pointermove", "pointerup", "pointercancel", "contextmenu"]) {
    element.addEventListener(eventName, (event) => {
      event.stopPropagation();
    });
  }
}

function renderHistory(): void {
  const list = notificationHistoryList();
  if (!list) return;
  list.innerHTML = "";
  if (!notificationHistory.length) {
    const empty = document.createElement("div");
    empty.className = "stage-notification-history-empty";
    empty.textContent = "No notifications yet.";
    list.appendChild(empty);
    return;
  }
  notificationHistory.forEach((item) => {
    const row = document.createElement("div");
    row.className = "stage-notification-history-item";
    row.dataset.tone = item.tone;
    row.textContent = item.message;
    list.appendChild(row);
  });
}

function notificationSection(): HTMLButtonElement | null {
  return document.getElementById("stage-notification-section") as HTMLButtonElement | null;
}

function notificationCurrent(): HTMLSpanElement | null {
  return document.getElementById("stage-notification-current") as HTMLSpanElement | null;
}

function notificationHistoryPanel(): HTMLDivElement | null {
  return document.getElementById("stage-notification-history") as HTMLDivElement | null;
}

function notificationHistoryList(): HTMLDivElement | null {
  return document.getElementById("stage-notification-history-list") as HTMLDivElement | null;
}

function edgeControlStatus(): HTMLDivElement | null {
  return document.getElementById("stage-edge-controls-status") as HTMLDivElement | null;
}

function labelForTarget(componentKind: string): string {
  switch (componentKind) {
    case "background":
      return "Background";
    case "border":
      return "Border";
    case "textarea":
      return "Text";
    default:
      return "Card";
  }
}

function labelForComponent(componentKind: string): string {
  switch (componentKind) {
    case "textarea":
      return "Text";
    case "shape":
      return "Shape";
    default:
      return "Card";
  }
}
