// frontend/admin/shell/src/features/company/presentation/hooks/useContractProductBlueprintDetail.ts

import { useCallback, useEffect, useState } from "react";

import type { ContractProductBlueprintDetailResponse } from "../../../../shared/type/contractProductBlueprintDetail";
import { getContractProductBlueprintDetail } from "../../infrastructure/companyApi";

export function useContractProductBlueprintDetail(
  companyId: string | undefined,
  productBlueprintId: string | undefined,
) {
  const [detail, setDetail] =
    useState<ContractProductBlueprintDetailResponse | null>(null);
  const [loading, setLoading] = useState(
    Boolean(companyId?.trim() && productBlueprintId?.trim()),
  );
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async (): Promise<void> => {
    const normalizedCompanyId = companyId?.trim() ?? "";
    const normalizedProductBlueprintId = productBlueprintId?.trim() ?? "";

    if (!normalizedCompanyId || !normalizedProductBlueprintId) {
      setDetail(null);
      setLoading(false);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);
    setDetail(null);

    try {
      const result = await getContractProductBlueprintDetail(
        normalizedCompanyId,
        normalizedProductBlueprintId,
      );
      setDetail(result);
    } catch (cause) {
      setDetail(null);
      setError(
        cause instanceof Error
          ? cause.message
          : "商品設計詳細の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [companyId, productBlueprintId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return {
    detail,
    loading,
    error,
    reload,
  };
}