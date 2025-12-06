// frontend/console/tokenOperation/src/presentation/pages/tokenOperationDetail.tsx

import * as React from "react";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import TokenBlueprintCard from "../../../../tokenBlueprint/src/presentation/components/tokenBlueprintCard";
import TokenContentsCard from "../../../../tokenContents/src/presentation/components/tokenContentsCard";
import { useTokenOperationDetail } from "../hook/useTokenOperationDetail";

export default function TokenOperationDetail() {
  const {
    title,
    loading,
    error,
    blueprint,
    cardVm,
    cardHandlers,
    assignee,
    creator,
    createdAt,
    onBack,
    handleSave,
  } = useTokenOperationDetail();

  if (loading) {
    return (
      <PageStyle layout="grid-2" title={title} onBack={onBack}>
        <div>読み込み中です…</div>
      </PageStyle>
    );
  }

  if (error || !blueprint) {
    return (
      <PageStyle layout="grid-2" title={title} onBack={onBack}>
        <div>{error ?? "トークン設計が見つかりませんでした。"}</div>
      </PageStyle>
    );
  }

  return (
    <PageStyle layout="grid-2" title={title} onBack={onBack} onSave={handleSave}>
      {/* 左カラム：トークン設計＋コンテンツ */}
      <div>
        <TokenBlueprintCard vm={cardVm} handlers={cardHandlers} />

        <div style={{ marginTop: 16 }}>
          {/* TokenContentsCard: TokenBlueprint.contentFiles（ID配列）と連動させる想定 */}
          <TokenContentsCard mode="edit" images={blueprint.contentFiles ?? []} />
        </div>
      </div>

      {/* 右カラム：管理情報（現状モック） */}
      <AdminCard
        title="管理情報"
        assigneeName={assignee}
        createdByName={creator}
        createdAt={createdAt}
        onEditAssignee={undefined /* hook 側で必要になれば拡張 */}
        onClickAssignee={undefined /* hook 側で必要になれば拡張 */}
      />
    </PageStyle>
  );
}
