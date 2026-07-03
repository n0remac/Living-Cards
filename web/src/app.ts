import { initDesigner, resetDraft } from "./designer/controller";
import { initStage } from "./stage/StageController";

document.addEventListener("DOMContentLoaded", () => {
  initDesigner();
  initStage({ resetDraft });
});
