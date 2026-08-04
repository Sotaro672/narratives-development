// frontend/amol/src/features/contents/api/contentsApi.ts

import {
  requestJson,
} from "../../../lib/http";

import type { ContentsMetadata } from "../../shared/types/contents";
import { parseContentsMetadata } from "../utils/metadata";

const METADATA_PROXY_PATH =
  "/mall/me/wallets/metadata/proxy";

export async function fetchContentsMetadata(
  metadataUri: string,
): Promise<ContentsMetadata | null> {
  const body =
    await requestJson<unknown>(
      METADATA_PROXY_PATH,
      {
        method: "GET",
        auth: "required",
        query: {
          url: metadataUri,
        },
        messages: {
          requestErrorMessage:
            "metadata fetch failed.",
          nonJsonErrorMessage:
            "metadata API が JSON 以外を返しました。",
        },
      },
    );

  return parseContentsMetadata(body);
}