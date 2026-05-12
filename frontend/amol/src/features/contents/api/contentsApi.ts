//frontend\amol\src\features\contents\api\contentsApi.ts
import { getAuth } from "firebase/auth";

import type { ContentsMetadata } from "../types";
import { parseContentsMetadata } from "../utils/metadata";

const BACKEND_BASE_URL = import.meta.env.VITE_API_BASE_URL;

function normalizeBackendUrl(backendUrl: string): string {
  return backendUrl.replace(/\/+$/, "");
}

export async function fetchContentsMetadata(
  metadataUri: string
): Promise<ContentsMetadata | null> {
  if (!BACKEND_BASE_URL) {
    throw new Error("VITE_API_BASE_URL is not configured.");
  }

  const auth = getAuth();
  const user = auth.currentUser;

  if (!user) {
    throw new Error("ログインが必要です。");
  }

  const idToken = await user.getIdToken();
  const baseUrl = normalizeBackendUrl(BACKEND_BASE_URL);
  const url = new URL(`${baseUrl}/mall/me/wallets/metadata/proxy`);

  url.searchParams.set("url", metadataUri);

  const response = await fetch(url.toString(), {
    method: "GET",
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${idToken}`,
    },
  });

  if (!response.ok) {
    const body = await response.text().catch(() => "");
    throw new Error(`metadata fetch failed: ${response.status} ${body}`);
  }

  const contentType = response.headers.get("content-type") || "";

  if (!contentType.includes("application/json")) {
    throw new Error("metadata API が JSON 以外を返しました。");
  }

  const body: unknown = await response.json();

  return parseContentsMetadata(body);
}