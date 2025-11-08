// frontend/tokenBlueprint/src/pages/tokenBlueprintDetail.tsx
import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../admin/src/pages/AdminCard";
import TokenBlueprintCard from "./tokenBlueprintCard";

export default function TokenBlueprintDetail() {
  const navigate = useNavigate();
  const { tokenBlueprintId } = useParams<{ tokenBlueprintId: string }>();

  // 管理情報（右カラム用モック）
  const [assignee, setAssignee] = React.useState("佐藤 美咲");
  const [creator] = React.useState("佐藤 美咲");
  const [createdAt] = React.useState("2024/05/01");

  // 戻るボタン
  const handleBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  return (
    <PageStyle
      layout="grid-2"
      title={`トークン設計：${tokenBlueprintId ?? "token_blueprint_001"}`}
      onBack={handleBack}
    >
      {/* 左カラム：トークン設計カード */}
      <div>
        <TokenBlueprintCard />
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
