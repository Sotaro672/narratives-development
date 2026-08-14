// frontend/amol/src/features/contents/api/contentsApi.ts

import { requestJson } from "../../../lib/http";
import type { ContentsMetadata } from "../../shared/types/contents";

const METADATA_PROXY_PATH = "/mall/me/wallets/metadata/proxy";

type MetadataProxyResponse = {
  name: string;
  image: string;
  properties: {
    files: Array<{
      uri: string;
      type: string;
    }>;
  };
};

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
    files: body.properties.files,
  };
}