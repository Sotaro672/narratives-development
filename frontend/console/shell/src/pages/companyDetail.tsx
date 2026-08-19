// frontend/console/shell/src/pages/companyDetail.tsx

import PageStyle from "../layout/PageStyle/PageStyle";
import { AdminCard } from "../features/admin/presentation/components/AdminCard";
import CompanyNameCard from "../features/company/presentation/components/CompanyNameCard";
import { useCompanyDetail } from "../features/company/presentation/hook/useCompanyDetail";

export default function CompanyDetail() {
  const {
    companyName,
    setCompanyName,
    adminMemberId,
    adminMemberName,
    adminCandidates,
    loadingMembers,
    loading,
    saving,
    error,
    createdByName,
    createdAt,
    updatedByName,
    updatedAt,
    handleSelectAdmin,
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

      {error && <p className="text-sm text-red-500">{error}</p>}
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
    <PageStyle
      layout="grid-2"
      title="会社情報"
      onBack={handleBack}
      onSave={handleSave}
      isSaving={saving}
    >
      {[left, right]}
    </PageStyle>
  );
}