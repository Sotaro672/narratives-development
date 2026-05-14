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

// ViewModel（方針B / 以降はキー名を modelId に統一）
import type { ProductionQuantityRowVM } from "../viewModels/productionQuantityRowVM";
import { buildProductionQuantityRowVMs } from "../viewModels/buildProductionQuantityRowVMs";

type ProductBlueprintModelRef = {
  modelId: string;
  displayOrder?: number;
};

export function useProductionCreate() {
  const navigate = useNavigate();

  // ==========================
  // currentMember 情報
  // ==========================
  const { currentMember, user } = useAuth();
  const creator = currentMember?.fullName ?? "-";

  // createdBy は members docId ではなく Firebase Auth UID を保存する。
  // currentMember.uid が backend response の影響で docId になる可能性があるため、
  // AuthContext の user.uid を最優先にする。
  const currentMemberUid = user?.uid ?? currentMember?.uid ?? null;

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

    let cancelled = false;

    (async () => {
      try {
        const { detail, models } = await loadDetailAndModels(selectedId);

        if (cancelled) return;

        setSelectedDetail(detail);
        setModelVariations(
          Array.isArray(models) ? (models as ModelVariationResponse[]) : [],
        );
      } catch {
        if (cancelled) return;

        setSelectedDetail(null);
        setModelVariations([]);
        setModelIndex({});
        setQuantityRowVMs([]);
      }
    })();

    return () => {
      cancelled = true;
    };
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

        if (!cancelled) {
          setModelIndex(index);
        }
      } catch {
        if (!cancelled) {
          setModelIndex({});
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [selectedId]);

  // ==========================
  // detail.modelRefs + modelVariations → VM rows
  // - ProductBlueprint.detail.modelRefs を主データとして扱う
  // - modelVariations は modelRefs が無い場合の fallback
  // - builder は backend の production.Models 形式
  //   （ModelID/Quantity/DisplayOrder）を正として読む
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

    const refModels = refs
      .map((ref, index) => {
        const modelId = String(ref?.modelId ?? "").trim();

        if (!modelId) {
          return null;
        }

        const displayOrderNum =
          typeof ref?.displayOrder === "number"
            ? ref.displayOrder
            : Number(ref?.displayOrder);

        return {
          ModelID: modelId,
          Quantity: 0,
          DisplayOrder: Number.isFinite(displayOrderNum)
            ? displayOrderNum
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

    const orderByModelId = new Map<string, number>();

    for (const ref of refs) {
      const modelId = String(ref?.modelId ?? "").trim();

      if (!modelId) {
        continue;
      }

      const displayOrderNum =
        typeof ref?.displayOrder === "number"
          ? ref.displayOrder
          : Number(ref?.displayOrder);

      if (!Number.isFinite(displayOrderNum)) {
        continue;
      }

      orderByModelId.set(modelId, displayOrderNum);
    }

    const fallbackModels = safeModels
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

    const pseudoModels = refModels.length > 0 ? refModels : fallbackModels;

    const vms = buildProductionQuantityRowVMs(pseudoModels, modelIndex);
    setQuantityRowVMs(vms);
  }, [selectedId, modelVariations, selectedDetail, modelIndex]);

  // ==========================
  // ProductBlueprintCard 表示用データ
  // ==========================
  const selectedProductBlueprintForCard: ProductBlueprintForCard =
    React.useMemo(
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

    let cancelled = false;

    (async () => {
      try {
        setLoadingMembers(true);
        const members: Member[] = await loadAssigneeCandidates(companyId);

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
    })();

    return () => {
      cancelled = true;
    };
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
        (option: { id: string; name: string }) => option.id === id,
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

    if (!currentMemberUid) {
      alert("ログインユーザー情報を取得できませんでした");
      return;
    }

    const payload = buildProductionPayload({
      productBlueprintId: selectedId,
      assigneeId,
      rows: (Array.isArray(quantityRowVMs) ? quantityRowVMs : []).map(
        (vm, index) => {
          const modelId = String(vm.modelId ?? "").trim() || String(index);

          return {
            modelId,
            quantity: vm.quantity ?? 0,
          };
        },
      ),
      currentMemberUid,
    });

    try {
      const repo = new ProductionRepositoryHTTP();
      await createProduction(repo, payload);

      alert("生産計画を作成しました");
      navigate("/production");
    } catch {
      alert("生産計画の作成に失敗しました");
    }
  }, [selectedId, assigneeId, quantityRowVMs, currentMemberUid, navigate]);

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