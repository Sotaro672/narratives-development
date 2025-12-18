// frontend/console/inventory/src/presentation/pages/inventoryDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import ProductBlueprintCard from "../../../../productBlueprint/src/presentation/components/productBlueprintCard";
import InventoryCard from "../components/inventoryCard";

// ✅ 追加：TokenBlueprintCard
import TokenBlueprintCard, {
  type TokenBlueprintCardViewModel,
  type TokenBlueprintCardHandlers,
} from "../../../../tokenBlueprint/src/presentation/components/tokenBlueprintCard";

import { useInventoryDetail } from "../hook/useInventoryDetail";

export default function InventoryDetail() {
  const navigate = useNavigate();

  // ✅ 方針A: URL で pbId + tbId を受け取る
  const { productBlueprintId, tokenBlueprintId } = useParams<{
    productBlueprintId?: string;
    tokenBlueprintId?: string;
  }>();

  /**
   * ★ pbId/tbId が無い（＝ /inventory/detail だけ or 旧ルートに誤マッチ）
   *    → 一覧ページへ強制リダイレクト
   */
  React.useEffect(() => {
    if (!productBlueprintId || !tokenBlueprintId) {
      console.warn(
        "[inventory/InventoryDetail] missing productBlueprintId or tokenBlueprintId -> redirect",
        { productBlueprintId, tokenBlueprintId },
      );
      navigate("/inventory", { replace: true });
    }
  }, [productBlueprintId, tokenBlueprintId, navigate]);

  // ★ 戻るボタンも常に一覧へ戻す
  const onBack = React.useCallback(() => {
    navigate("/inventory");
  }, [navigate]);

  // ✅ hook（方針A）: pbId + tbId -> inventoryIds -> details -> merge
  const { rows, loading, error, vm } = useInventoryDetail(
    productBlueprintId,
    tokenBlueprintId,
  );

  const pbId = (vm?.productBlueprintId ?? productBlueprintId ?? "").trim();
  const tbId = (vm?.tokenBlueprintId ?? tokenBlueprintId ?? "").trim();
  const pbPatch = vm?.productBlueprintPatch;

  const title = pbPatch?.productName
    ? `在庫詳細：${pbPatch.productName}`
    : vm
      ? `在庫詳細：${vm.productBlueprintId} / ${vm.tokenBlueprintId}`
      : `在庫詳細：${productBlueprintId ?? ""} / ${tokenBlueprintId ?? ""}`;

  // ============================================================
  // ✅ TokenBlueprintCard (view only) 用の VM/Handlers を組み立て
  //  - useInventoryDetail が tokenBlueprintPatch を返していればそれを利用
  //  - 無い場合でも型を満たすように fallback を入れて表示だけできるようにする
  // ============================================================

  const tbPatchAny = (vm as any)?.tokenBlueprintPatch ?? (vm as any)?.tokenBlueprint ?? null;

  const tokenCardVM: TokenBlueprintCardViewModel = React.useMemo(
    () => ({
      id: tbId,
      name: String(tbPatchAny?.name ?? ""),
      symbol: String(tbPatchAny?.symbol ?? ""),
      brandId: String(tbPatchAny?.brandId ?? ""),
      brandName: String(tbPatchAny?.brandName ?? ""),
      description: String(tbPatchAny?.description ?? ""),
      iconUrl: tbPatchAny?.iconUrl ?? tbPatchAny?.iconURL ?? undefined,

      // ここは「在庫詳細では基本 view」運用に寄せるなら false 固定でもOK
      // （minted:true だとアイコン編集UIが出るため）
      minted: false,

      iconFile: null,
      isEditMode: false,
      brandOptions: [],
    }),
    [tbId, tbPatchAny],
  );

  const tokenCardHandlers: TokenBlueprintCardHandlers = React.useMemo(
    () => ({
      // 在庫詳細では view-only 想定：プレビューは軽い動作だけ入れておく
      onPreview: () => {
        const url = tokenCardVM.iconUrl;
        if (url) window.open(url, "_blank", "noopener,noreferrer");
      },
    }),
    [tokenCardVM.iconUrl],
  );

  return (
    <PageStyle layout="grid-2" title={title} onBack={onBack} onSave={undefined}>
      {/* 左カラム：商品情報カード + デバッグ情報 + 在庫一覧カード */}
      <div>
        {/* ✅ ProductBlueprintCardProps に productBlueprintId が無いので渡さない */}
        {/* ✅ inventory 側で取れた patch をそのまま渡す */}
        <ProductBlueprintCard
          mode="view"
          productBlueprintPatch={vm?.productBlueprintPatch}
        />

        {/* ✅ 追加：ProductBlueprintCard の直下に TokenBlueprintCard を表示 */}
        {tbId && (
          <div className="mt-3">
            <TokenBlueprintCard vm={tokenCardVM} handlers={tokenCardHandlers} />
          </div>
        )}

        {/* --- hook 取得データの可視化（確認用） --- */}
        <div className="mt-3 rounded-md border border-[hsl(var(--border))] bg-[hsl(var(--card))] p-3">
          <div className="text-sm font-semibold">取得データ（hook）</div>

          <div className="mt-2 text-xs text-[hsl(var(--muted-foreground))] space-y-1">
            <div>
              <span className="font-medium">productBlueprintId:</span>{" "}
              {pbId || "-"}
            </div>
            <div>
              <span className="font-medium">tokenBlueprintId:</span> {tbId || "-"}
            </div>
            <div>
              <span className="font-medium">inventoryKey:</span>{" "}
              {vm?.inventoryKey ?? "-"}
            </div>
            <div>
              <span className="font-medium">inventoryIds:</span>{" "}
              {Array.isArray(vm?.inventoryIds) ? vm!.inventoryIds.length : 0}
            </div>
            {Array.isArray(vm?.inventoryIds) && vm!.inventoryIds.length > 0 && (
              <div className="break-all">
                <span className="font-medium">inventoryIds(sample):</span>{" "}
                {vm!.inventoryIds.slice(0, 10).join(", ")}
                {vm!.inventoryIds.length > 10 ? " ..." : ""}
              </div>
            )}
            <div>
              <span className="font-medium">totalStock:</span>{" "}
              {vm?.totalStock ?? 0}
            </div>
            <div>
              <span className="font-medium">rows:</span>{" "}
              {Array.isArray(rows) ? rows.length : 0}
            </div>
            <div>
              <span className="font-medium">updatedAt:</span>{" "}
              {vm?.updatedAt ?? "-"}
            </div>

            {/* ✅ Patch が取れているか可視化 */}
            <div className="pt-2">
              <span className="font-medium">productBlueprintPatch:</span>{" "}
              {pbPatch ? "✅ ok" : "❌ none"}
            </div>
            {pbPatch && (
              <div className="break-all">
                <span className="font-medium">patchKeys:</span>{" "}
                {Object.keys(pbPatch as any).join(", ")}
              </div>
            )}
          </div>
        </div>
        {/* --- /hook 取得データの可視化 --- */}

        {/* --- style elements only --- */}
        {loading && (
          <div className="text-sm text-[hsl(var(--muted-foreground))] mt-2">
            読み込み中...
          </div>
        )}

        {error && (
          <div className="text-sm text-red-600 mt-2">
            読み込みに失敗しました: {error}
          </div>
        )}
        {/* --- style elements only --- */}

        <InventoryCard rows={rows} />
      </div>

      {/* 右カラム：削除（空要素を置いて grid-2 を維持） */}
      <div />
    </PageStyle>
  );
}
