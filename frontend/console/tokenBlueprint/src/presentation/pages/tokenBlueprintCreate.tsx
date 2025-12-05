// frontend/tokenBlueprint/src/presentation/pages/tokenBlueprintCreate.tsx

import * as React from "react";
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
  } = useTokenBlueprintCreate();

  // TokenBlueprintCard 用の ViewModel / Handlers を構築
  const { vm, handlers } = useTokenBlueprintCard({
    initialTokenBlueprint,
    initialBurnAt: "", // 必要なら useTokenBlueprintCreate 側から渡す形に拡張
    initialIconUrl: undefined,
    initialEditMode: true,
  });

  return (
    <PageStyle
      layout="grid-2"
      title="トークン設計を作成"
      onBack={onBack}
      // PageStyle の onSave 型は () => void 想定なので、
      // 内部で onSave(Partial<TokenBlueprint>) を fire-and-forget で呼び出す
      onSave={() => {
        const input: Partial<TokenBlueprint> = {
          ...initialTokenBlueprint,
          name: vm.name,
          symbol: vm.symbol,
          brandId: vm.brandId,
          description: vm.description,
          // burnAt や iconId などを保存したい場合はここで追加
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
