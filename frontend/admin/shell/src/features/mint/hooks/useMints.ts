// frontend/admin/shell/src/features/mint/hooks/useMints.ts

import { useCallback, useEffect, useState } from "react";

import type { Mint } from "../../../shared/type/mint";
import { listMints } from "../infrastructure/mintApi";

export function useMints() {
  const [mints, setMints] = useState<Mint[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const result = await listMints();
      setMints(result);
    } catch (error) {
      setMints([]);
      setError(error instanceof Error ? error.message : "Failed to load mints.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  return {
    mints,
    loading,
    error,
    reload,
  };
}