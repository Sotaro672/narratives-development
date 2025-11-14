// frontend/tokenBlueprint/src/presentation/pages/tokenBlueprintCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import TokenBlueprintCard from "../components/tokenBlueprintCard";
import TokenContentsCard from "../../../../tokenContents/src/presentation/components/tokenContentsCard";
import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";

/**
 * トークン設計作成ページ
 * frontend/tokenBlueprint/src/domain/entity/tokenBlueprint.tsx の TokenBlueprint
 * スキーマに準拠し、新規作成用のモック状態のみを扱う。
 */
export default function TokenBlueprintCreate() {
  const navigate = useNavigate();

  // 管理情報（新規作成時のメタ情報をモック）
  const [assignee, setAssignee] = React.useState("assignee-001");
  const [createdBy] = React.useState("creator-001");
  const [createdAt] = React.useState("2024-01-20T00:00:00Z");

  // 戻るボタン
  const handleBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // 保存ボタン（実際はフォーム内容を収集して API へ送信する想定）
  const handleSave = React.useCallback(() => {
    console.log("トークン設計を作成しました（モック）");
    alert("トークン設計を作成しました（モック）");
  }, []);

  // TokenBlueprint スキーマに沿った初期値（新規作成用）
  const initialTokenBlueprint: Partial<TokenBlueprint> = {
    id: "", // 新規のため未採番
    name: "",
    symbol: "",
    brandId: "",
    description: "",
    iconId: null,
    contentFiles: [],
    assigneeId: assignee,
    createdBy,
    createdAt,
    // 新規作成時は updated 系も created と同一で初期化しておく
    updatedBy: createdBy,
    updatedAt: createdAt,
    deletedAt: null,
    deletedBy: null,
  };

  return (
    <PageStyle
      layout="grid-2"
      title="トークン設計を作成"
      onBack={handleBack}
      onSave={handleSave}
    >
      {/* 左カラム：トークン設計フォーム＋コンテンツカード */}
      <div>
        {/* 新規作成なので initialEditMode=true で空フォーム表示 */}
        <TokenBlueprintCard
          initialEditMode
          initialTokenBlueprint={initialTokenBlueprint}
        />

        {/* コンテンツカード：新規作成時は空配列で表示（contentFiles と連動想定） */}
        <div style={{ marginTop: 16 }}>
          <TokenContentsCard images={[]} />
        </div>
      </div>

      {/* 右カラム：管理情報（TokenBlueprint のメタ情報に対応） */}
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
