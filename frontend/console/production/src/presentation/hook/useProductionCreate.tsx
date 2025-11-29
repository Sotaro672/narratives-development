// frontend/console/production/src/presentation/hook/useProductionCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";

import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

// ★ アプリケーション層サービス
import {
  loadBrands,
  buildBrandOptions,
  loadProductBlueprints,
  filterProductBlueprintsByBrand,
  buildProductRows,
  loadDetailAndModels,
  buildSelectedForCard,
  loadAssigneeCandidates,
  buildAssigneeOptions,
  mapModelVariationsToRows,
} from "../../application/productionCreateService";

import type {
  Brand,
  ProductBlueprintManagementRow,
  Member,
  ProductBlueprintForCard,
  ModelVariationResponse,
} from "../../application/productionCreateService";

export function useProductionCreate() {
  const navigate = useNavigate();

  // ★ currentMember から fullName / companyId を利用
  const { currentMember } = useAuth();
  const creator = currentMember?.fullName ?? "-";
  const companyId = currentMember?.companyId?.trim() ?? "";

  // ==========================
  // 商品設計一覧 / 選択状態
  // ==========================
  const [allProductBlueprints, setAllProductBlueprints] =
    React.useState<ProductBlueprintManagementRow[]>([]);
  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [selectedBrand, setSelectedBrand] = React.useState<string | null>(null);

  // 選択中の商品設計の詳細 + ModelVariations
  const [selectedDetail, setSelectedDetail] = React.useState<any | null>(null);
  const [modelVariations, setModelVariations] = React.useState<
    ModelVariationResponse[]
  >([]);

  // Colors（現状はダミー。将来 API 連携予定）
  const [colors] = React.useState<string[]>([]);

  // ==========================
  // 管理情報
  // ==========================
  const [assignee, setAssignee] = React.useState("未設定");
  const [createdAt] = React.useState(() =>
    new Date().toLocaleDateString("ja-JP"),
  );

  const handleBack = React.useCallback(() => {
    navigate("/production");
  }, [navigate]);

  // ==========================
  // ブランド一覧
  // ==========================
  const [brands, setBrands] = React.useState<Brand[]>([]);

  React.useEffect(() => {
    loadBrands()
      .then((items) => setBrands(items))
      .catch(() => setBrands([]));
  }, []);

  const brandOptions = React.useMemo(
    () => buildBrandOptions(brands),
    [brands],
  );

  // ==========================
  // 商品設計一覧取得
  // ==========================
  React.useEffect(() => {
    loadProductBlueprints()
      .then((rows) => setAllProductBlueprints(rows))
      .catch(() => setAllProductBlueprints([]));
  }, []);

  // ブランドで商品設計を絞る
  const filteredBlueprints = React.useMemo(
    () => filterProductBlueprintsByBrand(allProductBlueprints, selectedBrand),
    [allProductBlueprints, selectedBrand],
  );

  const productRows = React.useMemo(
    () => buildProductRows(filteredBlueprints),
    [filteredBlueprints],
  );

  // 管理一覧上での選択行
  const selectedMgmtRow = React.useMemo(
    () => allProductBlueprints.find((pb) => pb.id === selectedId) ?? null,
    [allProductBlueprints, selectedId],
  );

  // ==========================
  // 詳細 + models の取得
  // ==========================
  React.useEffect(() => {
    if (!selectedId) {
      setSelectedDetail(null);
      setModelVariations([]);
      return;
    }

    (async () => {
      try {
        const { detail, models } = await loadDetailAndModels(selectedId);
        setSelectedDetail(detail);
        setModelVariations(models);
      } catch {
        setSelectedDetail(null);
        setModelVariations([]);
      }
    })();
  }, [selectedId]);

  // ProductBlueprintCard 用データ
  const selectedProductBlueprintForCard: ProductBlueprintForCard =
    React.useMemo(
      () => buildSelectedForCard(selectedDetail, selectedMgmtRow),
      [selectedDetail, selectedMgmtRow],
    );

  const hasSelectedProductBlueprint =
    selectedDetail != null || selectedMgmtRow != null;

  // ==========================
  // 担当者候補一覧
  // ==========================
  const [assigneeCandidates, setAssigneeCandidates] = React.useState<Member[]>(
    [],
  );
  const [loadingMembers, setLoadingMembers] = React.useState(false);

  React.useEffect(() => {
    if (!companyId) {
      setAssigneeCandidates([]);
      return;
    }

    (async () => {
      try {
        setLoadingMembers(true);
        const members = await loadAssigneeCandidates(companyId);
        setAssigneeCandidates(members);
      } catch {
        setAssigneeCandidates([]);
      } finally {
        setLoadingMembers(false);
      }
    })();
  }, [companyId]);

  const assigneeOptions = React.useMemo(
    () => buildAssigneeOptions(assigneeCandidates),
    [assigneeCandidates],
  );

  const handleSelectAssignee = React.useCallback(
    (id: string) => {
      const target = assigneeCandidates.find((m) => m.id === id);
      if (!target) return;

      const match = assigneeOptions.find((o) => o.id === id);
      const name = match?.name ?? target.email ?? target.id;
      setAssignee(name);
    },
    [assigneeCandidates, assigneeOptions],
  );

  // ==========================
  // ProductionQuantityCard 用 rows 変換
  // ==========================
  const modelVariationsForCard = React.useMemo(
    () => mapModelVariationsToRows(modelVariations),
    [modelVariations],
  );

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
      createdBy: creator,
      assignee,
      colors,
      modelVariations,
    });

    alert("生産計画を作成しました（ダミー）");
    navigate("/production");
  }, [navigate, selectedId, colors, creator, assignee, modelVariations]);

  return {
    // PageStyle 用
    onBack: handleBack,
    onSave: handleSave,

    // 左カラム
    hasSelectedProductBlueprint,
    selectedProductBlueprintForCard,

    // 管理カード用
    assignee,
    creator,
    createdAt,
    assigneeOptions,
    loadingMembers,
    onSelectAssignee: handleSelectAssignee,

    // ブランド選択用
    selectedBrand,
    brandOptions,
    selectBrand: setSelectedBrand,

    // 商品設計テーブル用
    productRows,
    selectedProductId: selectedId,
    selectProductById: setSelectedId,

    // ModelVariations とカード用 rows
    modelVariations,
    modelVariationsForCard,
  };
}
