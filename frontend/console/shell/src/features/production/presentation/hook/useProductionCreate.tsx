// frontend/console/shell/src/features/production/presentation/hook/useProductionCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";

import { useAuthContext } from "../../../../auth/application/AuthContext";

import {
  loadBrands,
  loadProductBlueprints,
  loadDetailAndModels,
  loadAssigneeCandidates,
} from "../../infrastructure/api/productionCreateApi";

import { buildModelIndexFromVariations } from "../../application/detail/buildModelVariationIndex";
import type { ModelVariationSummary } from "../../application/detail/types";

import {
  buildBrandOptions,
  filterProductBlueprintsByBrand,
  buildProductRows,
  buildSelectedForCard,
  buildAssigneeOptions,
} from "../create/mappers";

import type { Brand } from "../../../../shared/types/brand";
import type { Member } from "../../../../shared/types/member";
import type { ProductBlueprintManagementRow } from "../../../productBlueprint/infrastructure/query/productBlueprintQuery";
import type { ModelVariationResponse } from "../../../productBlueprint/application/productBlueprintDetailService";
import type { ProductBlueprintForCard } from "../create/types";

import { buildProductionPayload } from "../../application/create/ProductionCreateService";
import { ProductionRepositoryHTTP } from "../../infrastructure/http/productionRepositoryHTTP";

import type { ProductionQuantityRowVM } from "../viewModels/productionQuantityRowVM";
import { buildProductionQuantityRowVMs } from "../viewModels/buildProductionQuantityRowVMs";

type ProductBlueprintModelRef = {
  modelId: string;
  displayOrder?: number;
};

export function useProductionCreate() {
  const navigate = useNavigate();
  const { currentMember, user } = useAuthContext();

  const creator = currentMember?.displayName?.trim() || "-";

  // createdBy は Firebase Auth UID を保存する。
  const currentMemberUid = user?.uid ?? currentMember?.uid ?? null;

  const [allProductBlueprints, setAllProductBlueprints] =
    React.useState<ProductBlueprintManagementRow[]>([]);
  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [selectedBrand, setSelectedBrand] = React.useState<string | null>(null);
  const [selectedDetail, setSelectedDetail] = React.useState<any | null>(null);
  const [modelVariations, setModelVariations] =
    React.useState<ModelVariationResponse[]>([]);
  const [modelIndex, setModelIndex] = React.useState<
    Record<string, ModelVariationSummary>
  >({});
  const [quantityRowVMs, setQuantityRowVMs] =
    React.useState<ProductionQuantityRowVM[]>([]);

  const [assignee, setAssignee] = React.useState("未設定");
  const [assigneeId, setAssigneeId] = React.useState<string | null>(null);

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
      .then((items: Brand[]) => setBrands(items))
      .catch(() => setBrands([]));
  }, []);

  const brandOptions = React.useMemo(
    () => buildBrandOptions(brands),
    [brands],
  );

  // ==========================
  // 商品設計一覧
  // ==========================
  React.useEffect(() => {
    loadProductBlueprints()
      .then((rows: ProductBlueprintManagementRow[]) =>
        setAllProductBlueprints(rows),
      )
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

  const selectedMgmtRow = React.useMemo(
    () =>
      allProductBlueprints.find(
        (productBlueprint) => productBlueprint.id === selectedId,
      ) ?? null,
    [allProductBlueprints, selectedId],
  );

  // ==========================
  // 詳細＋ModelVariation
  // ==========================
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

        const safeModels = Array.isArray(models)
          ? (models as ModelVariationResponse[])
          : [];

        setSelectedDetail(detail);
        setModelVariations(safeModels);
        setModelIndex(buildModelIndexFromVariations(safeModels as any));
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

  // ==========================
  // detail.modelRefs + modelVariations → VM rows
  //
  // Production Create はまだ backend BFF 化前のため、
  // ModelVariation と modelRefs を使って表示用 rows を生成する。
  // ==========================
  React.useEffect(() => {
    if (!selectedId) {
      setQuantityRowVMs([]);
      return;
    }

    const safeModels: ModelVariationResponse[] = Array.isArray(modelVariations)
      ? modelVariations
      : [];

    const refs = Array.isArray(selectedDetail?.modelRefs)
      ? ((selectedDetail.modelRefs as ProductBlueprintModelRef[]) ?? [])
      : [];

    const orderByModelId = new Map<string, number>();

    for (const ref of refs) {
      const modelId = String(ref?.modelId ?? "").trim();

      if (!modelId) {
        continue;
      }

      const displayOrderNumber =
        typeof ref?.displayOrder === "number"
          ? ref.displayOrder
          : Number(ref?.displayOrder);

      if (!Number.isFinite(displayOrderNumber)) {
        continue;
      }

      orderByModelId.set(modelId, displayOrderNumber);
    }

    const pseudoModels = safeModels
      .map((model: any, index: number) => {
        const modelId = String(model?.id ?? "").trim();

        if (!modelId) {
          return null;
        }

        const order = orderByModelId.get(modelId);

        return {
          ModelID: modelId,
          Quantity: 0,
          DisplayOrder:
            typeof order === "number" && Number.isFinite(order)
              ? order
              : index + 1,
        };
      })
      .filter(
        (
          model,
        ): model is {
          ModelID: string;
          Quantity: number;
          DisplayOrder: number;
        } => model !== null,
      );

    const viewModels = buildProductionQuantityRowVMs(
      pseudoModels,
      modelIndex,
    );

    setQuantityRowVMs(viewModels);
  }, [selectedId, modelVariations, selectedDetail, modelIndex]);

  // ==========================
  // ProductBlueprintCard
  // ==========================
  const selectedProductBlueprintForCard: ProductBlueprintForCard =
    React.useMemo(
      () => buildSelectedForCard(selectedDetail, selectedMgmtRow),
      [selectedDetail, selectedMgmtRow],
    );

  const hasSelectedProductBlueprint =
    selectedDetail !== null || selectedMgmtRow !== null;

  // ==========================
  // 担当者候補
  // ==========================
  const [assigneeCandidates, setAssigneeCandidates] =
    React.useState<Member[]>([]);
  const [loadingMembers, setLoadingMembers] = React.useState(false);

  React.useEffect(() => {
    let cancelled = false;

    async function loadMembers() {
      try {
        setLoadingMembers(true);

        const members: Member[] = await loadAssigneeCandidates();

        if (!cancelled) {
          setAssigneeCandidates(members);
        }
      } catch {
        if (!cancelled) {
          setAssigneeCandidates([]);
        }
      } finally {
        if (!cancelled) {
          setLoadingMembers(false);
        }
      }
    }

    void loadMembers();

    return () => {
      cancelled = true;
    };
  }, []);

  const assigneeOptions = React.useMemo(
    () =>
      buildAssigneeOptions(assigneeCandidates) as Array<{
        id: string;
        name: string;
      }>,
    [assigneeCandidates],
  );

  const handleSelectAssignee = React.useCallback(
    (id: string) => {
      const selected = assigneeOptions.find((option) => option.id === id);
      const name = selected?.name ?? "未設定";

      setAssigneeId(id);
      setAssignee(name);
    },
    [assigneeOptions],
  );

  // ==========================
  // 保存
  // ==========================
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
      rows: quantityRowVMs.map((viewModel) => ({
        modelId: viewModel.modelId,
        quantity: viewModel.quantity,
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
    hasSelectedProductBlueprint,
    selectedProductBlueprintForCard,
    assignee,
    creator,
    createdAt,
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