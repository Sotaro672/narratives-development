// frontend/console/member/src/presentation/components/BrandSelect.tsx
import * as React from "react";
import type { BrandRow } from "../hooks/useMemberCreate";
import {
  Popover,
  PopoverTrigger,
  PopoverContent,
} from "../../../../shell/src/shared/ui/popover";
import { Button } from "../../../../shell/src/shared/ui/button";
import { Checkbox } from "../../../../shell/src/shared/ui/checkbox";
import { Badge } from "../../../../shell/src/shared/ui/badge";

type BrandSelectProps = {
  brandRows: BrandRow[];
  selectedBrandIds: Set<string>;
  onToggleBrand: (id: string, explicit?: boolean) => void;
};

export function BrandSelect({
  brandRows,
  selectedBrandIds,
  onToggleBrand,
}: BrandSelectProps) {
  const selectedCount = React.useMemo(
    () => Array.from(selectedBrandIds).length,
    [selectedBrandIds],
  );

  return (
    <div>
      <label className="block text-sm text-slate-300 mb-1">
        ブランド（任意・複数選択可）
      </label>

      <Popover>
        <PopoverTrigger>
          <Button
            type="button"
            variant="outline"
            className="w-full justify-start text-left"
          >
            {selectedCount > 0
              ? `選択中のブランド: ${selectedCount}件`
              : "ブランドを選択"}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-[320px] max-h-[400px] overflow-y-auto text-sm">
          {brandRows.length === 0 ? (
            <p className="text-xs text-[hsl(var(--muted-foreground))]">
              現在、選択可能なブランドがありません。
            </p>
          ) : (
            <ul className="space-y-2">
              {brandRows.map((b) => {
                const checked = selectedBrandIds.has(b.id);
                const inputId = `brand_${b.id}`;
                return (
                  <li key={b.id} className="flex items-center gap-2">
                    <Checkbox
                      id={inputId}
                      checked={checked}
                      onCheckedChange={(v) => onToggleBrand(b.id, !!v)}
                    />
                    <label
                      id={inputId}
                      className="cursor-pointer select-none"
                      onClick={() => onToggleBrand(b.id)}
                    >
                      {b.name}
                    </label>
                  </li>
                );
              })}
            </ul>
          )}
        </PopoverContent>
      </Popover>

      {/* 選択済みブランドのバッジ表示 */}
      <div className="mt-2 flex flex-wrap gap-2">
        {selectedCount === 0 ? (
          <span className="text-xs text-[hsl(var(--muted-foreground))]">
            選択したブランドがここに表示されます。
          </span>
        ) : (
          brandRows
            .filter((b) => selectedBrandIds.has(b.id))
            .map((b) => (
              <Badge key={`brand_badge_${b.id}`}>{b.name}</Badge>
            ))
        )}
      </div>
    </div>
  );
}
