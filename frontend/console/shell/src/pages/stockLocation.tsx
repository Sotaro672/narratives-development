// frontend/console/shell/src/pages/stockLocation.tsx

import { Plus } from "lucide-react";

import PageStyle from "../layout/PageStyle/PageStyle";

import CompanyShippingAddressCard from "../features/company/presentation/components/CompanyShippingAddressCard";
import ShippingAddressFormModal from "../features/company/presentation/components/ShippingAddressFormModal";
import { useStockLocation } from "../features/company/presentation/hook/useStockLocation";

export default function StockLocation() {
  const {
    shippingAddresses,
    shippingAddressFormOpen,
    editingShippingAddress,

    loading,
    saving,
    error,

    handleBack,
    handleOpenCreateShippingAddress,
    handleOpenEditShippingAddress,
    handleCloseShippingAddressForm,
    handleSaveShippingAddress,
    handleDeleteShippingAddress,
  } = useStockLocation();

  return (
    <>
      <PageStyle
        layout="single"
        title="在庫保管場所"
        onBack={handleBack}
      >
        <div className="max-w-3xl space-y-4">
          <div className="flex items-center justify-between gap-4">
            <p className="text-sm text-slate-500">
              在庫の保管場所として使用する会社住所を登録します。
            </p>

            <button
              type="button"
              className="inline-flex shrink-0 items-center gap-2 rounded-md border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
              onClick={handleOpenCreateShippingAddress}
              disabled={loading || saving}
            >
              <Plus
                size={16}
                aria-hidden
              />
              住所を追加
            </button>
          </div>

          {loading ? (
            <div className="rounded-lg border border-slate-200 bg-white p-4 text-sm text-slate-500">
              在庫保管場所を読み込んでいます...
            </div>
          ) : shippingAddresses.length === 0 ? (
            <div className="rounded-lg border border-dashed border-slate-200 bg-white p-6 text-center">
              <p className="text-sm text-slate-500">
                在庫保管場所が登録されていません。
              </p>

              <button
                type="button"
                className="mt-3 inline-flex items-center gap-2 text-sm font-medium text-slate-700 hover:text-slate-900 disabled:cursor-not-allowed disabled:opacity-50"
                onClick={handleOpenCreateShippingAddress}
                disabled={saving}
              >
                <Plus
                  size={16}
                  aria-hidden
                />
                最初の住所を登録
              </button>
            </div>
          ) : (
            <div className="space-y-4">
              {shippingAddresses.map((address) => (
                <CompanyShippingAddressCard
                  key={address.id}
                  address={address}
                  disabled={saving}
                  onEdit={() =>
                    handleOpenEditShippingAddress(
                      address.id,
                    )
                  }
                  onDelete={() =>
                    handleDeleteShippingAddress(
                      address.id,
                    )
                  }
                />
              ))}
            </div>
          )}

          {error && (
            <p className="text-sm text-red-500">
              {error}
            </p>
          )}
        </div>
      </PageStyle>

      <ShippingAddressFormModal
        open={shippingAddressFormOpen}
        address={editingShippingAddress}
        saving={saving}
        onClose={handleCloseShippingAddressForm}
        onSave={handleSaveShippingAddress}
      />
    </>
  );
}