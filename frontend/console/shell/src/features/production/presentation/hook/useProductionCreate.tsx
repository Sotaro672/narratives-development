// frontend/console/shell/src/features/production/presentation/hook/useProductionCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";

import { useAuthContext } from "../../../../auth/application/AuthContext";

import {
  loadBrands,
  loadProductBlueprints,
  loadDetailAndModels,
  type ProductBlueprintDetailResponse,
} from "../../infrastructure/api/productionCreateApi";

import { buildModelIndexFromVariations } from "../../application/detail/buildModelVariationIndex";
import type { ModelVariationSummary } from "../../application/detail/types";

import {
  buildBrandOptions,
  filterProductBlueprintsByBrand,
  buildProductRows,
} from "../create/mappers";

import type { Brand } from "../../../../shared/types/brand";
import type { ProductBlueprintManagementRow } from "../../../productBlueprint/infrastructure/query/productBlueprintQuery";
import type { ModelVariationResponse } from "../../../productBlueprint/application/productBlueprintDetailService";

import { buildProductionPayload } from "../../application/create/ProductionCreateService";
import { ProductionRepositoryHTTP } from "../../infrastructure/http/productionRepositoryHTTP";

import type { ProductionQuantityRowVM } from "../viewModels/productionQuantityRowVM";
import { buildProductionQuantityRowVMs } from "../viewModels/buildProductionQuantityRowVMs";

import { useAssigneeSelection } from "../../../admin/presentation/hook/useAssigneeSelection";

export function useProductionCreate() {
  const navigate = useNavigate();
  const { user } = useAuthContext();
  const currentMemberUid = user?.uid ?? null;

  const [allProductBlueprints, setAllProductBlueprints] =
    React.useState<ProductBlueprintManagementRow[]>([]);
  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [selectedBrand, setSelectedBrand] = React.useState<string | null>(null);
  const [selectedDetail, setSelectedDetail] =
    React.useState<ProductBlueprintDetailResponse | null>(null);
  const [modelVariations, setModelVariations] =
    React.useState<ModelVariationResponse[]>([]);
  const [modelIndex, setModelIndex] =
    React.useState<Record<string, ModelVariationSummary>>({});
  const [quantityRowVMs, setQuantityRowVMs] =
    React.useState<ProductionQuantityRowVM[]>([]);
  const [brands, setBrands] = React.useState<Brand[]>([]);

  const {
    assigneeId,
    assigneeName: assignee,
    assigneeCandidates: assigneeOptions,
    loadingMembers,
    handleSelectAssignee,
  } = useAssigneeSelection({
    defaultToCurrentMember: false,
  });

  const handleBack = React.useCallback(() => {
    navigate("/production");
  }, [navigate]);

  React.useEffect(() => {
    loadBrands()
      .then(setBrands)
      .catch(() => setBrands([]));
  }, []);

  const brandOptions = React.useMemo(
    () => buildBrandOptions(brands),
    [brands],
  );

  React.useEffect(() => {
    loadProductBlueprints()
      .then(setAllProductBlueprints)
      .catch(() => setAllProductBlueprints([]));
  }, []);

  const filteredBlueprints = React.useMemo(
    () => filterProductBlueprintsByBrand(allProductBlueprints, selectedBrand),
    [allProductBlueprints, selectedBrand],
  );

  const productRows = React.useMemo(
    () => buildProductRows(filteredBlueprints),
    [filteredBlueprints],
  );

  React.useEffect(() => {
    if (!selectedId) {
      setSelectedDetail(null);
      setModelVariations([]);
      setModelIndex({});
      setQuantityRowVMs([]);
      return;
    }

    const productBlueprintId = selectedId;
    let cancelled = false;

    async function loadSelectedDetail() {
      try {
        const { detail, models } = await loadDetailAndModels(productBlueprintId);

        if (cancelled) {
          return;
        }

        setSelectedDetail(detail);
        setModelVariations(models);
        setModelIndex(buildModelIndexFromVariations(models));
      } catch {
        if (cancelled) {
          return;
        }

        setSelectedDetail(null);
        setModelVariations([]);
        setModelIndex({});
        setQuantityRowVMs([]);
      }
    }

    void loadSelectedDetail();

    return () => {
      cancelled = true;
    };
  }, [selectedId]);

  React.useEffect(() => {
    if (!selectedDetail) {
      setQuantityRowVMs([]);
      return;
    }

    const orderByModelId = new Map(
      (selectedDetail.modelRefs ?? []).map((ref) => [
        ref.modelId,
        ref.displayOrder,
      ]),
    );

    const pseudoModels = modelVariations.map((model, index) => ({
      ModelID: model.id,
      Quantity: 0,
      DisplayOrder: orderByModelId.get(model.id) ?? index + 1,
    }));

    setQuantityRowVMs(
      buildProductionQuantityRowVMs(
        pseudoModels,
        modelIndex,
      ),
    );
  }, [modelVariations, selectedDetail, modelIndex]);

  const handleSave = React.useCallback(async () => {
    if (!selectedId) {
      alert("商品設計を選択してください");
      return;
    }

    if (!assigneeId) {
      alert("担当者を選択してください");
      return;
    }

    if (!currentMemberUid) {
      alert("ログインユーザー情報を取得できませんでした");
      return;
    }

    const payload = buildProductionPayload({
      productBlueprintId: selectedId,
      assigneeId,
      rows: quantityRowVMs.map((row) => ({
        modelId: row.modelId,
        quantity: row.quantity,
      })),
      currentMemberUid,
    });

    try {
      const repository = new ProductionRepositoryHTTP();
      await repository.create(payload);
      alert("生産計画を作成しました");
      navigate("/production");
    } catch {
      alert("生産計画の作成に失敗しました");
    }
  }, [
    selectedId,
    assigneeId,
    quantityRowVMs,
    currentMemberUid,
    navigate,
  ]);

  return {
    onBack: handleBack,
    onSave: handleSave,
    hasSelectedProductBlueprint: selectedDetail !== null,
    selectedProductBlueprint: selectedDetail,
    assignee,
    assigneeOptions,
    loadingMembers,
    onSelectAssignee: handleSelectAssignee,
    selectedBrand,
    brandOptions,
    selectBrand: setSelectedBrand,
    productRows,
    selectedProductId: selectedId,
    selectProductById: setSelectedId,
    modelVariationsForCard: quantityRowVMs,
    setQuantityRows: setQuantityRowVMs,
  };
}

export default useProductionCreate;