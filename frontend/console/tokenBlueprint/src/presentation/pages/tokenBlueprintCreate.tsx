// frontend/tokenBlueprint/src/presentation/pages/tokenBlueprintCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import TokenBlueprintCard from "../components/tokenBlueprintCard";
import TokenContentsCard from "../../../../tokenContents/src/presentation/components/tokenContentsCard";
import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";

// ★ companyId を currentMember から取得（モックの場合は固定値でも可）
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

/**
 * トークン設計作成ページ
 */
export default function TokenBlueprintCreate() {
  const navigate = useNavigate();

  // ★ currentMember から companyId を取得
  const { currentMember } = useAuth();
  const companyId = currentMember?.companyId ?? "company-mock-001";

  // 管理情報（新規作成時のメタ情報をモック）
  const [assignee, setAssignee] = React.useState("assignee-001");
  const [createdBy] = React.useState("creator-001");
  const [createdAt] = React.useState("2024-01-20T00:00:00Z");

// 戻るボタン（絶対パスで TokenBlueprintManagement に戻る）
const handleBack = React.useCallback(() => {
  navigate("/token-blueprint", { replace: true });
}, [navigate]);

  // 保存ボタン（実際はフォーム内容を収集して API へ送信する想定）
  const handleSave = React.useCallback(() => {
    console.log("トークン設計を作成しました（モック）");
    alert("トークン設計を作成しました（モック）");
  }, []);

  // ★ companyId を追加した新規作成時の TokenBlueprint 初期値
  const initialTokenBlueprint: Partial<TokenBlueprint> = {
    id: "", // 新規のため未採番
    name: "",
    symbol: "",
    brandId: "",
    description: "",
    companyId, // ★ 追加（必須）
    iconId: null,
    contentFiles: [],
    assigneeId: assignee,
    createdBy,
    createdAt,
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
        <TokenBlueprintCard
          initialEditMode
          initialTokenBlueprint={initialTokenBlueprint}
        />

        <div style={{ marginTop: 16 }}>
          <TokenContentsCard images={[]} />
        </div>
      </div>

      {/* 右カラム：管理情報 */}
      <AdminCard
        title="管理情報"
        assigneeName={assignee}
        onEditAssignee={() => setAssignee("new-assignee-id")}
        onClickAssignee={() => console.log("assignee clicked:", assignee)}
      />
    </PageStyle>
  );
}
