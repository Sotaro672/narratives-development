// frontend/console/inventory/src/presentation/hook/listCreate/useListCreateDTO.ts

import * as React from "react";
import type { useNavigate } from "react-router-dom";

import {
  canFetchListCreate,
  getInventoryIdFromDTO,
  shouldRedirectToInventoryIdRoute,
  buildInventoryListCreatePath,
  extractDisplayStrings,
} from "../../../application/listCreate/listCreateService";

import { loadListCreateDTOFromParams } from "../../../application/listCreate/listCreate.usecase";

import type {
  PriceRow,
  ResolvedListCreateParams,
} from "../../../application/listCreate/listCreate.types";

import type { ListCreateDTO } from "../../../infrastructure/http/listCreateRepositoryHTTP.types";

function initPriceRowsFromDTOKeepingModelFields(dto: ListCreateDTO): PriceRow[] {
  const rows = Array.isArray(dto.priceRows) ? dto.priceRows : [];

  return rows.map((row) => ({
    modelId: row.modelId,

    kind: row.kind ?? null,

    displayOrder:
      row.displayOrder === undefined || row.displayOrder === null
        ? null
        : row.displayOrder,

    // apparel category 用
    size: row.size ?? null,
    color: row.color ?? null,
    rgb: row.rgb ?? null,

    // alcohol category 用
    volumeValue: row.volumeValue ?? null,
    volumeUnit: row.volumeUnit ?? null,

    stock: Number.isFinite(Number(row.stock)) ? Number(row.stock) : 0,

    price:
      row.price === undefined || row.price === null
        ? row.price
        : Number.isFinite(Number(row.price))
          ? Number(row.price)
          : null,
  }));
}

export function useListCreateDTO(args: {
  navigate: ReturnType<typeof useNavigate>;
  inventoryId: string | undefined;
  resolvedParams: ResolvedListCreateParams;
  initializedPriceRowsRef: React.MutableRefObject<boolean>;
  setPriceRows: React.Dispatch<React.SetStateAction<PriceRow[]>>;
}): {
  dto: ListCreateDTO | null;
  loadingDTO: boolean;
  dtoError: string;
  productBrandName: string;
  productName: string;
  tokenBrandName: string;
  tokenName: string;
} {
  const {
    navigate,
    inventoryId,
    resolvedParams,
    initializedPriceRowsRef,
    setPriceRows,
  } = args;

  const [dto, setDTO] = React.useState<ListCreateDTO | null>(null);
  const [loadingDTO, setLoadingDTO] = React.useState(false);
  const [dtoError, setDTOError] = React.useState<string>("");

  const redirectedRef = React.useRef(false);

  React.useEffect(() => {
    let cancelled = false;

    const run = async () => {
      const canFetch = canFetchListCreate(resolvedParams);
      if (!canFetch) return;

      setLoadingDTO(true);
      setDTOError("");

      try {
        /**
         * ここでは必ず usecase を通す。
         *
         * loadListCreateDTOFromParams() 内で:
         * - /inventory/list-create/:inventoryId を取得
         * - /models/by-blueprint/:productBlueprintId/variations を取得
         * - priceRows に kind / volumeValue / volumeUnit を合成
         *
         * まで行うため、raw API や旧 initPriceRowsFromDTO を使うと
         * alcohol 用 field が PriceCard に届かない。
         */
        const data = await loadListCreateDTOFromParams(resolvedParams);
        if (cancelled) return;

        const gotInventoryId = getInventoryIdFromDTO(data);
        const currentInventoryId = String(inventoryId ?? "");

        if (
          shouldRedirectToInventoryIdRoute({
            currentInventoryId,
            gotInventoryId,
            alreadyRedirected: redirectedRef.current,
          })
        ) {
          redirectedRef.current = true;
          navigate(buildInventoryListCreatePath(gotInventoryId), {
            replace: true,
          });
        }

        setDTO(data);

        if (!initializedPriceRowsRef.current) {
          /**
           * dto.priceRows を丸めてはいけない。
           *
           * 特に alcohol の場合、以下を PriceCard まで残す必要がある。
           * - kind: "alcohol"
           * - volumeValue
           * - volumeUnit
           */
          const nextRows = initPriceRowsFromDTOKeepingModelFields(data);
          setPriceRows(nextRows);
          initializedPriceRowsRef.current = true;
        }
      } catch (e) {
        if (cancelled) return;

        const msg = String(e instanceof Error ? e.message : e);
        setDTOError(msg);
      } finally {
        if (cancelled) return;
        setLoadingDTO(false);
      }
    };

    void run();

    return () => {
      cancelled = true;
    };
  }, [navigate, inventoryId, resolvedParams, setPriceRows, initializedPriceRowsRef]);

  const { productBrandName, productName, tokenBrandName, tokenName } =
    React.useMemo(() => extractDisplayStrings(dto), [dto]);

  return {
    dto,
    loadingDTO,
    dtoError,
    productBrandName,
    productName,
    tokenBrandName,
    tokenName,
  };
}