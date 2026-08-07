// frontend/console/shell/src/features/productBlueprint/presentation/hooks/shared/useBrandOptions.ts
import * as React from "react";
import { listBrands } from "../../../../brand/application/brandService";

export type BrandOption = {
  id: string;
  name: string;
};

export type UseBrandOptionsArgs = {
  /**
   * ProductBlueprintDetailのcompanyId。
   * 現在の認証情報が読み込まれるまで一覧を取得しないために使用する。
   *
   * BackendへのQuery Parameterとしては送信しない。
   */
  companyId?: string | null;
  /**
   * 詳細に設定されているbrandId。
   * 表示名解決のfallbackに使用する。
   */
  brandId?: string | null;
  /**
   * Service側で解決済みのbrandName。
   * 値がある場合は最優先で使用する。
   */
  brandNameFromService?: string | null;
};

export type UseBrandOptionsResult = {
  brandOptions: BrandOption[];
  brandLoading: boolean;
  brandError: Error | null;
  /**
   * 表示用ブランド名。
   *
   * brandNameFromServiceがあればそれを使用し、
   * なければbrandOptionsからbrandIdに対応する名前を取得する。
   */
  resolvedBrandName: string;
  /**
   * brandIdに対応するブランド名を返す。
   */
  getBrandNameById: (id: string) => string;
};

export function useBrandOptions(args: UseBrandOptionsArgs): UseBrandOptionsResult {
  const companyId = String(args.companyId ?? "");
  const brandId = String(args.brandId ?? "");
  const brandNameFromService = String(args.brandNameFromService ?? "");
  const [brandOptions, setBrandOptions] = React.useState<BrandOption[]>([]);
  const [brandLoading, setBrandLoading] = React.useState(false);
  const [brandError, setBrandError] = React.useState<Error | null>(null);

  React.useEffect(() => {
    let cancelled = false;
    const loadBrandOptions = async () => {
      if (!companyId) {
        setBrandOptions([]);
        setBrandLoading(false);
        setBrandError(null);
        return;
      }
      try {
        setBrandLoading(true);
        setBrandError(null);
        const brands = await listBrands();
        const options: BrandOption[] = brands.map((brand) => ({
          id: brand.id,
          name: brand.name,
        }));
        if (!cancelled) {
          setBrandOptions(options);
        }
      } catch (error: unknown) {
        if (!cancelled) {
          const normalizedError = error instanceof Error ? error : new Error(String(error));
          setBrandError(normalizedError);
          setBrandOptions([]);
        }
      } finally {
        if (!cancelled) {
          setBrandLoading(false);
        }
      }
    };
    void loadBrandOptions();
    return () => {
      cancelled = true;
    };
  }, [companyId]);

  const getBrandNameById = React.useCallback(
    (id: string): string => {
      const brandIdToFind = String(id ?? "");
      if (!brandIdToFind) {
        return "";
      }
      return brandOptions.find((option) => option.id === brandIdToFind)?.name ?? "";
    },
    [brandOptions],
  );

  const resolvedBrandName = React.useMemo(() => {
    if (brandNameFromService) {
      return brandNameFromService;
    }
    if (!brandId) {
      return "";
    }
    return getBrandNameById(brandId);
  }, [brandNameFromService, brandId, getBrandNameById]);

  return {
    brandOptions,
    brandLoading,
    brandError,
    resolvedBrandName,
    getBrandNameById,
  };
}