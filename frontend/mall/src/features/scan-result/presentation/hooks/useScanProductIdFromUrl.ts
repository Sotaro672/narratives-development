// frontend/mall/src/features/scan-result/presentation/hooks/useScanProductIdFromUrl.ts

import { useMemo } from "react";
import { useParams, useSearchParams } from "react-router-dom";

function safeDecodeURIComponent(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}

export function useScanProductIdFromUrl(): string {
  const params = useParams();
  const [searchParams] = useSearchParams();

  return useMemo(() => {
    const fromQuery = searchParams.get("productId");

    if (fromQuery?.trim()) {
      return fromQuery.trim();
    }

    const fromParams = params.productId;

    if (fromParams?.trim()) {
      return safeDecodeURIComponent(fromParams.trim());
    }

    return "";
  }, [params.productId, searchParams]);
}