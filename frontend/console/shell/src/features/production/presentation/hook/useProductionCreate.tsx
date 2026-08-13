// frontend/console/shell/src/features/production/presentation/hook/useProductionCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";

import type { Brand } from "../../../../shared/types/brand";
import { useAssigneeSelection } from "../../../admin/presentation/hook/useAssigneeSelection";
import type { ProductBlueprintManagementRow } from "../../../productBlueprint/infrastructure/query/productBlueprintQuery";
import { buildProductionPayload } from "../../application/create/ProductionCreateService";
import type { ProductionQuantityRow } from "../../application/productionQuantityRow";
import {
  loadBrands,
  loadProductBlueprints,
  loadProductionCreateContext,
  type ProductionCreateProductBlueprintResponse,
} from "../../infrastructure/api/productionCreateApi";
import { ProductionRepositoryHTTP } from "../../infrastructure/http/productionRepositoryHTTP";
import {
  buildBrandOptions,
  buildProductRows,
  filterProductBlueprintsByBrand,
} from "../create/mappers";

export function useProductionCreate() {
  const navigate = useNavigate();

  const [allProductBlueprints, setAllProductBlueprints] = React.useState<ProductBlueprintManagementRow[]>([]);
  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [selectedBrand, setSelectedBrand] = React.useState<string | null>(null);
  const [selectedProductBlueprint, setSelectedProductBlueprint] =
    React.useState<ProductionCreateProductBlueprintResponse | null>(null);
  const [quantityRows, setQuantityRows] = React.useState<ProductionQuantityRow[]>([]);
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
    loadBrands().then(setBrands).catch(() => setBrands([]));
  }, []);

  React.useEffect(() => {
    loadProductBlueprints().then(setAllProductBlueprints).catch(() => setAllProductBlueprints([]));
  }, []);

  const brandOptions = React.useMemo(
    () => buildBrandOptions(brands),
    [brands],
  );

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
      setSelectedProductBlueprint(null);
      setQuantityRows([]);
      return;
    }

    const productBlueprintId = selectedId;
    let cancelled = false;

    async function loadSelectedProductBlueprint() {
      try {
        const context = await loadProductionCreateContext(productBlueprintId);
        if (cancelled) return;

        setSelectedProductBlueprint(context.productBlueprintPatch);
        setQuantityRows(context.rows);
      } catch {
        if (cancelled) return;

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
    selectedBrand,
    brandOptions,
    selectBrand: setSelectedBrand,
    productRows,
    selectedProductId: selectedId,
    selectProductById: setSelectedId,
    quantityRows,
    setQuantityRows,
  };
}

export default useProductionCreate;