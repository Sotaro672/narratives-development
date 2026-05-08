//frontend\amol\src\features\shipping-address\components\ShippingAddressForm.tsx
import type { FormEvent, ChangeEvent } from "react";

import Input from "../../../components/ui/Input";
import type { ShippingAddressFormValues } from "../types";

type ShippingAddressFormProps = {
  form: ShippingAddressFormValues;
  isLoading: boolean;
  isLookingUpAddress: boolean;
  zipCodeError: string;
  onChange: (
    name: keyof ShippingAddressFormValues
  ) => (event: ChangeEvent<HTMLInputElement>) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
};

export default function ShippingAddressForm({
  form,
  isLoading,
  isLookingUpAddress,
  zipCodeError,
  onChange,
  onSubmit,
}: ShippingAddressFormProps) {
  return (
    <form
      className="settings-form shipping-address-page__form"
      onSubmit={onSubmit}
    >
      {isLoading ? (
        <p className="settings-page__text">読み込み中...</p>
      ) : (
        <>
          <div className="shipping-address-page__name-grid">
            <Input
              label="姓"
              type="text"
              value={form.lastName}
              onChange={onChange("lastName")}
              placeholder="山田"
              autoComplete="family-name"
              required
            />

            <Input
              label="セイ"
              type="text"
              value={form.lastNameKana}
              onChange={onChange("lastNameKana")}
              placeholder="ヤマダ"
              required
            />

            <Input
              label="名"
              type="text"
              value={form.firstName}
              onChange={onChange("firstName")}
              placeholder="太郎"
              autoComplete="given-name"
              required
            />

            <Input
              label="メイ"
              type="text"
              value={form.firstNameKana}
              onChange={onChange("firstNameKana")}
              placeholder="タロウ"
              required
            />
          </div>

          <div className="shipping-address-page__address-grid">
            <Input
              label="郵便番号"
              type="text"
              value={form.zipCode}
              onChange={onChange("zipCode")}
              placeholder="1000001"
              autoComplete="postal-code"
              inputMode="numeric"
              helperText={
                isLookingUpAddress ? "住所を自動入力しています..." : undefined
              }
              error={zipCodeError || undefined}
              required
            />

            <Input
              label="都道府県"
              type="text"
              value={form.state}
              onChange={onChange("state")}
              placeholder="東京都"
              autoComplete="address-level1"
              required
            />

            <Input
              label="市区町村"
              type="text"
              value={form.city}
              onChange={onChange("city")}
              placeholder="千代田区"
              autoComplete="address-level2"
              required
            />

            <Input
              label="住所1"
              type="text"
              value={form.street}
              onChange={onChange("street")}
              placeholder="千代田1-1"
              autoComplete="address-line1"
              required
            />

            <Input
              label="住所2"
              type="text"
              value={form.street2}
              onChange={onChange("street2")}
              placeholder="建物名・部屋番号"
              autoComplete="address-line2"
            />
          </div>
        </>
      )}
    </form>
  );
}