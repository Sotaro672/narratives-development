// frontend/tokenBlueprint/src/presentation/pages/tokenBlueprintCreate.tsx
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import TokenBlueprintCard from "../components/tokenBlueprintCard";
import TokenContentsCard from "../components/tokenContentsCard";

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
        // ★ Partial<TokenBlueprint> に存在しないフィールド（iconId 等）は入れない
        // ★ contentFiles は TokenBlueprint 型上は string[] 想定（shared/types/tokenBlueprint.ts に準拠）
        const input: Partial<TokenBlueprint> & { iconFile?: File | null } = {
          name: vm.name,
          symbol: vm.symbol,
          brandId: vm.brandId,
          description: vm.description,

          // TokenBlueprint の定義に合わせる（ID配列）
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
          {/* 方針A: 呼び出し側で onUploadClick を配線する */}
          <TokenContentsCard
            mode="edit"
            contents={[]}
            onUploadClick={() => {
              // ここは後続で「file picker → upload → contents 更新」に接続する想定。
              // 現段階では「押下できる」ことの確認用。
              // eslint-disable-next-line no-console
              console.log(
                "[TokenBlueprintCreate.page] TokenContentsCard upload clicked",
              );
              alert("ファイル追加（未接続）");
            }}
          />
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
