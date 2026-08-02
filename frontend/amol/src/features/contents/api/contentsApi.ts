// frontend/amol/src/features/contents/api/contentsApi.ts

import {
  buildApiUrl,
  getApiBaseUrl,
} from "../../../lib/apiBaseUrl";
import { getFirebaseIdToken } from "../../../lib/authToken";

import type { ContentsMetadata } from "../../shared/types/contents";
import { parseContentsMetadata } from "../utils/metadata";

const METADATA_PROXY_PATH =
  "/mall/me/wallets/metadata/proxy";

function assertApiBaseUrl(): string {
  const baseUrl = getApiBaseUrl();

  if (!baseUrl) {
    throw new Error(
      "VITE_API_BASE_URL is not configured.",
    );
  }

  return baseUrl;
}

export async function fetchContentsMetadata(
  metadataUri: string,
): Promise<ContentsMetadata | null> {
  const baseUrl = assertApiBaseUrl();
  const idToken = await getFirebaseIdToken();

  const url = new URL(
    buildApiUrl(
      baseUrl,
      METADATA_PROXY_PATH,
    ),
  );

  url.searchParams.set("url", metadataUri);

  const response = await fetch(
    url.toString(),
    {
      method: "GET",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${idToken}`,
      },
    },
  );

  if (!response.ok) {
    const body = await response
      .text()
      .catch(() => "");

    throw new Error(
      `metadata fetch failed: ${response.status} ${body}`,
    );
  }

  const contentType =
    response.headers.get("content-type") || "";

  if (
    !contentType.includes(
      "application/json",
    )
  ) {
    throw new Error(
      "metadata API が JSON 以外を返しました。",
    );
  }

  const body: unknown =
    await response.json();

  return parseContentsMetadata(body);
}