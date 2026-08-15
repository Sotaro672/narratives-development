// frontend/console/shell/src/features/brand/presentation/hook/useBrandSelection.ts

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import {
  fetchActiveBrandOptionsForCurrentCompany,
  type BrandOption,
} from "../../application/BrandSelectionService";

export type UseBrandSelectionArgs = {
  initialBrandId?: string | null;
  initialBrandName?: string | null;
  enabled?: boolean;
};

export type UseBrandSelectionResult = {
  brandId: string;
  brandName: string;
  brandOptions: BrandOption[];
  loadingBrands: boolean;
  brandError: Error | null;
  selectBrand: (brandId: string) => void;
  clearBrand: () => void;
};

export function useBrandSelection(
  args: UseBrandSelectionArgs = {},
): UseBrandSelectionResult {
  const {
    initialBrandId = null,
    initialBrandName = null,
    enabled = true,
  } = args;

  const normalizedInitialBrandId = String(initialBrandId ?? "").trim();
  const normalizedInitialBrandName = String(initialBrandName ?? "").trim();

  const [brandId, setBrandId] = useState(normalizedInitialBrandId);
  const [brandOptions, setBrandOptions] = useState<BrandOption[]>([]);
  const [loadingBrands, setLoadingBrands] = useState(false);
  const [brandError, setBrandError] = useState<Error | null>(null);

  const previousInitialBrandIdRef = useRef(normalizedInitialBrandId);

  useEffect(() => {
    if (previousInitialBrandIdRef.current === normalizedInitialBrandId) {
      return;
    }

    previousInitialBrandIdRef.current = normalizedInitialBrandId;
    setBrandId(normalizedInitialBrandId);
  }, [normalizedInitialBrandId]);

  useEffect(() => {
    if (!enabled) {
      setBrandId("");
      setBrandOptions([]);
      setLoadingBrands(false);
      setBrandError(null);
      return;
    }

    let cancelled = false;

    const loadBrands = async () => {
      setLoadingBrands(true);
      setBrandError(null);

      try {
        const options = await fetchActiveBrandOptionsForCurrentCompany();

        if (cancelled) {
          return;
        }

        setBrandOptions(options);

        setBrandId((currentBrandId) => {
          if (!currentBrandId) {
            return "";
          }

          return options.some((brand) => brand.id === currentBrandId)
            ? currentBrandId
            : "";
        });
      } catch (error: unknown) {
        if (cancelled) {
          return;
        }

        setBrandOptions([]);
        setBrandId("");
        setBrandError(
          error instanceof Error
            ? error
            : new Error(String(error)),
        );
      } finally {
        if (!cancelled) {
          setLoadingBrands(false);
        }
      }
    };

    void loadBrands();

    return () => {
      cancelled = true;
    };
  }, [enabled]);

  const brandName = useMemo(() => {
    if (!brandId) {
      return "";
    }

    const matched = brandOptions.find((brand) => brand.id === brandId);

    if (matched) {
      return matched.name;
    }

    if (
      brandId === normalizedInitialBrandId &&
      normalizedInitialBrandName
    ) {
      return normalizedInitialBrandName;
    }

    return "";
  }, [
    brandId,
    brandOptions,
    normalizedInitialBrandId,
    normalizedInitialBrandName,
  ]);

  const selectBrand = useCallback((nextBrandId: string) => {
    setBrandId(String(nextBrandId ?? "").trim());
  }, []);

  const clearBrand = useCallback(() => {
    setBrandId("");
  }, []);

  return {
    brandId,
    brandName,
    brandOptions,
    loadingBrands,
    brandError,
    selectBrand,
    clearBrand,
  };
}