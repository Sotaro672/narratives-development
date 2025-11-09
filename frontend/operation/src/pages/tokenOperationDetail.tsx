// frontend/operation/src/pages/tokenOperationDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../admin/src/pages/AdminCard";
import TokenBlueprintCard from "../../../tokenBlueprint/src/pages/tokenBlueprintCard";
import TokenContentsCard from "../../../tokenContents/src/pages/tokenContentsCard";

export default function TokenOperationDetail() {
  const navigate = useNavigate();
  const { tokenOperationId } = useParams<{ tokenOperationId: string }>();

  // 管理情報（右カラム）
  const [assignee, setAssignee] = React.useState("佐藤 美咲");
  const [creator] = React.useState("山田 太郎");
  const [createdAt] = React.useState("2025/11/06 20:55");

  // 戻る
  const onBack = React.useCallback(() => navigate(-1), [navigate]);

  // 保存ボタンのアクション
  const handleSave = React.useCallback(() => {
    alert("トークン運用情報を保存しました（モック）");
  }, []);

  return (
    <PageStyle
      layout="grid-2"
      title={`トークン運用：${tokenOperationId ?? "不明ID"}`}
      onBack={onBack}
      onSave={handleSave} // ✅ 保存ボタンをPageHeaderに追加
    >
      {/* 左カラム：トークン設計＋コンテンツ */}
      <div>
        {/* トークン設計カード（閲覧用。初期値は空でプレースホルダ表示） */}
        <TokenBlueprintCard />

        {/* コンテンツカード（空状態で表示） */}
        <div style={{ marginTop: 16 }}>
          <TokenContentsCard images={[]} />
        </div>
      </div>

      {/* 右カラム：管理情報 */}
      <AdminCard
        title="管理情報"
        assigneeName={assignee}
        createdByName={creator}
        createdAt={createdAt}
        onEditAssignee={() => setAssignee("新担当者")}
        onClickAssignee={() => console.log("assignee clicked:", assignee)}
        onClickCreatedBy={() => console.log("createdBy clicked:", creator)}
      />
    </PageStyle>
  );
}
