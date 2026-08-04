//frontend\amol\src\features\market\infrastructure\marketHttpClient.ts
import {
  getApiBaseUrl,
} from "../../../lib/apiBaseUrl";

export function getMarketApiBaseUrl(): string {
  const baseUrl =
    getApiBaseUrl().trim();

  if (!baseUrl) {
    throw new Error(
      "API Base URLが未設定です。",
    );
  }

  return baseUrl.replace(/\/+$/, "");
}

export async function readMarketJsonResponse<T>(
  response: Response,
  fallbackMessage: string,
): Promise<T> {
  const contentType =
    response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    throw new Error(fallbackMessage);
  }

  const data =
    (await response.json()) as T;

  if (!response.ok) {
    throw new Error(
      getMarketErrorMessage(response.status),
    );
  }

  return data;
}

function getMarketErrorMessage(
  status: number,
): string {
  if (status === 400) {
    return "マーケット一覧の取得条件が不正です。";
  }

  if (status === 401) {
    return "ログインが必要です。";
  }

  if (status === 403) {
    return "マーケット情報を取得する権限がありません。";
  }

  if (status === 404) {
    return "マーケット情報が見つかりません。";
  }

  if (status >= 500) {
    return "サーバー側でエラーが発生しました。";
  }

  return "マーケット情報の取得に失敗しました。";
}