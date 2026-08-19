// frontend/console/shell/src/features/company/presentation/components/ShippingAddressFormModal.tsx

import * as React from "react";
import { X } from "lucide-react";

import type {
  ShippingAddress,
  ShippingAddressFormValue,
} from "../../../../shared/types/shippingAddress";

import { CardInput, CardLabel } from "../../../../shared/ui/card";

type ShippingAddressEditableFormValue = {
  name: string;
  zipCode: string;
  state: string;
  city: string;
  street: string;
  street2: string;
};

export type ShippingAddressFormModalProps = {
  open: boolean;
  address?: ShippingAddress | null;
  saving?: boolean;
  onClose: () => void;
  onSave: (value: ShippingAddressFormValue) => void | Promise<void>;
};

const emptyFormValue: ShippingAddressEditableFormValue = {
  name: "",
  zipCode: "",
  state: "",
  city: "",
  street: "",
  street2: "",
};

function createFormValue(
  address?: ShippingAddress | null,
): ShippingAddressEditableFormValue {
  if (!address) {
    return { ...emptyFormValue };
  }

  return {
    name: address.name,
    zipCode: address.zipCode,
    state: address.state,
    city: address.city,
    street: address.street,
    street2: address.street2,
  };
}

export const ShippingAddressFormModal: React.FC<
  ShippingAddressFormModalProps
> = ({
  open,
  address = null,
  saving = false,
  onClose,
  onSave,
}) => {
  const [form, setForm] = React.useState<ShippingAddressEditableFormValue>(() =>
    createFormValue(address),
  );
  const [error, setError] = React.useState<string | null>(null);

  const isEdit = Boolean(address?.id);

  React.useEffect(() => {
    if (!open) {
      return;
    }

    setForm(createFormValue(address));
    setError(null);
  }, [open, address]);

  React.useEffect(() => {
    if (!open) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && !saving) {
        onClose();
      }
    };

    document.addEventListener("keydown", handleKeyDown);

    return () => {
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open, saving, onClose]);

  const updateField = React.useCallback(
    (field: keyof ShippingAddressEditableFormValue, value: string) => {
      setForm((current) => ({
        ...current,
        [field]: value,
      }));
      setError(null);
    },
    [],
  );

  const handleBackdropMouseDown = React.useCallback(
    (event: React.MouseEvent<HTMLDivElement>) => {
      if (saving) {
        return;
      }

      if (event.target === event.currentTarget) {
        onClose();
      }
    },
    [saving, onClose],
  );

  const handleSubmit = React.useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();

      if (saving) {
        return;
      }

      if (!form.name) {
        setError("保管場所名を入力してください。");
        return;
      }

      if (!form.zipCode) {
        setError("郵便番号を入力してください。");
        return;
      }

      if (!form.state) {
        setError("都道府県を入力してください。");
        return;
      }

      if (!form.city) {
        setError("市区町村を入力してください。");
        return;
      }

      if (!form.street) {
        setError("住所を入力してください。");
        return;
      }

      setError(null);

      await onSave({
        name: form.name,
        zipCode: form.zipCode,
        state: form.state,
        city: form.city,
        street: form.street,
        street2: form.street2,
        country: "JP",
      });
    },
    [form, saving, onSave],
  );

  if (!open) {
    return null;
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4 py-6"
      role="presentation"
      onMouseDown={handleBackdropMouseDown}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="shipping-address-form-title"
        className="w-full max-w-xl overflow-hidden rounded-xl border border-[hsl(var(--border))] bg-[hsl(var(--card))] shadow-xl"
      >
        <div className="flex items-center justify-between border-b border-[hsl(var(--border))] px-5 py-4">
          <h2
            id="shipping-address-form-title"
            className="text-sm font-semibold text-[hsl(var(--foreground))]"
          >
            {isEdit ? "在庫保管場所を編集" : "在庫保管場所を追加"}
          </h2>

          <button
            type="button"
            className="inline-flex h-8 w-8 items-center justify-center rounded-md text-[hsl(var(--muted-foreground))] hover:bg-[hsl(var(--muted))] hover:text-[hsl(var(--foreground))] disabled:cursor-not-allowed disabled:opacity-50"
            onClick={onClose}
            disabled={saving}
            aria-label="閉じる"
          >
            <X size={18} aria-hidden />
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="px-5 py-5">
            <CardLabel htmlFor="shipping-address-name">
              保管場所名
            </CardLabel>

            <CardInput
              id="shipping-address-name"
              name="name"
              type="text"
              placeholder="本社倉庫"
              value={form.name}
              onChange={(event) => updateField("name", event.target.value)}
              disabled={saving}
            />

            <div className="grid grid-cols-1 gap-x-4 sm:grid-cols-2">
              <div>
                <CardLabel htmlFor="shipping-address-zip-code">
                  郵便番号
                </CardLabel>

                <CardInput
                  id="shipping-address-zip-code"
                  name="zipCode"
                  type="text"
                  inputMode="numeric"
                  autoComplete="postal-code"
                  placeholder="100-0001"
                  value={form.zipCode}
                  onChange={(event) =>
                    updateField("zipCode", event.target.value)
                  }
                  disabled={saving}
                />
              </div>

              <div>
                <CardLabel htmlFor="shipping-address-state">
                  都道府県
                </CardLabel>

                <CardInput
                  id="shipping-address-state"
                  name="state"
                  type="text"
                  autoComplete="address-level1"
                  placeholder="東京都"
                  value={form.state}
                  onChange={(event) =>
                    updateField("state", event.target.value)
                  }
                  disabled={saving}
                />
              </div>
            </div>

            <CardLabel htmlFor="shipping-address-city">
              市区町村
            </CardLabel>

            <CardInput
              id="shipping-address-city"
              name="city"
              type="text"
              autoComplete="address-level2"
              placeholder="千代田区"
              value={form.city}
              onChange={(event) => updateField("city", event.target.value)}
              disabled={saving}
            />

            <CardLabel htmlFor="shipping-address-street">
              住所
            </CardLabel>

            <CardInput
              id="shipping-address-street"
              name="street"
              type="text"
              autoComplete="address-line1"
              placeholder="千代田1-1"
              value={form.street}
              onChange={(event) => updateField("street", event.target.value)}
              disabled={saving}
            />

            <CardLabel htmlFor="shipping-address-street2">
              建物名・部屋番号
            </CardLabel>

            <CardInput
              id="shipping-address-street2"
              name="street2"
              type="text"
              autoComplete="address-line2"
              placeholder="AMOLビル 3F"
              value={form.street2}
              onChange={(event) => updateField("street2", event.target.value)}
              disabled={saving}
            />

            {error && (
              <p className="mt-4 text-sm text-red-500">
                {error}
              </p>
            )}
          </div>

          <div className="flex items-center justify-end gap-2 border-t border-[hsl(var(--border))] px-5 py-4">
            <button
              type="button"
              className="inline-flex h-9 items-center justify-center rounded-[10px] border border-[hsl(var(--border))] bg-[hsl(var(--card))] px-4 text-sm text-[hsl(var(--foreground))] hover:bg-[hsl(var(--muted))] disabled:cursor-not-allowed disabled:opacity-50"
              onClick={onClose}
              disabled={saving}
            >
              キャンセル
            </button>

            <button
              type="submit"
              className="inline-flex h-9 items-center justify-center rounded-[10px] bg-[hsl(var(--primary))] px-4 text-sm font-medium text-[hsl(var(--primary-foreground))] hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-50"
              disabled={saving}
            >
              {saving
                ? "保存しています..."
                : isEdit
                  ? "変更を保存"
                  : "住所を追加"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default ShippingAddressFormModal;