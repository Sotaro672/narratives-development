// frontend/console/shell/src/features/company/presentation/components/CompanyShippingAddressCard.tsx

import * as React from "react";
import { Pencil, Trash2 } from "lucide-react";

import type { ShippingAddress } from "../../../../shared/types/shippingAddress";

import {
  Card,
  CardButton,
  CardContent,
  CardHeader,
  CardTitle,
} from "../../../../shared/ui/card";

export type CompanyShippingAddressCardProps = {
  address: ShippingAddress;
  disabled?: boolean;
  onEdit: () => void;
  onDelete: () => void;
};

export const CompanyShippingAddressCard: React.FC<
  CompanyShippingAddressCardProps
> = ({
  address,
  disabled = false,
  onEdit,
  onDelete,
}) => {
  const mainAddress = [
    address.state,
    address.city,
    address.street,
  ]
    .filter(Boolean)
    .join(" ");

  const showCountry =
    address.country &&
    address.country !== "JP";

  return (
    <Card>
      <CardHeader className="justify-between">
        <CardTitle>
          登録住所
        </CardTitle>

        <div className="flex items-center gap-2">
          <CardButton
            type="button"
            onClick={onEdit}
            disabled={disabled}
            aria-label="住所を編集"
          >
            <Pencil size={15} aria-hidden />
            編集
          </CardButton>

          <CardButton
            type="button"
            onClick={onDelete}
            disabled={disabled}
            aria-label="住所を削除"
            className="text-red-600 hover:bg-red-50 hover:text-red-700"
          >
            <Trash2 size={15} aria-hidden />
            削除
          </CardButton>
        </div>
      </CardHeader>

      <CardContent>
        <div className="space-y-1 text-sm text-[hsl(var(--foreground))]">
          <div className="font-medium">
            〒{address.zipCode}
          </div>

          <div>
            {mainAddress}
          </div>

          {address.street2 && (
            <div>
              {address.street2}
            </div>
          )}

          {showCountry && (
            <div className="text-xs text-[hsl(var(--muted-foreground))]">
              {address.country}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
};

export default CompanyShippingAddressCard;