// frontend/console/production/src/presentation/pages/productionDetail.tsx

import React from "react";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";

import ProductBlueprintCard from "../../../../productBlueprint/src/presentation/components/productBlueprintCard";
import ProductionQuantityCard from "../components/productionQuantityCard";
import type { ProductionQuantityRow } from "../../application/productionCreateService";

import { useProductionDetail } from "../hook/useProductionDetail";
import "../styles/production.css";

export default function ProductionDetail() {
  const { onBack, production, loading, error, creator } = useProductionDetail();

  // AdminCard 用の表示値
  const assigneeDisplay =
    production?.assigneeName ||
    production?.assigneeId ||
    "担当者が設定されていません";

  const createdAtLabel = production?.createdAt
    ? new Date(production.createdAt).toLocaleDateString("ja-JP")
    : "-";

  // ProductionQuantityCard 用: Production.models → ProductionQuantityRow[] にマッピング
  const quantityRows: ProductionQuantityRow[] = React.useMemo(() => {
    if (!production) return [];

    const rawModels = Array.isArray((production as any).models)
      ? ((production as any).models as any[])
      : [];

    return rawModels.map((m: any, idx: number): ProductionQuantityRow => ({
      // ★ 必須: modelVariationId
      modelVariationId:
        m.modelVariationId ?? m.id ?? m.variationId ?? `variation-${idx}`,

      // モデル番号 / 型番
      modelCode: m.modelCode ?? m.modelNumber ?? `#${idx + 1}`,

      // サイズ
      size: m.size ?? m.sizeLabel ?? "",

      // カラー名
      colorName: m.colorName ?? "",

      // カラーコード（例: "#ffffff" / "rgb(...)"）
      colorCode:
        m.colorCode ??
        (typeof m.colorRgb === "string" ? m.colorRgb : undefined),

      // 生産数
      stock: m.quantity ?? m.stock ?? 0,
    }));
  }, [production]);

  return (
    <PageStyle layout="grid-2" title="生産詳細" onBack={onBack}>
      {/* ========== 左カラム ========== */}
      <div className="space-y-4">
        {loading && (
          <div className="flex h-full items-center justify-center text-gray-500">
            生産情報を読み込み中です…
          </div>
        )}

        {!loading && error && (
          <div className="flex h-full items-center justify-center text-red-500">
            {error}
          </div>
        )}

        {!loading && !error && !production && (
          <div className="flex h-full items-center justify-center text-gray-500">
            対象の生産情報が見つかりません。
          </div>
        )}

        {!loading && !error && production && (
          <>
            {/* 商品設計カード（閲覧モード） */}
            <ProductBlueprintCard
              mode="view"
              productName={production.productBlueprintName ?? ""}
              // backend が brandName を返していればそれを使う（なければ空）
              brand={(production as any).brandName ?? ""}
            />

            {/* ★ モデル別 生産数一覧（閲覧モード） */}
            <ProductionQuantityCard
              title="モデル別 生産数一覧"
              rows={quantityRows}
              mode="view"
            />
          </>
        )}
      </div>

      {/* ========== 右カラム ========== */}
      <div className="space-y-4">
        <AdminCard
          title="管理情報"
          assigneeName={assigneeDisplay}
          assigneeCandidates={[]} // 詳細画面では編集しないので空
          loadingMembers={false}
          createdByName={creator}
          createdAt={createdAtLabel}
          // 詳細画面では担当者変更しないので no-op
          onSelectAssignee={() => {}}
        />
      </div>
    </PageStyle>
  );
}
