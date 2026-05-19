// frontend/console/mintRequest/src/presentation/hook/useMintRequestDetail.useTokenBlueprintPatch.ts

import * as React from "react";

// ✅ ロールバック後に ../../infrastructure/adapter/inventoryTokenBlueprintPatch は存在しないため、
// この hook 内で必要最小限の型を定義する。
export type TokenBlueprintPatchDTO = {
  id?: string | null;
  tokenName?: string | null;
  name?: string | null;
  symbol?: string | null;
  brandId?: string | null;
  brandName?: string | null;
  companyId?: string | null;
  description?: string | null;
  minted?: boolean | null;
  metadataUri?: string | null;
  iconUrl?: string | null;
};

// ✅ B設計: application/usecase + DI
import { mintRequestContainer } from "../di/mintRequestContainer";
import { getTokenBlueprintPatch } from "../../application/usecase/getTokenBlueprintPatch";

export function useTokenBlueprintPatch(tokenBlueprintIdForPatch: string) {
  // DI（infra repo はここで組み立て、hook内ではusecase経由でしか触らない）
  const { mintRequestRepo } = React.useMemo(() => mintRequestContainer(), []);

  const [tokenBlueprintPatch, setTokenBlueprintPatch] =
    React.useState<TokenBlueprintPatchDTO | null>(null);

  React.useEffect(() => {
    if (!tokenBlueprintIdForPatch) {
      setTokenBlueprintPatch(null);
      return;
    }

    let cancelled = false;

    (async () => {
      try {
        const p = await getTokenBlueprintPatch(
          mintRequestRepo,
          tokenBlueprintIdForPatch,
        );

        if (cancelled) return;

        setTokenBlueprintPatch((p ?? null) as TokenBlueprintPatchDTO | null);
      } catch {
        if (cancelled) return;
        setTokenBlueprintPatch(null);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [tokenBlueprintIdForPatch, mintRequestRepo]);

  return { tokenBlueprintPatch, setTokenBlueprintPatch };
}