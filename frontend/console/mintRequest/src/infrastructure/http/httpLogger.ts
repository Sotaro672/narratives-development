// frontend/console/mintRequest/src/infrastructure/http/httpLogger.ts

// ---------------------------------------------------------
// ✅ DEBUG: HTTP ログ（渡したリクエストが分かる）
// - DEV では常に出す
// - 本番で出したい場合は VITE_DEBUG_HTTP=1
// ---------------------------------------------------------
const DEBUG_HTTP =
  Boolean((import.meta as any).env?.DEV) ||
  String((import.meta as any).env?.VITE_DEBUG_HTTP ?? "") === "1";

export function safeTokenHint(idToken: string): string {
  const t = String(idToken ?? "");
  if (!t) return "(empty)";
  return `${t.slice(0, 10)}...(${t.length})`;
}

export function logHttpRequest(tag: string, info: any) {
  if (!DEBUG_HTTP) return;
  // eslint-disable-next-line no-console
  console.log(`[mintRequest/mintRequestRepositoryHTTP] ${tag} request`, info);
}

export function logHttpResponse(tag: string, info: any) {
  if (!DEBUG_HTTP) return;
  // eslint-disable-next-line no-console
  console.log(`[mintRequest/mintRequestRepositoryHTTP] ${tag} response`, info);
}

export function logHttpError(tag: string, info: any) {
  if (!DEBUG_HTTP) return;
  // eslint-disable-next-line no-console
  console.log(`[mintRequest/mintRequestRepositoryHTTP] ${tag} error`, info);
}
