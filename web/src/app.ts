import { initDesigner, resetDraft } from "./designer/controller";
import { initAppHeader } from "../../internal/web/components/appheader/client";

document.addEventListener("DOMContentLoaded", () => {
  initAppHeader({ resetDraft });
  initDesigner();
});
