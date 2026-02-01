// frontend/console/inventory/src/presentation/hook/listCreate/useListCreateDTO.ts
import * as React from "react";
import type { useNavigate } from "react-router-dom";

import {
  canFetchListCreate,
  loadListCreateDTOFromParams,
  getInventoryIdFromDTO,
  shouldRedirectToInventoryIdRoute,
  buildInventoryListCreatePath,
  extractDisplayStrings,
  initPriceRowsFromDTO,
  type ResolvedListCreateParams,
  type PriceRowEx,
} from "../../../application/listCreate/listCreateService";

import type { ListCreateDTO } from "../../../infrastructure/http/inventoryRepositoryHTTP";

export function useListCreateDTO(args: {
  navigate: ReturnType<typeof useNavigate>;
  inventoryId: string | undefined;
  resolvedParams: ResolvedListCreateParams;
  initializedPriceRowsRef: React.MutableRefObject<boolean>;
  setPriceRows: React.Dispatch<React.SetStateAction<PriceRowEx[]>>;
}): {
  dto: ListCreateDTO | null;
  loadingDTO: boolean;
  dtoError: string;
  productBrandName: string;
  productName: string;
  tokenBrandName: string;
  tokenName: string;
} {
  const { navigate, inventoryId, resolvedParams, initializedPriceRowsRef, setPriceRows } =
    args;

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
        const data = await loadListCreateDTOFromParams(resolvedParams);
        if (cancelled) return;

        const gotInventoryId = getInventoryIdFromDTO(data);

        // ✅ currentInventoryId は string 必須なので、undefined を渡さない
        const currentInventoryId = String(inventoryId ?? "");

        if (
          shouldRedirectToInventoryIdRoute({
            currentInventoryId,
            gotInventoryId,
            alreadyRedirected: redirectedRef.current,
          })
        ) {
          redirectedRef.current = true;
          navigate(buildInventoryListCreatePath(gotInventoryId), { replace: true });
        }

        setDTO(data);

        if (!initializedPriceRowsRef.current) {
          const nextRows = initPriceRowsFromDTO(data);
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

  const { productBrandName, productName, tokenBrandName, tokenName } = React.useMemo(
    () => extractDisplayStrings(dto),
    [dto],
  );

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
