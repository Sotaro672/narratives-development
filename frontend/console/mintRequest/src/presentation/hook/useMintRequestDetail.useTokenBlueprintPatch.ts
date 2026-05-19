// frontend/console/mintRequest/src/presentation/hook/useMintRequestDetail.useTokenBlueprintPatch.ts

import * as React from "react";
import type { MintRequestRepository } from "../../application/port/MintRequestRepository";

export type TokenBlueprintPatchDTO = {
  id?: string | null;
  tokenName?: string | null;
  symbol?: string | null;
  brandId?: string | null;
  brandName?: string | null;
  companyId?: string | null;
  description?: string | null;
  minted?: boolean | null;
  metadataUri?: string | null;
  iconUrl?: string | null;
};

function toTokenBlueprintPatchDTO(input: unknown): TokenBlueprintPatchDTO | null {
  if (!input || typeof input !== "object") return null;

  const raw = input as any;

  const id = String(raw.id ?? "").trim();
  if (!id) return null;

  return {
    id,
    tokenName: String(raw.tokenName ?? raw.name ?? "").trim() || null,
    symbol: String(raw.symbol ?? "").trim() || null,
    brandId: String(raw.brandId ?? "").trim() || null,
    brandName: String(raw.brandName ?? "").trim() || null,
    companyId: String(raw.companyId ?? "").trim() || null,
    description:
      typeof raw.description === "string" ? raw.description : null,
    minted:
      typeof raw.minted === "boolean"
        ? raw.minted
        : typeof raw.mint === "boolean"
          ? raw.mint
          : null,
    metadataUri: String(raw.metadataUri ?? "").trim() || null,
    iconUrl: String(raw.iconUrl ?? "").trim() || null,
  };
}

export function useTokenBlueprintPatch(
  repo: MintRequestRepository,
  tokenBlueprintId: string,
  initialPatch?: TokenBlueprintPatchDTO | null,
) {
  const [tokenBlueprintPatch, setTokenBlueprintPatch] =
    React.useState<TokenBlueprintPatchDTO | null>(initialPatch ?? null);

  React.useEffect(() => {
    const id = String(tokenBlueprintId ?? "").trim();

    if (!id) {
      setTokenBlueprintPatch(initialPatch ?? null);
      return;
    }

    if (initialPatch?.id && String(initialPatch.id).trim() === id) {
      setTokenBlueprintPatch(initialPatch);
      return;
    }

    let cancelled = false;

    setTokenBlueprintPatch(initialPatch ?? null);

    (async () => {
      try {
        const patch = await repo.fetchTokenBlueprintPatch(id);
        if (cancelled) return;

        setTokenBlueprintPatch(toTokenBlueprintPatchDTO(patch));
      } catch {
        if (cancelled) return;

        setTokenBlueprintPatch(initialPatch ?? null);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [repo, tokenBlueprintId, initialPatch]);

  return { tokenBlueprintPatch, setTokenBlueprintPatch };
}