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
    onSave, // (input: Partial<TokenBlueprint> & { iconFile?: File | null }) => Promise<void>
    initialEditMode, // ★ 追加
  } = useTokenBlueprintCreate();

  // TokenBlueprintCard 用の ViewModel / Handlers を構築
  // ★ selectedIconFile を受け取り、onSave に渡す
  const { vm, handlers, selectedIconFile } = useTokenBlueprintCard({
    initialTokenBlueprint,
    initialBurnAt: "",
    initialIconUrl: undefined,
    initialEditMode, // ★ edit/view モード切替を受け渡し
  });

  return (
    <PageStyle
      layout="grid-2"
      title="トークン設計を作成"
      onBack={onBack}
      onSave={() => {
        // ★ initialTokenBlueprint は上書きしない
        // ★ iconFile を一緒に渡す（ここが抜けていたため service に届かなかった）
        const input: Partial<TokenBlueprint> & { iconFile?: File | null } = {
          name: vm.name,
          symbol: vm.symbol,
          brandId: vm.brandId,
          description: vm.description,
          iconId: null,
          contentFiles: [],

          // ★ 追加: hook が保持している File を渡す
          iconFile: selectedIconFile ?? null,
        };

        // eslint-disable-next-line no-console
        console.log("[TokenBlueprintCreate.page] onSave input:", {
          name: input.name,
          symbol: input.symbol,
          brandId: input.brandId,
          hasIconFile: Boolean(input.iconFile),
          iconFile: input.iconFile
            ? {
                name: input.iconFile.name,
                type: input.iconFile.type,
                size: input.iconFile.size,
              }
            : null,
        });

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
