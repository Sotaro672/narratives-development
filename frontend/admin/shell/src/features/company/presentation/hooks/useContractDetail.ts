// frontend/admin/shell/src/features/company/presentation/hooks/useContractDetail.ts

import { useCallback, useEffect, useState } from "react";

import type { ContractDetailResponse } from "../../../../shared/type/contractDetail";
import { getContractDetail } from "../../infrastructure/companyApi";

export function useContractDetail(
  companyId: string | undefined,
) {
  const [detail, setDetail] = useState<ContractDetailResponse | null>(null);
  const [loading, setLoading] = useState(Boolean(companyId?.trim()));
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async (): Promise<void> => {
    const normalizedCompanyId = companyId?.trim() ?? "";

    if (!normalizedCompanyId) {
      setDetail(null);
      setLoading(false);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);
    setDetail(null);

    try {
      const result = await getContractDetail(normalizedCompanyId);
      setDetail(result);
    } catch (cause) {
      setDetail(null);
      setError(
        cause instanceof Error
          ? cause.message
          : "契約詳細の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [companyId]);

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