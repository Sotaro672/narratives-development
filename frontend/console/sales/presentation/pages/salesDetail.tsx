// frontend/console/sales/src/presentation/pages/salesDetail.tsx
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../admin/src/presentation/components/AdminCard";
import LogCard from "../../../log/presentation/LogCard";
import SalesOwnersCard from "../components/salesOwnersCard";
import InputCard from "../components/inputCard";

import { useSalesDetail } from "../hook/useSalesDetail";

export default function SalesDetail() {
  const { vm, handlers } = useSalesDetail();

  const {
    sales,
    assigneeId,
    assigneeName,
    createdByName,
    createdAt,
    updatedByName,
    updatedAt,
    owners,
  } = vm;

  const { onBack } = handlers;

  if (!sales) {
    return (
      <PageStyle layout="single" title="営業" onBack={onBack}>
        <p className="p-4 text-sm text-muted-foreground">
          表示可能な営業情報がありません。
        </p>
      </PageStyle>
    );
  }

  return (
    <PageStyle layout="grid-2" title="営業" onBack={onBack}>
      <div className="space-y-4">
        <div style={{ marginTop: 16 }}>
          <SalesOwnersCard owners={owners} />
        </div>

        <InputCard title="入力" />
      </div>

      <div className="space-y-4">
        <AdminCard
          title="管理情報"
          mode="view"
          assigneeId={assigneeId}
          assigneeName={assigneeName}
          createdByName={createdByName}
          createdAt={createdAt}
          updatedByName={updatedByName}
          updatedAt={updatedAt}
        />

        <LogCard title="更新ログ" />
      </div>
    </PageStyle>
  );
}