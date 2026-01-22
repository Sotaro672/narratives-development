// frontend/console/tokenOperation/src/presentation/pages/tokenOperationDetail.tsx
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import TokenBlueprintCard from "../../../../tokenBlueprint/src/presentation/components/tokenBlueprintCard";
import TokenContentsCard from "../../../../tokenBlueprint/src/presentation/components/tokenContentsCard";
import { useTokenOperationDetail } from "../hook/useTokenOperationDetail";

export default function TokenOperationDetail() {
  const {
    title,
    loading,
    error,
    blueprint,
    cardVm,
    cardHandlers,
    assignee,
    creator,
    createdAt,
    onBack,
    handleSave,
  } = useTokenOperationDetail();

  if (loading) {
    return (
      <PageStyle layout="grid-2" title={title} onBack={onBack}>
        <div>読み込み中です…</div>
      </PageStyle>
    );
  }

  if (error || !blueprint) {
    return (
      <PageStyle layout="grid-2" title={title} onBack={onBack}>
        <div>{error ?? "トークン設計が見つかりませんでした。"}</div>
      </PageStyle>
    );
  }

  /**
   * TokenContentsCard は images 互換を廃止したため、contents(GCSTokenContent[]) を渡す。
   *
   * 現時点では blueprint.contentFiles の型が環境によって
   * - string[]（ID 配列）
   * - ContentFile[]（{ id, name, type, url, size }）
   * の両方になり得るため、表示できるものだけ（url を持つ object）を抽出して渡す。
   *
   * NOTE:
   * - string[] のみの場合は、ここでは URL を復元できないので空表示になります。
   *   （将来的に「ID → TokenContents を List/Get して contents を構築」するのが正道です）
   */
  const contents = Array.isArray((blueprint as any)?.contentFiles)
    ? (blueprint as any).contentFiles
        .map((x: any) => {
          if (!x || typeof x !== "object") return null;
          const url = x.url != null ? String(x.url).trim() : "";
          if (!url) return null;

          return {
            id: String(x.id ?? "").trim(),
            name: String(x.name ?? "").trim(),
            type: String(x.type ?? "").trim(),
            url,
            size: Number(x.size ?? 0) || 0,
          };
        })
        .filter(Boolean)
    : [];

  return (
    <PageStyle layout="grid-2" title={title} onBack={onBack} onSave={handleSave}>
      {/* 左カラム：トークン設計＋コンテンツ */}
      <div>
        <TokenBlueprintCard vm={cardVm} handlers={cardHandlers} />

        <div style={{ marginTop: 16 }}>
          {/* TokenContentsCard: contents(GCSTokenContent[]) を渡す */}
          <TokenContentsCard mode="edit" contents={contents} />
        </div>
      </div>

      {/* 右カラム：管理情報 */}
      <AdminCard
        title="管理情報"
        assigneeName={assignee}
        createdByName={creator}
        createdAt={createdAt}
        onEditAssignee={undefined /* hook 側で必要になれば拡張 */}
        onClickAssignee={undefined /* hook 側で必要になれば拡張 */}
      />
    </PageStyle>
  );
}
