// frontend/tokenBlueprint/src/pages/tokenBlueprintDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../admin/src/presentation/components/AdminCard";
import TokenBlueprintCard from "./tokenBlueprintCard";
import TokenContentsCard from "../../../tokenContents/src/pages/tokenContentsCard";
import { TOKEN_BLUEPRINTS } from "../../mockdata";

export default function TokenBlueprintDetail() {
  const navigate = useNavigate();
  const { tokenBlueprintId } = useParams<{ tokenBlueprintId: string }>();

  // ─────────────────────────────────────────────
  // 該当するトークン設計データを取得（ID一致で検索）
  // ─────────────────────────────────────────────
  const blueprint = React.useMemo(() => {
    return (
      TOKEN_BLUEPRINTS.find(
        (b) => b.tokenBlueprintId === tokenBlueprintId
      ) || TOKEN_BLUEPRINTS[0]
    );
  }, [tokenBlueprintId]);

  // 管理情報（右カラム用）
  const [assignee, setAssignee] = React.useState(blueprint.assignee);
  const [createdBy] = React.useState(blueprint.createdBy);
  const [createdAt] = React.useState(blueprint.createdAt);

  // 戻るボタン
  const handleBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  // 保存ボタン（PageHeader / PageStyle 用）
  const handleSave = React.useCallback(() => {
    // 実際は TokenBlueprintCard 内の状態を集約して保存APIを叩く想定
    console.log("トークン設計を保存しました（モック）");
    alert("トークン設計を保存しました（モック）");
  }, []);

  return (
    <PageStyle
      layout="grid-2"
      title={`トークン設計：${blueprint.tokenBlueprintId}`}
      onBack={handleBack}
      onSave={handleSave} // ← PageHeader に保存ボタンを表示
    >
      {/* 左カラム：トークン設計カード＋コンテンツビューア */}
      <div>
        <TokenBlueprintCard initialEditMode />
        <div style={{ marginTop: 16 }}>
          <TokenContentsCard />
        </div>
      </div>

      {/* 右カラム：管理情報 */}
      <AdminCard
        title="管理情報"
        assigneeName={assignee}
        createdByName={createdBy}
        createdAt={createdAt}
        onEditAssignee={() => setAssignee("新担当者")}
        onClickAssignee={() => console.log("assignee clicked:", assignee)}
        onClickCreatedBy={() => console.log("createdBy clicked:", createdBy)}
      />
    </PageStyle>
  );
}
