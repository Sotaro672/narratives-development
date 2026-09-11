// frontend/admin/shell/src/features/brand/presentation/hooks/useBrandDetail.ts

import { useEffect, useState } from "react";

import { useContractDetail } from "../../../company/presentation/hooks/useContractDetail";
import {
  getBrandBackgroundImageUrl,
  getBrandIconUrl,
} from "../../infrastructure/brandAssetStorage";

export function useBrandDetail(
  companyId: string | undefined,
  brandId: string | undefined,
) {
  const resolvedCompanyId = companyId?.trim() ?? "";
  const resolvedBrandId = brandId?.trim() ?? "";

  const {
    detail,
    loading: detailLoading,
    error,
    reload,
  } = useContractDetail(resolvedCompanyId);

  const company = detail?.company ?? null;
  const brand =
    detail?.brands.find((item) => item.id === resolvedBrandId) ?? null;

  const [brandIconUrl, setBrandIconUrl] = useState("");
  const [brandBackgroundImageUrl, setBrandBackgroundImageUrl] = useState("");
  const [assetsLoading, setAssetsLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;

    const loadBrandAssets = async () => {
      if (!resolvedCompanyId || !resolvedBrandId || !brand) {
        setBrandIconUrl("");
        setBrandBackgroundImageUrl("");
        setAssetsLoading(false);
        return;
      }

      setAssetsLoading(true);

      try {
        const [storageBrandIconUrl, storageBrandBackgroundImageUrl] =
          await Promise.all([
            getBrandIconUrl(resolvedCompanyId, resolvedBrandId),
            getBrandBackgroundImageUrl(resolvedCompanyId, resolvedBrandId),
          ]);

        if (cancelled) return;

        setBrandIconUrl(
          storageBrandIconUrl || brand.brandIcon || "",
        );
        setBrandBackgroundImageUrl(
          storageBrandBackgroundImageUrl ||
            brand.brandBackgroundImage ||
            "",
        );
      } finally {
        if (!cancelled) {
          setAssetsLoading(false);
        }
      }
    };

    void loadBrandAssets();

    return () => {
      cancelled = true;
    };
  }, [
    resolvedCompanyId,
    resolvedBrandId,
    brand,
  ]);

  return {
    company,
    brand,
    brandIconUrl,
    brandBackgroundImageUrl,
    loading: detailLoading || assetsLoading,
    error,
    reload,
  };
}