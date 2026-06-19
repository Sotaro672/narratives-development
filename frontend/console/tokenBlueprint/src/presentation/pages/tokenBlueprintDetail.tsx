// frontend/console/tokenBlueprint/src/presentation/pages/tokenBlueprintDetail.tsx

import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import TokenBlueprintCard from "../components/tokenBlueprintCard";
import TokenContentsCard from "../components/tokenContentsCard";
import LogCard from "../../../../log/presentation/LogCard";

import { useTokenBlueprintDetail } from "../hook/useTokenBlueprintDetail";

export default function TokenBlueprintDetail() {
  const { vm, handlers } = useTokenBlueprintDetail();

  const {
    blueprint,
    assigneeId,
    assigneeName,
    minted,

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
    onSelectAssignee,
    onEditAssignee,
    onClickAssignee,
    cardHandlers,

    onTokenContentsFilesSelected,
    onDeleteTokenContent,
  } = handlers;

  if (!blueprint) {
    return (
      <PageStyle layout="single" title="トークン設計" onBack={onBack}>
        <p className="p-4 text-sm text-muted-foreground">
          表示可能なトークン設計がありません。
        </p>
      </PageStyle>
    );
  }

  const selectedIconFile: File | null = cardVm.iconFile ?? null;
  const pageTitle = blueprint.name || "トークン設計";

  return (
    <PageStyle
      layout="grid-2"
      title={pageTitle}
      onBack={onBack}
      onEdit={!isEditMode ? onEdit : undefined}
      onCancel={isEditMode ? onCancel : undefined}
      onSave={
        isEditMode
          ? () => {
              console.log("[TokenBlueprintDetail] onSave clicked", {
                id: blueprint.id,
                hasIconFile: Boolean(selectedIconFile),
                iconFile: selectedIconFile
                  ? {
                      name: selectedIconFile.name,
                      type: selectedIconFile.type,
                      size: selectedIconFile.size,
                    }
                  : null,
              });

              void onSave();
            }
          : undefined
      }
      onDelete={isEditMode && !minted ? onDelete : undefined}
    >
      <div>
        <TokenBlueprintCard
          vm={{
            ...cardVm,
            minted,
          }}
          handlers={cardHandlers}
        />

        <div style={{ marginTop: 16 }}>
          <TokenContentsCard
            mode={isEditMode ? "edit" : "view"}
            contents={tokenContents}
            onFilesSelected={onTokenContentsFilesSelected}
            onDelete={onDeleteTokenContent}
          />
        </div>
      </div>

      <div className="space-y-4">
        <AdminCard
          title="管理情報"
          mode={isEditMode ? "edit" : "view"}
          assigneeId={assigneeId}
          assigneeName={assigneeName}
          onSelectAssignee={onSelectAssignee}
          createdByName={createdByName}
          createdAt={createdAt}
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