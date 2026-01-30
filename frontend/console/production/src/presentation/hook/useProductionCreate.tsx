// frontend/console/production/src/presentation/hook/useProductionCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";

import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

// Infrastructure(API)
import {
  loadBrands,
  loadProductBlueprints,
  loadDetailAndModels,
  loadAssigneeCandidates,
} from "../../infrastructure/api/productionCreateApi";

// Detail 側の index loader（VM builder が要求するため）
import {
  loadModelVariationIndexByProductBlueprintId,
  type ModelVariationSummary,
} from "../../application/detail/index";

// Presentation(UI) 変換（既存 mappers）
import {
  buildBrandOptions,
  filterProductBlueprintsByBrand,
  buildProductRows,
  buildSelectedForCard,
  buildAssigneeOptions,
} from "../create/mappers";

// 型（既存）
import type { Brand } from "../../../../brand/src/domain/entity/brand";
import type { Member } from "../../../../member/src/domain/entity/member";
import type { ProductBlueprintManagementRow } from "../../../../productBlueprint/src/infrastructure/query/productBlueprintQuery";
import type { ModelVariationResponse } from "../../../../productBlueprint/src/application/productBlueprintDetailService";
import type { ProductBlueprintForCard } from "../create/types";

// Application(usecase)
import {
  buildProductionPayload,
  createProduction,
} from "../../application/create/ProductionCreateService";

// Application Port 実装（HTTP Adapter）
import { ProductionRepositoryHTTP } from "../../infrastructure/http/productionRepositoryHTTP";

// ViewModel（方針B）
import type { ProductionQuantityRowVM } from "../viewModels/productionQuantityRowVM";
import { buildProductionQuantityRowVMs } from "../viewModels/buildProductionQuantityRowVMs";
import { normalizeProductionModels } from "../viewModels/normalizeProductionModels";

export function useProductionCreate() {
  const navigate = useNavigate();

  // ==========================
  // currentMember 情報
  // ==========================
  const { currentMember } = useAuth();
  const creator = currentMember?.fullName ?? "-";
  const currentMemberId = currentMember?.id ?? null;
  const companyId = currentMember?.companyId?.trim() ?? "";

  // ==========================
  // 商品設計一覧 / 選択状態
  // ==========================
  const [allProductBlueprints, setAllProductBlueprints] = React.useState<
    ProductBlueprintManagementRow[]
  >([]);
  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [selectedBrand, setSelectedBrand] = React.useState<string | null>(null);

  // 選択中の商品設計 詳細 + models
  const [selectedDetail, setSelectedDetail] = React.useState<any | null>(null);
  const [modelVariations, setModelVariations] = React.useState<
    ModelVariationResponse[]
  >([]);

  // VM builder が要求する modelIndex
  const [modelIndex, setModelIndex] = React.useState<
    Record<string, ModelVariationSummary>
  >({});

  // ==========================
  // 生産数 rows（VM 正）
  // ==========================
  const [quantityRowVMs, setQuantityRowVMs] = React.useState<
    ProductionQuantityRowVM[]
  >([]);

  // ==========================
  // 管理情報（担当者など）
  // ==========================
  const [assignee, setAssignee] = React.useState("未設定");
  const [assigneeId, setAssigneeId] = React.useState<string | null>(null);
  const [createdAt] = React.useState(() =>
    new Date().toLocaleDateString("ja-JP"),
  );

  // ==========================
  // 戻る
  // ==========================
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

  const brandOptions = React.useMemo(() => buildBrandOptions(brands), [brands]);

  // ==========================
  // 商品設計一覧取得
  // ==========================
  React.useEffect(() => {
    loadProductBlueprints()
      .then((rows: ProductBlueprintManagementRow[]) =>
        setAllProductBlueprints(rows),
      )
      .catch(() => setAllProductBlueprints([]));
  }, []);

  // ブランドでフィルタ
  const filteredBlueprints = React.useMemo(
    () => filterProductBlueprintsByBrand(allProductBlueprints, selectedBrand),
    [allProductBlueprints, selectedBrand],
  );

  const productRows = React.useMemo(
    () => buildProductRows(filteredBlueprints),
    [filteredBlueprints],
  );

  // 選択中の行
  const selectedMgmtRow = React.useMemo(
    () => allProductBlueprints.find((pb) => pb.id === selectedId) ?? null,
    [allProductBlueprints, selectedId],
  );

  // ==========================
  // 詳細 + modelVariations
  // ==========================
  React.useEffect(() => {
    if (!selectedId) {
      setSelectedDetail(null);
      setModelVariations([]);
      setModelIndex({});
      setQuantityRowVMs([]);
      return;
    }

    (async () => {
      try {
        const { detail, models } = await loadDetailAndModels(selectedId);
        setSelectedDetail(detail);
        setModelVariations(models as ModelVariationResponse[]);
      } catch {
        setSelectedDetail(null);
        setModelVariations([]);
        setModelIndex({});
        setQuantityRowVMs([]);
      }
    })();
  }, [selectedId]);

  // ==========================
  // modelIndex（productBlueprintId ベース）
  // ==========================
  React.useEffect(() => {
    if (!selectedId) {
      setModelIndex({});
      return;
    }

    let cancelled = false;

    (async () => {
      try {
        const index = await loadModelVariationIndexByProductBlueprintId(
          selectedId,
        );
        if (!cancelled) setModelIndex(index);
      } catch {
        if (!cancelled) setModelIndex({});
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [selectedId]);

  // ==========================
  // modelVariations + detail.modelRefs → normalized → VM rows
  // ==========================
  React.useEffect(() => {
    if (!selectedId) return;

    const safeModels: ModelVariationResponse[] = Array.isArray(modelVariations)
      ? modelVariations
      : [];

    const refs = (selectedDetail?.modelRefs ?? []) as Array<{
      modelId: string;
      displayOrder?: number;
    }>;

    // displayOrder を modelId で引けるように index 化
    const orderByModelId = new Map<string, number>();
    for (const r of refs) {
      const id = String(r?.modelId ?? "").trim();
      if (!id) continue;

      const n = Number((r as any).displayOrder);
      if (!Number.isFinite(n)) continue;

      orderByModelId.set(id, n);
    }

    // builder が期待する「production.models 風」の配列に寄せる
    // → normalizeProductionModels に渡して揺れ吸収 & shape 統一
    const pseudoModels = safeModels.map((m: any, index: number) => {
      const id = String(m?.id ?? "").trim() || String(index);

      return {
        modelId: id,
        quantity: 0,
        modelNumber: m?.modelNumber ?? "",
        size: m?.size ?? "",
        color: m?.color ?? "",
        rgb: m?.rgb ?? null,
        displayOrder: orderByModelId.get(id),
      };
    });

    const normalized = normalizeProductionModels(pseudoModels as any[]);
    const vms = buildProductionQuantityRowVMs(normalized, modelIndex);

    setQuantityRowVMs(vms);
  }, [selectedId, modelVariations, selectedDetail, modelIndex]);

  // ==========================
  // ProductBlueprintCard 表示用データ
  // ==========================
  const selectedProductBlueprintForCard: ProductBlueprintForCard = React.useMemo(
    () => buildSelectedForCard(selectedDetail, selectedMgmtRow),
    [selectedDetail, selectedMgmtRow],
  );

  const hasSelectedProductBlueprint =
    selectedDetail != null || selectedMgmtRow != null;

  // ==========================
  // 担当者候補
  // ==========================
  const [assigneeCandidates, setAssigneeCandidates] = React.useState<Member[]>(
    [],
  );
  const [loadingMembers, setLoadingMembers] = React.useState(false);

  React.useEffect(() => {
    if (!companyId) return;

    (async () => {
      try {
        setLoadingMembers(true);
        const members: Member[] = await loadAssigneeCandidates(companyId);
        setAssigneeCandidates(members);
      } catch {
        setAssigneeCandidates([]);
      } finally {
        setLoadingMembers(false);
      }
    })();
  }, [companyId]);

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
      const selected = assigneeOptions.find(
        (o: { id: string; name: string }) => o.id === id,
      );
      const name = selected?.name ?? "未設定";

      setAssigneeId(id);
      setAssignee(name);
    },
    [assigneeOptions],
  );

  // ==========================
  // 保存（バックエンドへ POST）
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

    const payload = buildProductionPayload({
      productBlueprintId: selectedId,
      assigneeId,
      rows: (Array.isArray(quantityRowVMs) ? quantityRowVMs : []).map((vm) => ({
        modelVariationId: String(vm.id ?? "").trim(),
        quantity: vm.quantity ?? 0,
      })),
      currentMemberId,
    });

    try {
      const repo = new ProductionRepositoryHTTP();
      await createProduction(repo, payload);

      alert("生産計画を作成しました");
      navigate("/production");
    } catch {
      alert("生産計画の作成に失敗しました");
    }
  }, [selectedId, assigneeId, quantityRowVMs, currentMemberId, navigate]);

  // ==========================
  // hook 返却値（productionCreate.tsx が期待）
  // ==========================
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
