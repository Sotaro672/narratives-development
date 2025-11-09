// frontend/tokenBlueprint/src/presentation/pages/tokenBlueprintDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import TokenBlueprintCard from "../components/tokenBlueprintCard";
import TokenContentsCard from "../../../../tokenContents/src/presentation/components/tokenContentsCard";
import { TOKEN_BLUEPRINTS } from "../../infrastructure/mockdata/mockdata";
import type { TokenBlueprint } from "../../../../shell/src/shared/types/tokenBlueprint";

export default function TokenBlueprintDetail() {
  const navigate = useNavigate();
  const { tokenBlueprintId } = useParams<{ tokenBlueprintId: string }>();

  // 対象の TokenBlueprint をモックから取得（id で検索）
  const blueprint: TokenBlueprint | undefined = React.useMemo(() => {
    if (!TOKEN_BLUEPRINTS.length) return undefined;
    if (tokenBlueprintId) {
      const found = TOKEN_BLUEPRINTS.find(
        (b) => b.id === tokenBlueprintId,
      );
      if (found) return found;
    }
    // パラメータ不一致時は先頭をフォールバック表示（モック用）
    return TOKEN_BLUEPRINTS[0];
  }, [tokenBlueprintId]);

  const handleBack = React.useCallback(() => {
    navigate(-1);
  }, [navigate]);

  const handleSave = React.useCallback(() => {
    // TODO: TokenBlueprintCard の状態を集約して保存 API を呼ぶ
    console.log("トークン設計を保存しました（モック）");
    alert("トークン設計を保存しました（モック）");
  }, []);

  // モックが無い場合の簡易フォールバック
  if (!blueprint) {
    return (
      <PageStyle
        layout="single"
        title="トークン設計"
        onBack={handleBack}
      >
        <p className="p-4 text-sm text-muted-foreground">
          表示可能なトークン設計がありません（モックデータ未定義）。
        </p>
      </PageStyle>
    );
  }

  // 管理情報表示用（ID をそのまま表示しておく：実運用では名前解決する想定）
  const [assignee, setAssignee] = React.useState(blueprint.assigneeId);
  const [createdBy] = React.useState(blueprint.createdBy);
  const [createdAt] = React.useState(blueprint.createdAt);

  return (
    <PageStyle
      layout="grid-2"
      title={`トークン設計：${blueprint.id}`}
      onBack={handleBack}
      onSave={handleSave}
    >
      {/* 左カラム：トークン設計カード＋コンテンツビューア */}
      <div>
        {/* TokenBlueprintCard は内部で必要な情報を取得するモック想定 */}
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
        onEditAssignee={() => setAssignee("new-assignee-id")}
        onClickAssignee={() => console.log("assignee clicked:", assignee)}
        onClickCreatedBy={() => console.log("createdBy clicked:", createdBy)}
      />
    </PageStyle>
  );
}
