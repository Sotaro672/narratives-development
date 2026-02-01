// frontend/console/inventory/src/presentation/pages/inventoryDetail.tsx

import * as React from "react";
import { useNavigate, useParams } from "react-router-dom";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import ProductBlueprintCard from "../../../../productBlueprint/src/presentation/components/productBlueprintCard";
import InventoryCard from "../components/inventoryCard";

// ✅ TokenBlueprintCard（view-only）
import TokenBlueprintCard, {
  type TokenBlueprintCardViewModel,
  type TokenBlueprintCardHandlers,
} from "../../../../tokenBlueprint/src/presentation/components/tokenBlueprintCard";

import { useInventoryDetail } from "../hook/useInventoryDetail";

function s(v: unknown): string {
  return String(v ?? "").trim();
}

const SEP = "__";

function splitInventoryId(inventoryId: string): { pbId: string; tbId: string } | null {
  const id = s(inventoryId);
  if (!id || !id.includes(SEP)) return null;

  const parts = id.split(SEP);
  const pbId = s(parts[0]);
  const tbId = s(parts[1]);

  if (!pbId || !tbId) return null;
  return { pbId, tbId };
}

export default function InventoryDetail() {
  const navigate = useNavigate();

  // ✅ 新方針: URL は /inventory/detail/:inventoryId （docId = pb__tb）だけ受け取る
  const { inventoryId: inventoryIdParam } = useParams<{ inventoryId?: string }>();
  const inventoryId = s(inventoryIdParam);

  // ✅ inventoryId が無い（＝ /inventory/detail だけ or 旧ルートに誤マッチ）
  //    → 一覧ページへ強制リダイレクト
  React.useEffect(() => {
    if (!inventoryId) {
      navigate("/inventory", { replace: true });
    }
  }, [inventoryId, navigate]);

  // ★ 戻るボタンは常に一覧へ戻す
  const onBack = React.useCallback(() => {
    navigate("/inventory");
  }, [navigate]);

  // ✅ 互換: 既存 hook が pbId/tbId 前提のため、inventoryId(pb__tb) から split して渡す
  const split = React.useMemo(() => splitInventoryId(inventoryId), [inventoryId]);
  const pbId = s(split?.pbId);
  const tbId = s(split?.tbId);

  // split できない inventoryId は不正なので一覧へ戻す
  React.useEffect(() => {
    if (inventoryId && (!pbId || !tbId)) {
      navigate("/inventory", { replace: true });
    }
  }, [inventoryId, pbId, tbId, navigate]);

  // ✅ hook（内部 fetch は pbId/tbId を利用）
  // NOTE: hook 側のシグネチャが (pbId, tbId) のままならこれでOK
  const { rows, loading, error, vm } = useInventoryDetail(pbId, tbId);

  const pbPatch = vm?.productBlueprintPatch;

  // ✅ タイトル（ヘッダーは useInventoryDetail 側で `${productName} / ${tokenName}` を返す前提）
  const title = s(vm?.headerTitle)
    ? `在庫詳細：${vm!.headerTitle}`
    : pbPatch?.productName
      ? `在庫詳細：${pbPatch.productName}`
      : vm
        ? `在庫詳細：${vm.productBlueprintId} / ${vm.tokenBlueprintId}`
        : inventoryId
          ? `在庫詳細：${inventoryId}`
          : "在庫詳細";

  // ✅ 出品ボタン: /inventory/list/create/:inventoryId へ（docId をそのまま使う）
  const onList = React.useCallback(() => {
    if (!inventoryId) return;
    navigate(`/inventory/list/create/${encodeURIComponent(inventoryId)}`);
  }, [navigate, inventoryId]);

  // ============================================================
  // ✅ TokenBlueprintCard (view only)
  // ============================================================
  const tbPatch = vm?.tokenBlueprintPatch;

  const tokenCardVM: TokenBlueprintCardViewModel = React.useMemo(() => {
    const tokenName = s(tbPatch?.tokenName);
    const symbol = s(tbPatch?.symbol);
    const brandName = s(tbPatch?.brandName);
    const description = String(tbPatch?.description ?? "");
    const iconUrl = s(tbPatch?.iconUrl) || undefined;

    // ✅ brandId は不要方針：TokenBlueprintCard 側が必須なら空文字で埋める
    const brandId = "";

    // minted は在庫詳細では view-only に寄せる（trueにすると編集UIが出るため）
    const minted = false;

    return {
      id: tbId,
      name: tokenName || tbId || "-",
      symbol,
      brandId,
      brandName,
      description,
      iconUrl,

      minted,
      iconFile: null,
      isEditMode: false,
      brandOptions: [],
    };
  }, [tbId, tbPatch]);

  const tokenCardHandlers: TokenBlueprintCardHandlers = React.useMemo(
    () => ({
      onPreview: () => {
        const url = tokenCardVM.iconUrl;
        if (url) window.open(url, "_blank", "noopener,noreferrer");
      },
    }),
    [tokenCardVM.iconUrl],
  );

  return (
    <PageStyle
      layout="grid-2"
      title={title}
      onBack={onBack}
      onSave={undefined}
      onList={onList}
    >
      {/* 左カラム：商品情報カード + TokenBlueprintCard + 在庫一覧カード */}
      <div>
        <ProductBlueprintCard
          mode="view"
          productBlueprintPatch={vm?.productBlueprintPatch}
        />

        {/* TokenBlueprintCard */}
        {tbId && (
          <div className="mt-3">
            <TokenBlueprintCard vm={tokenCardVM} handlers={tokenCardHandlers} />
          </div>
        )}

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

        <InventoryCard rows={rows} />
      </div>

      {/* 右カラム：空要素（grid-2維持） */}
      <div />
    </PageStyle>
  );
}
