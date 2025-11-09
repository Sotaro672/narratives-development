// frontend/tokenBlueprint/src/presentation/pages/tokenBlueprintCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import TokenBlueprintCard from "../components/tokenBlueprintCard";
import TokenContentsCard from "../../../../tokenContents/src/presentation/components/tokenContentsCard";

/**
 * トークン設計作成ページ
 * frontend/shell/src/shared/types/tokenBlueprint.ts に準拠し、
 * 新規作成用として固定のモック状態のみを扱う（既存ID参照はしない）。
 */
export default function TokenBlueprintCreate() {
  const navigate = useNavigate();

  // 管理情報（作成中のメタ情報を簡易的にモック）
  const [assignee, setAssignee] = React.useState("assignee-001");
  const [createdBy] = React.useState("creator-001");
  const [createdAt] = React.useState("2024-01-20");

  // 戻るボタン
  const handleBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // 保存ボタン（実際はフォーム内容を収集して API へ送信する想定）
  const handleSave = React.useCallback(() => {
    console.log("トークン設計を作成しました（モック）");
    alert("トークン設計を作成しました（モック）");
  }, []);

  return (
    <PageStyle
      layout="grid-2"
      title="トークン設計を作成"
      onBack={handleBack}
      onSave={handleSave}
    >
      {/* 左カラム：トークン設計フォーム＋コンテンツカード */}
      <div>
        {/* 新規作成なので initialEditMode で空フォーム表示（実装側で制御） */}
        <TokenBlueprintCard initialEditMode />

        {/* コンテンツカード：新規作成時は空配列で表示 */}
        <div style={{ marginTop: 16 }}>
          <TokenContentsCard images={[]} />
        </div>
      </div>

      {/* 右カラム：管理情報（モック） */}
      <AdminCard
        title="管理情報"
        assigneeName={assignee}
        createdByName={createdBy}
        createdAt={createdAt}
        onEditAssignee={() => setAssignee("new-assignee-id")}
        onClickAssignee={() => console.log("assignee clicked:", assignee)}
        onClickCreatedBy={() => console.log("createdBy clicked:", createdBy)}
      />
    </PageStyle>
  );
}
