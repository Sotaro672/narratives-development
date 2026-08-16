// frontend/console/shell/src/features/mint/presentation/hook/useMintFundingEstimate.ts

import * as React from "react";
import type { MintFundingEstimate } from "../../infrastructure/dto/MintRequestRepository";
import { fetchMintFundingEstimateHTTP } from "../../infrastructure/repository/mintRequests";

export type UseMintFundingEstimateInput = {
  productionId: string;
  tokenBlueprintId: string;
  enabled: boolean;
};

export type UseMintFundingEstimateResult = {
  estimate: MintFundingEstimate | null;
  loading: boolean;
  error: string | null;
};

function getErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) return error.message;
  return "SOL見積の取得に失敗しました";
}

/**
 * Mint申請に必要なSOL見積を取得するPresentation hook。
 *
 * Backend BFF responseをそのまま正とし、Frontend側で変換・補完・fallbackは行わない。
 * enabled=false、productionId未指定、tokenBlueprintId未指定の場合は見積状態を初期化する。
 */
export function useMintFundingEstimate(
  input: UseMintFundingEstimateInput,
): UseMintFundingEstimateResult {
  const { productionId, tokenBlueprintId, enabled } = input;

  const [estimate, setEstimate] = React.useState<MintFundingEstimate | null>(null);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (!enabled || !productionId || !tokenBlueprintId) {
      setEstimate(null);
      setLoading(false);
      setError(null);
      return;
    }

    let cancelled = false;

    const run = async () => {
      setEstimate(null);
      setError(null);
      setLoading(true);

      try {
        const result = await fetchMintFundingEstimateHTTP(
          productionId,
          tokenBlueprintId,
        );

        if (!cancelled) setEstimate(result);
      } catch (error: unknown) {
        if (!cancelled) {
          setEstimate(null);
          setError(getErrorMessage(error));
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    };

    void run();

    return () => {
      cancelled = true;
    };
  }, [productionId, tokenBlueprintId, enabled]);

  return {
    estimate,
    loading,
    error,
  };
}