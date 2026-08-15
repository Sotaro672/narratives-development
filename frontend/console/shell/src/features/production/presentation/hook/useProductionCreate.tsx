// frontend/console/shell/src/features/production/presentation/hook/useProductionCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";

import type {
  ProductionCreateProductBlueprint,
  ProductionQuantityRow,
} from "../../../../shared/types/production";
import { useAssigneeSelection } from "../../../admin/presentation/hook/useAssigneeSelection";
import { useBrandSelection } from "../../../brand/presentation/hook/useBrandSelection";
import type { ProductBlueprintListRow } from "../../../productBlueprint/infrastructure/repository/productBlueprintRepositoryHTTP";
import { buildProductionPayload } from "../../application/productionCreateService";
import {
  loadProductBlueprints,
  loadProductionCreateContext,
} from "../../infrastructure/api/productionCreateApi";
import { ProductionRepositoryHTTP } from "../../infrastructure/http/productionRepositoryHTTP";
import {
  buildProductRows,
  filterProductBlueprintsByBrand,
} from "../create/mappers";

export function useProductionCreate() {
  const navigate = useNavigate();

  const [allProductBlueprints, setAllProductBlueprints] = React.useState<ProductBlueprintListRow[]>([]);
  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [selectedProductBlueprint, setSelectedProductBlueprint] =
    React.useState<ProductionCreateProductBlueprint | null>(null);
  const [quantityRows, setQuantityRows] = React.useState<ProductionQuantityRow[]>([]);

  const {
    assigneeId,
    assigneeName: assignee,
    assigneeCandidates: assigneeOptions,
    loadingMembers,
    handleSelectAssignee,
  } = useAssigneeSelection({
    defaultToCurrentMember: false,
  });

  const {
    brandId: selectedBrandId,
    brandName: selectedBrandName,
    brandOptions,
    loadingBrands,
    brandError,
    selectBrand,
  } = useBrandSelection();

  const handleBack = React.useCallback(() => {
    navigate("/production");
  }, [navigate]);

  React.useEffect(() => {
    loadProductBlueprints()
      .then(setAllProductBlueprints)
      .catch(() => setAllProductBlueprints([]));
  }, []);

  const filteredBlueprints = React.useMemo(
    () => filterProductBlueprintsByBrand(allProductBlueprints, selectedBrandName || null),
    [allProductBlueprints, selectedBrandName],
  );

  const productRows = React.useMemo(
    () => buildProductRows(filteredBlueprints),
    [filteredBlueprints],
  );

  const handleSelectBrand = React.useCallback(
    (brandId: string) => {
      selectBrand(brandId);
      setSelectedId(null);
      setSelectedProductBlueprint(null);
      setQuantityRows([]);
    },
    [selectBrand],
  );

  React.useEffect(() => {
    if (!selectedId) {
      setSelectedProductBlueprint(null);
      setQuantityRows([]);
      return;
    }

    const productBlueprintId = selectedId;
    let cancelled = false;

    async function loadSelectedProductBlueprint() {
      try {
        const context = await loadProductionCreateContext(productBlueprintId);

        if (cancelled) {
          return;
        }

        setSelectedProductBlueprint(context.productBlueprintPatch);
        setQuantityRows(context.rows);
      } catch {
        if (cancelled) {
          return;
        }

        setSelectedProductBlueprint(null);
        setQuantityRows([]);
      }
    }

    void loadSelectedProductBlueprint();

    return () => {
      cancelled = true;
    };
  }, [selectedId]);

  const handleSave = React.useCallback(async () => {
    if (!selectedId) {
      alert("商品設計を選択してください");
      return;
    }

    if (!assigneeId) {
      alert("担当者を選択してください");
      return;
    }

    const payload = buildProductionPayload({
      productBlueprintId: selectedId,
      assigneeId,
      rows: quantityRows.map(({ modelId, quantity }) => ({
        modelId,
        quantity,
      })),
    });

    try {
      const repository = new ProductionRepositoryHTTP();
      await repository.create(payload);

      alert("生産計画を作成しました");
      navigate("/production");
    } catch {
      alert("生産計画の作成に失敗しました");
    }
  }, [selectedId, assigneeId, quantityRows, navigate]);

  return {
    onBack: handleBack,
    onSave: handleSave,

    selectedProductBlueprint,

    assignee,
    assigneeOptions,
    loadingMembers,
    onSelectAssignee: handleSelectAssignee,

    selectedBrandId,
    selectedBrandName,
    brandOptions,
    loadingBrands,
    brandError,
    selectBrand: handleSelectBrand,

    productRows,
    selectedProductId: selectedId,
    selectProductById: setSelectedId,

    quantityRows,
    setQuantityRows,
  };
}

export default useProductionCreate;