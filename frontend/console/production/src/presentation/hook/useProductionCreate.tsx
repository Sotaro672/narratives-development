// frontend/console/production/src/presentation/hook/useProductionCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import { fetchAllBrandsForCompany } from "../../../../brand/src/infrastructure/query/brandQuery";
import type { Brand } from "../../../../brand/src/domain/entity/brand";

type ProductBlueprint = {
  id: string;
  name: string;
  brand?: string;
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
  // カラー（APIで後で埋める）
  // ==========================
  const [colors] = React.useState<string[]>([]);

  // ==========================
  // 管理情報
  // ==========================
  const [assignee, setAssignee] = React.useState("未設定");
  const [creator] = React.useState("現在のユーザー");
  const [createdAt] = React.useState(() =>
    new Date().toLocaleDateString("ja-JP"),
  );

  const handleBack = React.useCallback(() => {
    navigate("/production");
  }, [navigate]);

  // ==========================
  // ブランド一覧（API）
  // ==========================
  const [brands, setBrands] = React.useState<Brand[]>([]);

  React.useEffect(() => {
    // companyId は互換用ダミー。実際の絞り込みは backend 側の context.companyId で行われる
    fetchAllBrandsForCompany("", true)
      .then((items) => setBrands(items))
      .catch((e) => {
        console.error("ブランド取得失敗:", e);
        setBrands([]);
      });
  }, []);

  const brandOptions = React.useMemo(
    () => brands.map((b) => b.name).filter(Boolean),
    [brands],
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
    [selectedId, productBlueprints],
  );

  const selectedForCard: ProductBlueprint =
    selected ??
    ({
      id: "",
      name: "",
      brand: "",
    } as ProductBlueprint);

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
    });

    alert("生産計画を作成しました（ダミー）");
    navigate("/production");
  }, [navigate, selectedId, colors]);

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
