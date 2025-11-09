// frontend/operation/src/pages/tokenOperationDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../admin/src/pages/AdminCard";
import TokenBlueprintCard from "../../../tokenBlueprint/src/pages/tokenBlueprintCard";
import TokenContentsCard from "../../../tokenContents/src/pages/tokenContentsCard";
import { TOKEN_BLUEPRINTS } from "../../../tokenBlueprint/mockdata";
import { MOCK_IMAGES } from "../../../tokenContents/mockdata";

export default function TokenOperationDetail() {
  const navigate = useNavigate();
  const { tokenOperationId } = useParams<{ tokenOperationId: string }>();

  // ─────────────────────────────────────────
  // モックデータ（トークン設計 & コンテンツ）
  // ─────────────────────────────────────────
  const blueprint = TOKEN_BLUEPRINTS[0];
  const contentImages = MOCK_IMAGES;

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
      onSave={handleSave}
    >
      {/* 左カラム：トークン設計＋コンテンツ（モックでプリフィル） */}
      <div>
        <TokenBlueprintCard
          initialTokenBlueprintId={blueprint.tokenBlueprintId}
          initialTokenName={blueprint.name}
          initialSymbol={blueprint.symbol}
          initialBrand={blueprint.brand}
          initialDescription={blueprint.description}
          initialBurnAt={blueprint.burnAt}
          initialIconUrl={blueprint.iconUrl}
          initialEditMode={false}
        />

        <div style={{ marginTop: 16 }}>
          {/* ✅ コンテンツを編集モードで表示 */}
          <TokenContentsCard images={contentImages} mode="edit" />
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
