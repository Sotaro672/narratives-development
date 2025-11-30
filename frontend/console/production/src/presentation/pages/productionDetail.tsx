// frontend/console/production/src/presentation/pages/productionDetail.tsx

import React from "react";
import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";

import ProductBlueprintCard from "../../../../productBlueprint/src/presentation/components/productBlueprintCard";
import ProductionQuantityCard from "../components/productionQuantityCard";

import { useProductionDetail } from "../hook/useProductionDetail";
import "../styles/production.css";

// ProductBlueprintCard の型用
import type {
  ItemType,
  Fit,
} from "../../../../productBlueprint/src/domain/entity/catalog";

export default function ProductionDetail() {
  const {
    onBack,
    production,
    loading,
    error,
    creator,
    quantityRows,      // ★ Hook から受け取る（モデル別生産数）
    productBlueprint,  // ★ Hook から受け取る（商品設計の全データ）
    pbLoading,
    pbError,
  } = useProductionDetail();

  // AdminCard 用の表示値
  const assigneeDisplay =
    production?.assigneeName ||
    production?.assigneeId ||
    "担当者が設定されていません";

  const createdAtLabel = production?.createdAt
    ? new Date(production.createdAt).toLocaleDateString("ja-JP")
    : "-";

  return (
    <PageStyle layout="grid-2" title="生産詳細" onBack={onBack}>
      {/* ========== 左カラム ========== */}
      <div className="space-y-4">
        {/* Production 読み込み中 */}
        {loading && (
          <div className="flex h-full items-center justify-center text-gray-500">
            生産情報を読み込み中です…
          </div>
        )}

        {/* Production 読み込みエラー */}
        {!loading && error && (
          <div className="flex h-full items-center justify-center text-red-500">
            {error}
          </div>
        )}

        {/* 該当なし */}
        {!loading && !error && !production && (
          <div className="flex h-full items-center justify-center text-gray-500">
            対象の生産情報が見つかりません。
          </div>
        )}

        {/* --- Production が取得できているとき --- */}
        {!loading && !error && production && (
          <>
            {/* ===== 商品設計カード（閲覧モード） ===== */}
            {pbLoading && (
              <div className="p-4 text-gray-500">
                商品設計を読み込み中…
              </div>
            )}

            {!pbLoading && pbError && (
              <div className="p-4 text-red-500">{pbError}</div>
            )}

            {!pbLoading && !pbError && productBlueprint && (
              <ProductBlueprintCard
                mode="view"
                productName={productBlueprint.productName}
                // ブランド名は production 側で解決済みのものを表示
                brand={production.brandName ?? ""}
                brandId={productBlueprint.brandId}
                // 型は string だが、catalog 側の union と互換想定なので as で合わせる
                itemType={productBlueprint.itemType as ItemType}
                fit={productBlueprint.fit as Fit}
                materials={productBlueprint.material}
                weight={productBlueprint.weight}
                washTags={productBlueprint.qualityAssurance}
                productIdTag={productBlueprint.productIdTag}
              />
            )}

            {/* ===== モデル別 生産数一覧（閲覧モード） ===== */}
            <ProductionQuantityCard
              title="モデル別 生産数一覧"
              rows={quantityRows} // ★ Hook から渡された rows をそのまま使用
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
