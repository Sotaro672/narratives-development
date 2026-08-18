// frontend/console/shell/src/pages/companyDetail.tsx

import { Plus } from "lucide-react";

import PageStyle from "../layout/PageStyle/PageStyle";

import { AdminCard } from "../features/admin/presentation/components/AdminCard";
import CompanyNameCard from "../features/company/presentation/components/CompanyNameCard";
import CompanyShippingAddressCard from "../features/company/presentation/components/CompanyShippingAddressCard";
import ShippingAddressFormModal from "../features/company/presentation/components/ShippingAddressFormModal";
import { useCompanyDetail } from "../features/company/presentation/hook/useCompanyDetail";

export default function CompanyDetail() {
  const {
    companyName,
    setCompanyName,
    adminMemberId,
    adminMemberName,
    adminCandidates,
    loadingMembers,
    shippingAddresses,
    shippingAddressFormOpen,
    editingShippingAddress,
    loading,
    saving,
    shippingAddressSaving,
    error,
    createdByName,
    createdAt,
    updatedByName,
    updatedAt,
    handleSelectAdmin,
    handleOpenCreateShippingAddress,
    handleOpenEditShippingAddress,
    handleCloseShippingAddressForm,
    handleSaveShippingAddress,
    handleDeleteShippingAddress,
    handleBack,
    handleSave,
  } = useCompanyDetail();

  const left = (
    <div className="space-y-4">
      <CompanyNameCard
        companyName={companyName}
        onChangeCompanyName={setCompanyName}
        disabled={loading || saving}
      />

      <div className="space-y-4">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h2 className="text-sm font-semibold text-slate-900">在庫保管場所</h2>
            <p className="mt-1 text-xs text-slate-500">在庫の保管場所として使用する会社住所を登録します。</p>
          </div>

          <button
            type="button"
            className="inline-flex shrink-0 items-center gap-2 rounded-md border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-700 transition hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
            onClick={handleOpenCreateShippingAddress}
            disabled={loading || shippingAddressSaving}
          >
            <Plus size={16} aria-hidden />
            住所を追加
          </button>
        </div>

        {loading ? (
          <div className="rounded-lg border border-slate-200 bg-white p-4 text-sm text-slate-500">
            会社情報を読み込んでいます...
          </div>
        ) : shippingAddresses.length === 0 ? (
          <div className="rounded-lg border border-dashed border-slate-200 bg-white p-6 text-center">
            <p className="text-sm text-slate-500">在庫保管場所が登録されていません。</p>
            <button
              type="button"
              className="mt-3 inline-flex items-center gap-2 text-sm font-medium text-slate-700 hover:text-slate-900"
              onClick={handleOpenCreateShippingAddress}
              disabled={shippingAddressSaving}
            >
              <Plus size={16} aria-hidden />
              最初の住所を登録
            </button>
          </div>
        ) : (
          <div className="space-y-4">
            {shippingAddresses.map((address) => (
              <CompanyShippingAddressCard
                key={address.id}
                address={address}
                disabled={shippingAddressSaving}
                onEdit={() => handleOpenEditShippingAddress(address.id)}
                onDelete={() => handleDeleteShippingAddress(address.id)}
              />
            ))}
          </div>
        )}

        {error && <p className="text-sm text-red-500">{error}</p>}
      </div>
    </div>
  );

  const right = (
    <div className="space-y-4">
      <AdminCard
        title="管理者"
        mode="edit"
        assigneeId={adminMemberId}
        assigneeName={adminMemberName || "未設定"}
        assigneeCandidates={adminCandidates}
        loadingMembers={loadingMembers}
        onSelectAssignee={handleSelectAdmin}
        createdByName={createdByName}
        createdAt={createdAt}
        updatedByName={updatedByName}
        updatedAt={updatedAt}
      />
    </div>
  );

  return (
    <>
      <PageStyle
        layout="grid-2"
        title="会社情報"
        onBack={handleBack}
        onSave={handleSave}
        isSaving={saving}
      >
        {[left, right]}
      </PageStyle>

      <ShippingAddressFormModal
        open={shippingAddressFormOpen}
        address={editingShippingAddress}
        saving={shippingAddressSaving}
        onClose={handleCloseShippingAddressForm}
        onSave={handleSaveShippingAddress}
      />
    </>
  );
}