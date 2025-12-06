// frontend/tokenOperation/src/presentation/pages/tokenOperationDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import TokenBlueprintCard from "../../../../tokenBlueprint/src/presentation/components/tokenBlueprintCard";
import TokenContentsCard from "../../../../tokenContents/src/presentation/components/tokenContentsCard";
import { TOKEN_BLUEPRINTS } from "../../../../tokenBlueprint/src/infrastructure/mockdata/tokenBlueprint_mockdata";
import { useTokenBlueprintCard } from "../../../../tokenBlueprint/src/presentation/hook/useTokenBlueprintCard";

export default function TokenOperationDetail() {
  const navigate = useNavigate();
  const { tokenOperationId } = useParams<{ tokenOperationId: string }>();

  // ─────────────────────────────────────────
  // モックデータ（トークン設計）
  // 本来は tokenOperationId と紐付いた TokenBlueprint を取得する想定
  // ─────────────────────────────────────────
  const blueprint = TOKEN_BLUEPRINTS[0];

  // TokenBlueprintCard 用 VM / handlers（モック blueprint から初期化）
  const { vm: cardVm, handlers: cardHandlers } = useTokenBlueprintCard({
    initialTokenBlueprint: (blueprint ?? {}) as any,
    initialBurnAt: "",
    initialIconUrl: blueprint?.iconId ?? "",
    initialEditMode: false,
  });

  // 管理情報（右カラム：モック）
  const [assignee, setAssignee] = React.useState("member_sato");
  const [creator] = React.useState("member_yamada");
  const [createdAt] = React.useState("2025-11-06T20:55:00Z"); // ISO8601 形式に寄せる

  // 戻る
  const onBack = React.useCallback(() => navigate(-1), [navigate]);

  // 保存ボタンのアクション（モック）
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
        {blueprint && (
          <TokenBlueprintCard vm={cardVm} handlers={cardHandlers} />
        )}

        <div style={{ marginTop: 16 }}>
          {/* TokenContentsCard: TokenBlueprint.contentFiles（ID配列）と連動させる想定 */}
          <TokenContentsCard
            mode="edit"
            images={blueprint?.contentFiles ?? []}
          />
        </div>
      </div>

      {/* 右カラム：管理情報 */}
      <AdminCard
        title="管理情報"
        assigneeName={assignee}
        createdByName={creator}
        createdAt={createdAt}
        onEditAssignee={() => setAssignee("member_new")}
        onClickAssignee={() => console.log("assignee clicked:", assignee)}
      />
    </PageStyle>
  );
}
