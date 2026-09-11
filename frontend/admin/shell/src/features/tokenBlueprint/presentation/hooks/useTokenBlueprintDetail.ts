// frontend/admin/shell/src/features/tokenBlueprint/presentation/hooks/useTokenBlueprintDetail.ts

import { useCallback, useEffect, useState } from "react";

import { getContractTokenBlueprintDetail } from "../../infrastructure/tokenBlueprintApi";
import type { ContractTokenBlueprintDetailResponse } from "../../model/tokenBlueprintDetail";

export function useTokenBlueprintDetail(
  companyId: string | undefined,
  tokenBlueprintId: string | undefined,
) {
  const [detail, setDetail] =
    useState<ContractTokenBlueprintDetailResponse | null>(null);
  const [loading, setLoading] = useState(
    Boolean(companyId?.trim() && tokenBlueprintId?.trim()),
  );
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async (): Promise<void> => {
    const normalizedCompanyId = companyId?.trim() ?? "";
    const normalizedTokenBlueprintId = tokenBlueprintId?.trim() ?? "";

    if (!normalizedCompanyId || !normalizedTokenBlueprintId) {
      setDetail(null);
      setLoading(false);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);
    setDetail(null);

    try {
      const result = await getContractTokenBlueprintDetail(
        normalizedCompanyId,
        normalizedTokenBlueprintId,
      );
      setDetail(result);
    } catch (cause) {
      setDetail(null);
      setError(
        cause instanceof Error
          ? cause.message
          : "トークン設計詳細の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [companyId, tokenBlueprintId]);

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