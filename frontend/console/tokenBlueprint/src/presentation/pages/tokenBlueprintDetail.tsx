import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import TokenBlueprintCard from "../components/tokenBlueprintCard";
import TokenContentsCard from "../components/tokenContentsCard";
import LogCard from "../../../../log/src/presentation/LogCard";

// ★ ロジックはすべて Hook に移譲
import { useTokenBlueprintDetail } from "../hook/useTokenBlueprintDetail";

export default function TokenBlueprintDetail() {
  const { vm, handlers } = useTokenBlueprintDetail();

  const {
    blueprint,
    assigneeName,

    // ★ 追加: 管理情報
    createdByName,
    createdAt,
    updatedByName,
    updatedAt,

    tokenContents,
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

    // ★ token-contents: upload / delete handlers
    onTokenContentsFilesSelected,
    onDeleteTokenContent,
  } = handlers;

  // ★ A案：単一ソースは cardVm.iconFile
  // useTokenBlueprintDetail が selectedIconFile を vm に載せる実装でも壊れないようにフォールバックも残す
  const selectedIconFile: File | null =
    (cardVm?.iconFile as File | null | undefined) ??
    (((vm as any)?.selectedIconFile as File | null) ?? null);

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
      title="トークン設計"
      onBack={onBack}
      onEdit={!isEditMode ? onEdit : undefined}
      onCancel={isEditMode ? onCancel : undefined}
      onSave={
        isEditMode
          ? () => {
              // eslint-disable-next-line no-console
              console.log("[TokenBlueprintDetail] onSave clicked", {
                id: (blueprint as any)?.id,
                hasIconFile: Boolean(selectedIconFile),
                iconFile: selectedIconFile
                  ? {
                      name: selectedIconFile.name,
                      type: selectedIconFile.type,
                      size: selectedIconFile.size,
                    }
                  : null,
              });

              // hook 側が引数を取る実装でも / 取らない実装でも動くように any 呼び出し
              void (onSave as any)({ iconFile: selectedIconFile });
            }
          : undefined
      }
      onDelete={isEditMode ? onDelete : undefined}
    >
      {/* 左カラム：トークン設計カード＋コンテンツビューア */}
      <div>
        <TokenBlueprintCard vm={cardVm} handlers={cardHandlers} />

        <div style={{ marginTop: 16 }}>
          {/* ★ upload / delete を Hook の handler に接続 */}
          <TokenContentsCard
            mode={isEditMode ? "edit" : "view"}
            contents={tokenContents}
            onFilesSelected={onTokenContentsFilesSelected}
            onDelete={onDeleteTokenContent}
          />
        </div>
      </div>

      {/* 右カラム：管理情報＋ログ */}
      <div className="space-y-4">
        <AdminCard
          title="管理情報"
          assigneeName={assigneeName}
          createdByName={createdByName}
          createdAt={createdAt}
          // ★ 追加
          updatedByName={updatedByName}
          updatedAt={updatedAt}
          onEditAssignee={onEditAssignee}
          onClickAssignee={onClickAssignee}
        />

        <LogCard title="更新ログ" />
      </div>
    </PageStyle>
  );
}
