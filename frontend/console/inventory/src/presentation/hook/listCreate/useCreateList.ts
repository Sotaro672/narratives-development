// frontend/console/inventory/src/presentation/hook/listCreate/useCreateList.ts
import * as React from "react";
import type { useNavigate } from "react-router-dom";

import {
  createListWithImages,
  buildAfterCreatePath,
  type ResolvedListCreateParams,
  type PriceRow, // ✅ PriceRowEx 廃止（存在しない）→ PriceRow を使用
} from "../../../application/listCreate/listCreateService";

import type { ListingDecision } from "./types";

export function useCreateList(args: {
  navigate: ReturnType<typeof useNavigate>;
  resolvedParams: ResolvedListCreateParams;
  decision: ListingDecision;
  listingTitle: string;
  description: string;
  priceRows: PriceRow[]; // ✅ PriceRowEx → PriceRow
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
          alert(msg);
          throw new Error(msg);
        }

        // inventoryId は docId を正としてそのまま扱う（split しない）
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

        alert("作成しました");
        navigate(buildAfterCreatePath(resolvedParams));
      } catch (e) {
        const msg = String(e instanceof Error ? e.message : e);
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
