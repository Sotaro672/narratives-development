// frontend/console/shell/src/features/inventory/presentation/components/InventoryShippingAddressCard.tsx

import * as React from "react";
import { Plus } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle } from "../../../../shared/ui/card";
import Text from "../../../../shared/ui/text";
import type { InventoryShippingAddressDTO } from "../../../../shared/types/inventory";

import "../../../../styles/inventory.css";

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
 * ヘッダーの新規登録ボタンから stockLocation 画面への遷移を親componentへ委譲する。
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
      <CardHeader className="inventory-shipping-address__header">
        <CardTitle>在庫保管場所</CardTitle>

        <button
          type="button"
          className="inventory-shipping-address__create-button"
          onClick={onCreateShippingAddress}
          disabled={disabled}
        >
          <Plus size={16} aria-hidden />
          新規登録
        </button>
      </CardHeader>

      <CardContent className="inventory-shipping-address__content">
        <div>
          <Text
            as="div"
            size="xs"
            tone="muted"
            className="inventory-shipping-address__label"
          >
            保管場所
          </Text>

          <select
            className="inventory-shipping-address__select"
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

          {!loading && !hasShippingAddressOptions ? (
            <Text
              as="p"
              size="xs"
              tone="muted"
              className="inventory-shipping-address__empty"
            >
              在庫保管場所が登録されていません。
            </Text>
          ) : null}
        </div>
      </CardContent>
    </Card>
  );
};

export default InventoryShippingAddressCard;