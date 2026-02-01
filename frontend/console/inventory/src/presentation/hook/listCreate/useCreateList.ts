// frontend/console/inventory/src/presentation/hook/listCreate/useCreateList.ts
import * as React from "react";
import type { useNavigate } from "react-router-dom";

import {
  createListWithImages,
  buildAfterCreatePath,
  type ResolvedListCreateParams,
  type PriceRowEx,
} from "../../../application/listCreate/listCreateService";

import type { ListingDecision } from "./types";

export function useCreateList(args: {
  navigate: ReturnType<typeof useNavigate>;
  resolvedParams: ResolvedListCreateParams;
  decision: ListingDecision;
  listingTitle: string;
  description: string;
  priceRows: PriceRowEx[];
  assigneeId: string | undefined;
  images: File[];
  mainImageIndex: number;
}): { onCreate: () => void } {
  const {
    navigate,
    resolvedParams,
    decision,
    listingTitle,
    description,
    priceRows,
    assigneeId,
    images,
    mainImageIndex,
  } = args;

  const onCreate = React.useCallback(() => {
    void (async () => {
      try {
        if (images.length === 0) {
          const msg = "商品画像は1枚以上必須です。画像を追加してください。";

          // eslint-disable-next-line no-console
          console.log("[inventory/listImage] validation failed (no images)", {
            imagesCount: images.length,
            mainImageIndex,
          });

          alert(msg);
          throw new Error(msg);
        }

        // eslint-disable-next-line no-console
        console.log("[inventory/listImage] create start", {
          imagesCount: images.length,
          mainImageIndex,
          names: images.slice(0, 8).map((f) => f.name),
          sizes: images.slice(0, 8).map((f) => f.size),
          types: images.slice(0, 8).map((f) => f.type),
        });

        // inventoryId は "pbId__tbId" を含むため split("__") しない
        const rawInventoryId = String(resolvedParams.inventoryId ?? "");
        const safeInventoryId = rawInventoryId;

        await createListWithImages({
          params: { ...resolvedParams, inventoryId: safeInventoryId },
          listingTitle,
          description,
          priceRows: priceRows as any,
          decision,
          assigneeId,
          images,
          mainImageIndex,
        });

        // eslint-disable-next-line no-console
        console.log("[inventory/listImage] create success", {
          imagesCount: images.length,
          mainImageIndex,
        });

        alert("作成しました");
        navigate(buildAfterCreatePath(resolvedParams));
      } catch (e) {
        const msg = String(e instanceof Error ? e.message : e);

        // eslint-disable-next-line no-console
        console.log("[inventory/listImage] create failed", {
          msg,
          imagesCount: images.length,
          mainImageIndex,
        });

        alert(msg);
      }
    })();
  }, [
    assigneeId,
    decision,
    description,
    images,
    listingTitle,
    mainImageIndex,
    navigate,
    priceRows,
    resolvedParams,
  ]);

  return { onCreate };
}
