// frontend/console/shell/src/shared/http/apiBase.ts

/**
 * Console共通のBackend API URLを解決する。
 *
 * Policy:
 * - VITE_BACKEND_BASE_URLはoriginのみを想定する
 * - Console APIは現在、/consoleプレフィックスなしで提供される
 * - 環境変数に/console、/mall、/snsなどが含まれていても除去する
 */

/** Cloud Run fallback origin. */
export const FALLBACK_BACKEND_ORIGIN =
  "https://narratives-backend-871263659099.asia-northeast1.run.app";

/**
 * Backend URLをorigin形式へ正規化する。
 *
 * - 前後の空白を削除
 * - 末尾のスラッシュを削除
 * - 誤って付与された既知のAPIパスを削除
 */
function normalizeBackendOrigin(
  input: string,
): string {
  return input
    .trim()
    .replace(/\/+$/g, "")
    .replace(/\/(console|mall|sns)(\/.*)?$/i, "");
}

/**
 * Vite環境変数からBackend originを取得する。
 *
 * 有効な値を取得できない場合はCloud Runの
 * fallback originを使用する。
 */
function resolveBackendOrigin(): string {
  const envValue =
    (import.meta as {
      env?: {
        VITE_BACKEND_BASE_URL?: string;
      };
    }).env?.VITE_BACKEND_BASE_URL ?? "";

  return (
    normalizeBackendOrigin(envValue) ||
    FALLBACK_BACKEND_ORIGIN
  );
}

/**
 * Backend originとAPI pathを結合する。
 */
function joinUrl(
  base: string,
  path: string,
): string {
  const normalizedPath =
    path.replace(/^\/+/g, "");

  if (!normalizedPath) {
    return base;
  }

  return `${base}/${normalizedPath}`;
}

/**
 * Console APIのBackend origin。
 *
 * 環境変数は実行中に変化しないため、
 * モジュール読み込み時に一度だけ解決する。
 */
export const API_BASE =
  resolveBackendOrigin();

/**
 * Console APIの完全なURLを生成する。
 */
export function buildConsoleUrl(
  path: string,
): string {
  return joinUrl(
    API_BASE,
    path,
  );
}