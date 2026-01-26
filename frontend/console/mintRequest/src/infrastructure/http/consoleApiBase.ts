// frontend/console/mintRequest/src/infrastructure/http/consoleApiBase.ts

// ✅ Console API base resolver（修正案A）
// - VITE_BACKEND_BASE_URL は origin のみ想定
// - /console は apiBase 側で付与（env に /console が混入しても正規化して除去）
import { API_BASE as CONSOLE_API_BASE } from "../../../../shell/src/shared/http/apiBase";

// ✅ Console API base（/console 付き）
export const API_BASE = CONSOLE_API_BASE;
