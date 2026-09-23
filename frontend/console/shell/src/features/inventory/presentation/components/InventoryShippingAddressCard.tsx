// frontend/console/shell/src/features/inventory/presentation/components/InventoryShippingAddressCard.tsx

import * as React from "react";
import { Plus } from "lucide-react";

import type { InventoryShippingAddressDTO } from "../../../../shared/types/inventory";
import { Button } from "../../../../shared/ui/button";
import {
  Card,
  CardContent,
  CardField,
  CardHeader,
  CardSelect,
  CardSelectWrap,
  CardTitle,
} from "../../../../shared/ui/card";
import { Label } from "../../../../shared/ui/label";
import Stack from "../../../../shared/ui/stack";
import Text from "../../../../shared/ui/text";

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
export const InventoryShippingAddressCard: React.FC<
  InventoryShippingAddressCardProps
> = ({
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

        <Button
          type="button"
          variant="outline"
          onClick={onCreateShippingAddress}
          disabled={disabled}
        >
          <Plus size={16} aria-hidden />
          新規登録
        </Button>
      </CardHeader>

      <CardContent>
        <Stack gap="sm">
          <CardField>
            <Label htmlFor="inventory-shipping-address">
              保管場所
            </Label>

            <CardSelectWrap>
              <CardSelect
                id="inventory-shipping-address"
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
              </CardSelect>
            </CardSelectWrap>
          </CardField>

          {!loading && !hasShippingAddressOptions ? (
            <Text as="p" size="xs" tone="muted">
              在庫保管場所が登録されていません。
            </Text>
          ) : null}
        </Stack>
      </CardContent>
    </Card>
  );
};

export default InventoryShippingAddressCard;