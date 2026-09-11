// frontend/admin/shell/src/features/list/presentation/hooks/useListDetail.ts

import { useCallback, useEffect, useState } from "react";

import { getContractListDetail } from "../../infrastructure/listApi";
import type { ContractListDetailResponse } from "../../model/listDetail";

export function useListDetail(
  companyId: string | undefined,
  listId: string | undefined,
) {
  const [detail, setDetail] = useState<ContractListDetailResponse | null>(null);
  const [loading, setLoading] = useState(
    Boolean(companyId?.trim() && listId?.trim()),
  );
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async (): Promise<void> => {
    const normalizedCompanyId = companyId?.trim() ?? "";
    const normalizedListId = listId?.trim() ?? "";

    if (!normalizedCompanyId || !normalizedListId) {
      setDetail(null);
      setLoading(false);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);
    setDetail(null);

    try {
      const result = await getContractListDetail(
        normalizedCompanyId,
        normalizedListId,
      );
      setDetail(result);
    } catch (cause) {
      setDetail(null);
      setError(
        cause instanceof Error
          ? cause.message
          : "出品詳細の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, [companyId, listId]);

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