// frontend/console/productBlueprint/src/presentation/hook/useBrandOptions.ts
import * as React from "react";
import { fetchAllBrandsForCompany } from "../../../../brand/src/infrastructure/query/brandQuery";

export type BrandOption = {
  id: string;
  name: string;
};

export type UseBrandOptionsArgs = {
  /**
   * ProductBlueprintDetail の companyId（空なら fetch しない）
   */
  companyId?: string | null;

  /**
   * 詳細に入っている brandId（fallback のため）
   */
  brandId?: string | null;

  /**
   * service 側が brandName を解決して返してくる場合があるため（最優先）
   */
  brandNameFromService?: string | null;
};

export type UseBrandOptionsResult = {
  brandOptions: BrandOption[];
  brandLoading: boolean;
  brandError: Error | null;

  /**
   * 表示用 brand 名（service の brandName があればそれ。なければ options から brandId で解決）
   */
  resolvedBrandName: string;

  /**
   * brandId から name を引く helper（UI 側の onChangeBrandId で便利）
   */
  getBrandNameById: (id: string) => string;
};

/**
 * ブランド候補一覧の取得と、brandId -> brandName 解決をまとめる hook。
 *
 * - companyId が空の場合は fetch しない（一覧は空）
 * - brandNameFromService があればそれを最優先
 * - それ以外は brandId を options で引ければ name を返す
 */
export function useBrandOptions(args: UseBrandOptionsArgs): UseBrandOptionsResult {
  const companyId = String(args.companyId ?? "").trim();
  const brandId = String(args.brandId ?? "").trim();
  const brandNameFromService = String(args.brandNameFromService ?? "").trim();

  const [brandOptions, setBrandOptions] = React.useState<BrandOption[]>([]);
  const [brandLoading, setBrandLoading] = React.useState<boolean>(false);
  const [brandError, setBrandError] = React.useState<Error | null>(null);

  React.useEffect(() => {
    let cancelled = false;

    (async () => {
      if (!companyId) {
        // companyId が無い場合は fetch しない
        setBrandOptions([]);
        setBrandLoading(false);
        setBrandError(null);
        return;
      }

      setBrandLoading(true);
      setBrandError(null);

      try {
        const brands = await fetchAllBrandsForCompany(companyId, false);
        const options: BrandOption[] = (brands ?? []).map((b: any) => ({
          id: String(b?.id ?? ""),
          name: String(b?.name ?? ""),
        }));

        if (!cancelled) {
          setBrandOptions(options);
        }
      } catch (e) {
        if (!cancelled) {
          setBrandError(e as Error);
          setBrandOptions([]);
        }
      } finally {
        if (!cancelled) {
          setBrandLoading(false);
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [companyId]);

  const getBrandNameById = React.useCallback(
    (id: string): string => {
      const key = String(id ?? "").trim();
      if (!key) return "";
      return brandOptions.find((o) => o.id === key)?.name ?? "";
    },
    [brandOptions],
  );

  const resolvedBrandName = React.useMemo(() => {
    if (brandNameFromService) return brandNameFromService;
    if (!brandId) return "";
    return getBrandNameById(brandId) || "";
  }, [brandNameFromService, brandId, getBrandNameById]);

  return {
    brandOptions,
    brandLoading,
    brandError,
    resolvedBrandName,
    getBrandNameById,
  };
}
