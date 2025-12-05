// frontend/console/tokenBlueprint/src/presentation/pages/tokenBlueprintDetail.tsx
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import TokenBlueprintCard from "../components/tokenBlueprintCard";
import TokenContentsCard from "../../../../tokenContents/src/presentation/components/tokenContentsCard";
import LogCard from "../../../../log/src/presentation/LogCard"; // ★ 追加

// ★ ロジックはすべて Hook に移譲
import { useTokenBlueprintDetail } from "../hook/useTokenBlueprintDetail";

export default function TokenBlueprintDetail() {
  const { vm, handlers } = useTokenBlueprintDetail();

  const {
    blueprint,
    // title は使わない（ID を表示しないため）
    assigneeName,
    createdByName,
    createdAt,
    tokenContentsIds,
    cardVm,
    isEditMode,
  } = vm;

  const {
    onBack,
    onEdit,
    onCancel,
    onSave,
    onDelete,
    onEditAssignee,
    onClickAssignee,
    cardHandlers,
  } = handlers;

  // データが無い場合のフォールバック
  if (!blueprint) {
    return (
      <PageStyle layout="single" title="トークン設計" onBack={onBack}>
        <p className="p-4 text-sm text-muted-foreground">
          表示可能なトークン設計がありません。
        </p>
      </PageStyle>
    );
  }

  return (
    <PageStyle
      layout="grid-2"
      title="トークン設計" // ★ tokenBlueprintId を含めない固定タイトル
      onBack={onBack}
      // ★ 通常時は「編集」ボタンのみ
      onEdit={!isEditMode ? onEdit : undefined}
      // ★ 編集モード時は「キャンセル／保存／削除」を表示
      onCancel={isEditMode ? onCancel : undefined}
      onSave={isEditMode ? onSave : undefined}
      onDelete={isEditMode ? onDelete : undefined}
    >
      {/* 左カラム：トークン設計カード＋コンテンツビューア */}
      <div>
        <TokenBlueprintCard vm={cardVm} handlers={cardHandlers} />

        <div style={{ marginTop: 16 }}>
          <TokenContentsCard images={tokenContentsIds} />
        </div>
      </div>

      {/* 右カラム：管理情報（TokenBlueprint のメタ情報）＋ログ */}
      <div className="space-y-4">
        <AdminCard
          title="管理情報"
          assigneeName={assigneeName}
          createdByName={createdByName}
          createdAt={createdAt}
          onEditAssignee={onEditAssignee}
          onClickAssignee={onClickAssignee}
        />

        {/* ★ 追加：ログカード（現状はログデータ未連携のためデフォルト表示） */}
        <LogCard title="更新ログ" />
      </div>
    </PageStyle>
  );
}
