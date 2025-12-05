// frontend/tokenBlueprint/src/presentation/pages/tokenBlueprintCreate.tsx

import * as React from "react";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import TokenBlueprintCard from "../components/tokenBlueprintCard";
import TokenContentsCard from "../../../../tokenContents/src/presentation/components/tokenContentsCard";

import { useTokenBlueprintCreate } from "../hook/useTokenBlueprintCreate";

/**
 * トークン設計作成ページ（スタイルのみ保持）
 */
export default function TokenBlueprintCreate() {
  const {
    // --- useTokenBlueprintCreate から受け取る UI 用値・関数 ---
    initialTokenBlueprint,
    assigneeName,
    onEditAssignee,
    onClickAssignee,
    onBack,
    onSave,
  } = useTokenBlueprintCreate();

  return (
<PageStyle
  layout="grid-2"
  title="トークン設計を作成"
  onBack={onBack}
  onSave={() => onSave(initialTokenBlueprint)} // ← 引数なし関数として渡す

>
      {/* 左カラム：トークン設計フォーム */}
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
        mode="edit"
        assigneeName={assigneeName}
        onEditAssignee={onEditAssignee}
        onClickAssignee={onClickAssignee}
      />
    </PageStyle>
  );
}
