// frontend/console/shell/src/features/productBlueprint/presentation/hooks/useProductBlueprintCreateBrand.ts
import * as React from "react";
import { listBrands } from "../../../../brand/application/brandService";

export type BrandOption = {
  id: string;
  name: string;
};

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
  const [brandId, setBrandId] = React.useState("");
  const [brandOptions, setBrandOptions] = React.useState<BrandOption[]>([]);
  const [brandLoading, setBrandLoading] = React.useState(false);
  const [brandError, setBrandError] = React.useState<Error | null>(null);

  React.useEffect(() => {
    let cancelled = false;

    async function loadBrands() {
      if (!companyId) {
        setBrandId("");
        setBrandOptions([]);
        setBrandLoading(false);
        setBrandError(null);
        return;
      }

      try {
        setBrandLoading(true);
        setBrandError(null);

        const brands = await listBrands();
        const options: BrandOption[] = brands
          .filter((brand) => brand.isActive)
          .map((brand) => ({
            id: brand.id,
            name: brand.name,
          }));

        if (!cancelled) {
          setBrandOptions(options);
          setBrandId((currentBrandId) => {
            if (
              currentBrandId &&
              options.some((option) => option.id === currentBrandId)
            ) {
              return currentBrandId;
            }
            return "";
          });
        }
      } catch (error: unknown) {
        if (!cancelled) {
          const normalizedError =
            error instanceof Error ? error : new Error(String(error));
          setBrandId("");
          setBrandOptions([]);
          setBrandError(normalizedError);
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
    if (!brandId) {
      return "";
    }
    return brandOptions.find((brand) => brand.id === brandId)?.name ?? "";
  }, [brandId, brandOptions]);

  const onChangeBrandId = React.useCallback((id: string) => {
    setBrandId(id);
  }, []);

  return {
    brandId,
    brandName,
    brandOptions,
    brandLoading,
    brandError,
    onChangeBrandId,
  };
}