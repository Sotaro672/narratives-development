// frontend/console/production/src/presentation/hook/useProductionCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import type { SizeRow } from "../../../../model/src/domain/entity/catalog";

type ProductBlueprint = {
  id: string;
  name: string;
  brand?: string;
  description?: string;
};

export function useProductionCreate() {
  const navigate = useNavigate();

  // ==========================
  // 商品設計一覧（後で API 連携）
  // ==========================
  const [productBlueprints] = React.useState<ProductBlueprint[]>([]);

  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [selectedBrand, setSelectedBrand] = React.useState<string | null>(null);

  // ==========================
  // サイズ・カラー（APIで後で埋める）
  // ==========================
  const [colors] = React.useState<string[]>([]);
  const [sizes] = React.useState<SizeRow[]>([]);

  // ==========================
  // 管理情報
  // ==========================
  const [assignee, setAssignee] = React.useState("未設定");
  const [creator] = React.useState("現在のユーザー");
  const [createdAt] = React.useState(() =>
    new Date().toLocaleDateString("ja-JP")
  );

  const handleBack = React.useCallback(() => {
    navigate("/production");
  }, [navigate]);

  // ==========================
  // ブランド一覧
  // ==========================
  const brandOptions = React.useMemo(
    () =>
      Array.from(
        new Set(
          productBlueprints
            .map((p) => p.brand?.trim())
            .filter((b): b is string => !!b)
        )
      ),
    [productBlueprints]
  );

  // ==========================
  // ブランドフィルタ後の商品設計一覧
  // ==========================
  const filteredBlueprints = React.useMemo(() => {
    if (!selectedBrand) return productBlueprints;
    return productBlueprints.filter((p) => p.brand === selectedBrand);
  }, [productBlueprints, selectedBrand]);

  // ==========================
  // 選択中商品設計
  // ==========================
  const selected = React.useMemo(
    () => productBlueprints.find((p) => p.id === selectedId) ?? null,
    [selectedId, productBlueprints]
  );

  const selectedForCard: any =
    selected ??
    ({
      id: "",
      name: "",
      brand: "",
      description: "",
    } as any);

  const hasSelected = selected != null;

  // ==========================
  // 保存（ダミー）
  // ==========================
  const handleSave = React.useCallback(() => {
    if (!selectedId) {
      alert("商品設計を選択してください");
      return;
    }

    console.log("生産計画作成:", {
      productBlueprintId: selectedId,
      colors,
      sizes,
    });

    alert("生産計画を作成しました（ダミー）");
    navigate("/production");
  }, [navigate, selectedId, colors, sizes]);

  return {
    // PageStyle 用
    onBack: handleBack,
    onSave: handleSave,

    // 左カラム
    hasSelectedProductBlueprint: hasSelected,
    selectedProductBlueprintForCard: selectedForCard,

    // 管理カード用
    assignee,
    creator,
    createdAt,
    setAssignee,

    // ブランド選択用
    selectedBrand,
    brandOptions,
    selectBrand: setSelectedBrand,

    // 商品設計テーブル用
    productRows: filteredBlueprints,
    selectedProductId: selectedId,
    selectProductById: setSelectedId,
  };
}
