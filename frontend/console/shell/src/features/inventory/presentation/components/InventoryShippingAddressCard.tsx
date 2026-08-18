// frontend/console/shell/src/features/inventory/presentation/components/InventoryShippingAddressCard.tsx

import * as React from "react";

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";

import type {
  InventoryShippingAddressDTO,
} from "../../../../shared/types/inventory";

export type InventoryShippingAddressCardProps = {
  shippingAddressId: string;

  shippingAddressOptions:
    InventoryShippingAddressDTO[];

  loading?: boolean;

  saving?: boolean;

  onSelectShippingAddress: (
    shippingAddressId: string,
  ) => void;
};

/**
 * Inventory Detail 画面の
 * 在庫保管場所選択カード。
 *
 * AdminCard の担当者選択 UI と同じく、
 * select では ID を value として扱い、
 * 表示上は利用者向けの名称を表示する。
 *
 * shippingAddress は
 * GET /inventory/{inventoryId} の
 * shippingAddressOptions を唯一の正とする。
 */
export const InventoryShippingAddressCard:
  React.FC<InventoryShippingAddressCardProps> = ({
    shippingAddressId,

    shippingAddressOptions,

    loading = false,

    saving = false,

    onSelectShippingAddress,
  }) => {
    const disabled =
      loading ||
      saving;

    const selectedValue =
      shippingAddressId || "";

    const handleChange =
      React.useCallback(
        (
          event:
            React.ChangeEvent<HTMLSelectElement>,
        ) => {
          if (disabled) {
            return;
          }

          const nextId =
            event.target.value;

          if (!nextId) {
            return;
          }

          onSelectShippingAddress(
            nextId,
          );
        },
        [
          disabled,
          onSelectShippingAddress,
        ],
      );

    return (
      <Card>
        <CardHeader>
          <CardTitle>
            在庫保管場所
          </CardTitle>
        </CardHeader>

        <CardContent className="space-y-4">
          <div>
            <div className="mb-1 text-xs text-slate-500">
              保管場所
            </div>

            <select
              className="w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm outline-none focus:border-slate-400 disabled:cursor-not-allowed disabled:opacity-50"
              value={selectedValue}
              onChange={handleChange}
              disabled={disabled}
            >
              <option
                value=""
                disabled
              >
                {loading
                  ? "保管場所を読み込み中です…"
                  : saving
                    ? "保存中です…"
                    : "保管場所を選択してください"}
              </option>

              {shippingAddressOptions.map(
                (address) => (
                  <option
                    key={address.id}
                    value={address.id}
                  >
                    {buildShippingAddressLabel(
                      address,
                    )}
                  </option>
                ),
              )}
            </select>

            {!loading &&
              shippingAddressOptions.length ===
                0 && (
                <p className="mt-2 text-xs text-slate-400">
                  在庫保管場所が登録されていません。
                </p>
              )}
          </div>
        </CardContent>
      </Card>
    );
  };

function buildShippingAddressLabel(
  address: InventoryShippingAddressDTO,
): string {
  const zipCode =
    address.zipCode
      ? `〒${address.zipCode}`
      : "";

  const addressLine =
    [
      address.state,
      address.city,
      address.street,
    ]
      .filter(Boolean)
      .join("");

  const street2 =
    address.street2 || "";

  return [
    zipCode,
    addressLine,
    street2,
  ]
    .filter(Boolean)
    .join(" ");
}

export default InventoryShippingAddressCard;