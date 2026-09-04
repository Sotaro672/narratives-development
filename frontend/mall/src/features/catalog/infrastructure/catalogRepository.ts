// frontend/amol/src/features/catalog/infrastructure/catalogRepository.ts

import type { CatalogResponse } from "../../shared/types/catalog";

export async function fetchCatalogDetail(args: {
  apiBaseUrl: string;
  listId: string;
}): Promise<CatalogResponse> {
  const { apiBaseUrl, listId } = args;

  const response = await fetch(`${apiBaseUrl}/mall/catalog/${encodeURIComponent(listId)}`, {
    method: "GET",
    headers: {
      Accept: "application/json",
    },
    credentials: "include",
  });

  const contentType = response.headers.get("content-type") ?? "";

  if (!contentType.includes("application/json")) {
    throw new Error("カタログ詳細APIがJSON以外を返しました。");
  }

  const data = (await response.json()) as CatalogResponse;

  if (!response.ok) {
    throw new Error("カタログ詳細の取得に失敗しました。");
  }

  return data;
}