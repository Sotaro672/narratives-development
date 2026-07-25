// frontend/console/productBlueprint/src/presentation/hook/useProductBlueprintCreateBrand.ts

import * as React from "react";

import type { Brand } from "../../../../../brand/src/domain/entity/brand";
import { fetchAllBrandsForCompany } from "../../../../../brand/src/infrastructure/query/brandQuery";

export type UseProductBlueprintCreateBrandResult = {
  brandId: string;
  brandName: string;
  brandOptions: Brand[];
  brandLoading: boolean;
  brandError: Error | null;
  onChangeBrandId: (id: string) => void;
};

export function useProductBlueprintCreateBrand(
  companyId: string,
): UseProductBlueprintCreateBrandResult {
  const [brandId, setBrandId] = React.useState("");
  const [brandOptions, setBrandOptions] = React.useState<Brand[]>([]);
  const [brandLoading, setBrandLoading] = React.useState(false);
  const [brandError, setBrandError] = React.useState<Error | null>(null);

  React.useEffect(() => {
    let cancelled = false;

    async function loadBrands() {
      if (!companyId) {
        setBrandOptions([]);
        return;
      }

      setBrandLoading(true);
      setBrandError(null);

      try {
        const items = await fetchAllBrandsForCompany(companyId, true);

        if (!cancelled) {
          setBrandOptions(items);
        }
      } catch (error) {
        const err = error instanceof Error ? error : new Error(String(error));

        if (!cancelled) {
          setBrandError(err);
        }
      } finally {
        if (!cancelled) {
          setBrandLoading(false);
        }
      }
    }

    void loadBrands();

    return () => {
      cancelled = true;
    };
  }, [companyId]);

  const brandName = React.useMemo(() => {
    const found = brandOptions.find((brand) => brand.id === brandId);
    return found?.name ?? "";
  }, [brandId, brandOptions]);

  return {
    brandId,
    brandName,
    brandOptions,
    brandLoading,
    brandError,
    onChangeBrandId: setBrandId,
  };
}