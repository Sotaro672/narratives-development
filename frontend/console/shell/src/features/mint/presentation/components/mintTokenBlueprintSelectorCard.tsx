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
    <Card>
      <CardHeader>
        <CardTitle>トークン設計一覧</CardTitle>
      </CardHeader>

      <CardContent>
        {!selectedBrandId && (
          <div>
            先にブランドを選択してください。
          </div>
        )}

        {selectedBrandId && tokenBlueprintOptions.length > 0 && (
          <div>
            {tokenBlueprintOptions.map((tokenBlueprint) => (
              <button
                key={tokenBlueprint.id}
                type="button"
                aria-pressed={
                  selectedTokenBlueprintId === tokenBlueprint.id
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
          <div>
            選択中のブランドに紐づくトークン設計がありません。
          </div>
        )}
      </CardContent>
    </Card>
  );
}