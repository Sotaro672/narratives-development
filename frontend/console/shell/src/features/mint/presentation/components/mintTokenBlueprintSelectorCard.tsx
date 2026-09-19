// frontend/console/shell/src/features/mint/presentation/components/mintTokenBlueprintSelectorCard.tsx

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";

import type { TokenBlueprintSummary } from "../../infrastructure/dto/MintRequestRepository";

export type MintTokenBlueprintSelectorCardProps = {
  selectedBrandId: string;
  tokenBlueprintOptions: TokenBlueprintSummary[];
  selectedTokenBlueprintId: string;
  disabled?: boolean;
  onSelectTokenBlueprint: (tokenBlueprintId: string) => void;
};

export default function MintTokenBlueprintSelectorCard({
  selectedBrandId,
  tokenBlueprintOptions,
  selectedTokenBlueprintId,
  disabled = false,
  onSelectTokenBlueprint,
}: MintTokenBlueprintSelectorCardProps) {
  return (
    <Card className="pb-select">
      <CardHeader>
        <CardTitle>トークン設計一覧</CardTitle>
      </CardHeader>

      <CardContent>
        {!selectedBrandId && (
          <div className="pb-select__empty">
            先にブランドを選択してください。
          </div>
        )}

        {selectedBrandId && tokenBlueprintOptions.length > 0 && (
          <div className="pb-select__list">
            {tokenBlueprintOptions.map((tokenBlueprint) => (
              <button
                key={tokenBlueprint.id}
                type="button"
                className={
                  "pb-select__row" +
                  (selectedTokenBlueprintId === tokenBlueprint.id
                    ? " is-active"
                    : "")
                }
                onClick={() =>
                  onSelectTokenBlueprint(tokenBlueprint.id)
                }
                disabled={disabled}
              >
                {tokenBlueprint.tokenName}
              </button>
            ))}
          </div>
        )}

        {selectedBrandId && tokenBlueprintOptions.length === 0 && (
          <div className="pb-select__empty">
            選択中のブランドに紐づくトークン設計がありません。
          </div>
        )}
      </CardContent>
    </Card>
  );
}