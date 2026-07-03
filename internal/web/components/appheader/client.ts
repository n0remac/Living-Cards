import { byID } from "../../../../web/src/dom";

interface AppHeaderDeps {
  resetDraft: () => Promise<void>;
}

export function initAppHeader(deps: AppHeaderDeps): void {
  const reset = byID<HTMLButtonElement>("reset-draft-btn");
  if (reset) {
    reset.addEventListener("click", () => {
      void deps.resetDraft();
    });
  }
}
