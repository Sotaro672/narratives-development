// frontend/console/shell/src/features/inventory/presentation/components/InventoryShippingAddressCard.tsx

import * as React from "react";
import { Plus } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "../../../../shared/ui/card";
import type { InventoryShippingAddressDTO } from "../../../../shared/types/inventory";

export type InventoryShippingAddressCardProps = {
  shippingAddressId: string;
  shippingAddressOptions: InventoryShippingAddressDTO[];
  loading?: boolean;
  saving?: boolean;
  onSelectShippingAddress: (shippingAddressId: string) => void;
  onCreateShippingAddress: () => void;
};

/**
 * Inventory Detail 画面の在庫保管場所選択カード。
 * AdminCard の担当者選択 UI と同じく、select では ID を value として扱い、
 * 表示上は shippingAddress.name を表示する。
 * shippingAddress は GET /inventory/{inventoryId} の shippingAddressOptions を唯一の正とする。
 * 在庫保管場所が未登録の場合は新規登録ボタンを表示し、stockLocation 画面への遷移を親componentへ委譲する。
 */
export const InventoryShippingAddressCard: React.FC<InventoryShippingAddressCardProps> = ({
  shippingAddressId,
  shippingAddressOptions,
  loading = false,
  saving = false,
  onSelectShippingAddress,
  onCreateShippingAddress,
}) => {
  const disabled = loading || saving;
  const selectedValue = shippingAddressId || "";
  const hasShippingAddressOptions = shippingAddressOptions.length > 0;

  const handleChange = React.useCallback(
    (event: React.ChangeEvent<HTMLSelectElement>) => {
      if (disabled) return;

      const nextId = event.target.value;
      if (!nextId) return;

      onSelectShippingAddress(nextId);
    },
    [disabled, onSelectShippingAddress],
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle>在庫保管場所</CardTitle>
      </CardHeader>

      <CardContent className="space-y-4">
        <div>
          <div className="mb-1 text-xs text-slate-500">保管場所</div>

          <select
            className="w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm outline-none focus:border-slate-400 disabled:cursor-not-allowed disabled:opacity-50"
            value={selectedValue}
            onChange={handleChange}
            disabled={disabled || !hasShippingAddressOptions}
          >
            <option value="" disabled>
              {loading
                ? "保管場所を読み込み中です…"
                : saving
                  ? "保存中です…"
                  : hasShippingAddressOptions
                    ? "保管場所を選択してください"
                    : "在庫保管場所が登録されていません"}
            </option>

            {shippingAddressOptions.map((address) => (
              <option key={address.id} value={address.id}>
                {address.name}
              </option>
            ))}
          </select>

          {!loading && !hasShippingAddressOptions && (
            <div className="mt-3">
              <p className="text-xs text-slate-400">在庫保管場所が登録されていません。</p>

              <button
                type="button"
                className="mt-3 inline-flex items-center gap-2 rounded-md border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
                onClick={onCreateShippingAddress}
                disabled={saving}
              >
                <Plus size={16} aria-hidden />
                新規登録
              </button>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
};

export default InventoryShippingAddressCard;