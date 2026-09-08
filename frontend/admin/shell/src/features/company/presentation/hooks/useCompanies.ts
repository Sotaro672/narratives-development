// frontend/admin/shell/src/features/company/presentation/hooks/useCompanies.ts

import { useCallback, useEffect, useState } from "react";

import type { Company } from "../../../../shared/type/company";
import { listCompanies } from "../../infrastructure/companyApi";

export function useCompanies() {
  const [companies, setCompanies] = useState<Company[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async (): Promise<void> => {
    setLoading(true);
    setError(null);

    try {
      const result = await listCompanies();
      setCompanies(result);
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : "企業一覧の取得に失敗しました。",
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  return {
    companies,
    loading,
    error,
    reload,
  };
}