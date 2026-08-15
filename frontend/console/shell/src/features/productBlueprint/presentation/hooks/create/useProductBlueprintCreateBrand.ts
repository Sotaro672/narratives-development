// frontend/console/shell/src/features/productBlueprint/presentation/hooks/create/useProductBlueprintCreateBrand.ts

import { useBrandSelection } from "../../../../brand/presentation/hook/useBrandSelection";
import type { BrandOption } from "../../../../brand/application/BrandSelectionService";

export type { BrandOption };

export type UseProductBlueprintCreateBrandResult = {
  brandId: string;
  brandName: string;
  brandOptions: BrandOption[];
  brandLoading: boolean;
  brandError: Error | null;
  onChangeBrandId: (id: string) => void;
};

export function useProductBlueprintCreateBrand(
  companyId: string,
): UseProductBlueprintCreateBrandResult {
  const {
    brandId,
    brandName,
    brandOptions,
    loadingBrands,
    brandError,
    selectBrand,
  } = useBrandSelection({
    enabled: Boolean(companyId),
  });

  return {
    brandId,
    brandName,
    brandOptions,
    brandLoading: loadingBrands,
    brandError,
    onChangeBrandId: selectBrand,
  };
}