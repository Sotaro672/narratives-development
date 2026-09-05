// frontend/admin/shell/src/features/gas/hooks/useGasBalance.ts

import { useCallback, useEffect, useState } from "react";

import type { GasBalance } from "../../../shared/type/gas";
import { getGasBalance } from "../infrastructure/gasApi";

export function useGasBalance() {
  const [balance, setBalance] = useState<GasBalance | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const result = await getGasBalance();
      setBalance(result);
    } catch (error) {
      setBalance(null);
      setError(error instanceof Error ? error.message : "Failed to load gas balance.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  return {
    balance,
    loading,
    error,
    reload,
  };
}