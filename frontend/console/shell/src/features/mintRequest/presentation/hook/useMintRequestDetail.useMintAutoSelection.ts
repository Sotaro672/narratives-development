// frontend/console/mintRequest/src/presentation/hook/useMintRequestDetail.useMintAutoSelection.ts

import * as React from "react";
import type { MintInfo } from "../../application/mapper/mintInfoMapper";

export function useMintAutoSelection(params: {
  hasMint: boolean;
  mintRequestedBrandId: string;
  selectedBrandId: string;
  handleSelectBrand: (brandId: string) => Promise<void> | void;

  mintRequestedTokenBlueprintId: string;
  selectedTokenBlueprintId: string;
  setSelectedTokenBlueprintId: React.Dispatch<React.SetStateAction<string>>;

  mint: MintInfo | null;

  scheduledBurnDate: string;
  setScheduledBurnDate: React.Dispatch<React.SetStateAction<string>>;
}) {
  const {
    hasMint,
    mintRequestedBrandId,
    selectedBrandId,
    handleSelectBrand,
    mintRequestedTokenBlueprintId,
    selectedTokenBlueprintId,
    setSelectedTokenBlueprintId,
    mint,
    scheduledBurnDate,
    setScheduledBurnDate,
  } = params;

  // mint が存在し、brandId が取れるなら「初回だけ」ブランド自動選択
  React.useEffect(() => {
    if (!hasMint) return;
    if (!mintRequestedBrandId) return;
    if (selectedBrandId) return; // 手動選択を尊重

    (async () => {
      try {
        await handleSelectBrand(mintRequestedBrandId);
      } catch {
        // noop
      }
    })();
  }, [hasMint, mintRequestedBrandId, selectedBrandId, handleSelectBrand]);

  // mint が存在し、tokenBlueprintId が取れるなら「初回だけ」tokenBlueprint 自動選択
  React.useEffect(() => {
    if (!hasMint) return;
    if (!mintRequestedTokenBlueprintId) return;
    if (selectedTokenBlueprintId) return; // 手動選択を尊重
    setSelectedTokenBlueprintId(mintRequestedTokenBlueprintId);
  }, [hasMint, mintRequestedTokenBlueprintId, selectedTokenBlueprintId, setSelectedTokenBlueprintId]);

  // mint が存在し、scheduledBurnDate があるなら「初回だけ」入力欄へ反映（手入力を尊重）
  React.useEffect(() => {
    if (!hasMint) return;
    if (scheduledBurnDate) return; // 既に入力されているなら上書きしない

    const raw = mint?.scheduledBurnDate;
    if (!raw) return;

    const s = String(raw);
    const asDate = s.length >= 10 ? s.slice(0, 10) : s;
    if (asDate) setScheduledBurnDate(asDate);
  }, [hasMint, mint, scheduledBurnDate, setScheduledBurnDate]);
}
