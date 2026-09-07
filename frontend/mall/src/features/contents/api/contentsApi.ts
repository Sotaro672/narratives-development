// frontend/mall/src/features/contents/api/contentsApi.ts

import { requestJson } from "../../../lib/http";
import type { ContentsMetadata } from "../../shared/types/contents";

const METADATA_PROXY_PATH = "/mall/me/wallets/metadata/proxy";

export type TokenBlueprintModerationStatus =
  | "ACTIVE"
  | "HIDDEN_BY_MODERATION";

export type TokenBlueprintModerationStatusResponse = {
  tokenBlueprintId: string;
  status: TokenBlueprintModerationStatus;
};

type MetadataProxyResponse = {
  name: string;
  image: string;
  description: string;
  properties: {
    files: Array<{
      uri: string;
      type: string;
    }>;
  };
};

export async function fetchTokenBlueprintModerationStatus(
  tokenBlueprintId: string,
): Promise<TokenBlueprintModerationStatusResponse> {
  const normalizedTokenBlueprintId = tokenBlueprintId.trim();

  if (!normalizedTokenBlueprintId) {
    throw new Error("tokenBlueprintId is required.");
  }

  return requestJson<TokenBlueprintModerationStatusResponse>(
    `/mall/me/token-blueprints/${encodeURIComponent(
      normalizedTokenBlueprintId,
    )}/moderation-status`,
    {
      method: "GET",
      auth: "required",
      messages: {
        requestErrorMessage:
          "token blueprint moderation status fetch failed.",
        nonJsonErrorMessage:
          "TokenBlueprint moderation API が JSON 以外を返しました。",
      },
    },
  );
}

export async function fetchContentsMetadata(
  metadataUri: string,
): Promise<ContentsMetadata> {
  const body = await requestJson<MetadataProxyResponse>(
    METADATA_PROXY_PATH,
    {
      method: "GET",
      auth: "required",
      query: {
        url: metadataUri,
      },
      messages: {
        requestErrorMessage: "metadata fetch failed.",
        nonJsonErrorMessage: "metadata API が JSON 以外を返しました。",
      },
    },
  );

  return {
    name: body.name,
    image: body.image,
    description: body.description,
    files: body.properties.files,
  };
}