// frontend/tokenBlueprint/src/presentation/pages/tokenBlueprintCreate.tsx
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import TokenBlueprintCard from "../components/tokenBlueprintCard";
import TokenContentsCard from "../../../../tokenContents/src/presentation/components/tokenContentsCard";

import { useTokenBlueprintCreate } from "../hook/useTokenBlueprintCreate";
import { useTokenBlueprintCard } from "../hook/useTokenBlueprintCard";
import type { TokenBlueprint } from "../../domain/entity/tokenBlueprint";

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
    onSave, // (input: Partial<TokenBlueprint>) => Promise<void> を想定
    initialEditMode, // ★追加：edit モード制御
  } = useTokenBlueprintCreate();

  // TokenBlueprintCard 用の ViewModel / Handlers を構築
  const { vm, handlers } = useTokenBlueprintCard({
    initialTokenBlueprint,
    initialBurnAt: "",
    initialIconUrl: undefined,
    initialEditMode, // ★ useTokenBlueprintCreate から渡す
  });

  return (
    <PageStyle
      layout="grid-2"
      title="トークン設計を作成"
      onBack={onBack}
      onSave={() => {
        const input: Partial<TokenBlueprint> = {
          ...initialTokenBlueprint,
          name: vm.name,
          symbol: vm.symbol,
          brandId: vm.brandId,
          description: vm.description,
          // burnAt や iconId を保存したい場合はここへ追加
        };

        void onSave(input);
      }}
    >
      {/* 左カラム：トークン設計フォーム */}
      <div>
        <TokenBlueprintCard vm={vm} handlers={handlers} />

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
