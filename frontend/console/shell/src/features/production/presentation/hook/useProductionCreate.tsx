// frontend/console/shell/src/features/production/presentation/hook/useProductionCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";

import {
  useAuthContext,
} from "../../../../auth/application/AuthContext";

// Infrastructure(API)
import {
  loadBrands,
  loadProductBlueprints,
  loadDetailAndModels,
  loadAssigneeCandidates,
} from "../../infrastructure/api/productionCreateApi";

// Detail側のindex builder（VM builderが要求するため）
import {
  buildModelIndexFromVariations,
  type ModelVariationSummary,
} from "../../application/detail/index";

// Presentation(UI)変換
import {
  buildBrandOptions,
  filterProductBlueprintsByBrand,
  buildProductRows,
  buildSelectedForCard,
  buildAssigneeOptions,
} from "../create/mappers";

// 型
import type {
  Brand,
} from "../../../../shared/types/brand";
import type {
  Member,
} from "../../../../shared/types/member";
import type {
  ProductBlueprintManagementRow,
} from "../../../productBlueprint/infrastructure/query/productBlueprintQuery";
import type {
  ModelVariationResponse,
} from "../../../productBlueprint/application/productBlueprintDetailService";
import type {
  ProductBlueprintForCard,
} from "../create/types";

// Application(usecase)
import {
  buildProductionPayload,
  createProduction,
} from "../../application/create/ProductionCreateService";

// Application Port実装（HTTP Adapter）
import {
  ProductionRepositoryHTTP,
} from "../../infrastructure/http/productionRepositoryHTTP";

// ViewModel（方針B／以降はキー名をmodelIdに統一）
import type {
  ProductionQuantityRowVM,
} from "../viewModels/productionQuantityRowVM";
import {
  buildProductionQuantityRowVMs,
} from "../viewModels/buildProductionQuantityRowVMs";

type ProductBlueprintModelRef = {
  modelId: string;
  displayOrder?: number;
};

export function useProductionCreate() {
  const navigate = useNavigate();

  // ==========================
  // currentMember情報
  // ==========================
  const {
    currentMember,
    user,
  } = useAuthContext();

  const creator =
    currentMember?.displayName?.trim() ||
    "-";

  // createdByはmembers docIdではなくFirebase Auth UIDを保存する。
  // currentMember.uidがBackend responseの影響でdocIdになる可能性があるため、
  // AuthContextのuser.uidを最優先にする。
  const currentMemberUid =
    user?.uid ??
    currentMember?.uid ??
    null;

  // ==========================
  // 商品設計一覧／選択状態
  // ==========================
  const [
    allProductBlueprints,
    setAllProductBlueprints,
  ] = React.useState<
    ProductBlueprintManagementRow[]
  >([]);

  const [
    selectedId,
    setSelectedId,
  ] = React.useState<string | null>(null);

  const [
    selectedBrand,
    setSelectedBrand,
  ] = React.useState<string | null>(null);

  // 選択中の商品設計 詳細＋models
  const [
    selectedDetail,
    setSelectedDetail,
  ] = React.useState<any | null>(null);

  const [
    modelVariations,
    setModelVariations,
  ] = React.useState<
    ModelVariationResponse[]
  >([]);

  // VM builderが要求するmodelIndex
  const [
    modelIndex,
    setModelIndex,
  ] = React.useState<
    Record<string, ModelVariationSummary>
  >({});

  // ==========================
  // 生産数rows（VM正）
  // ==========================
  const [
    quantityRowVMs,
    setQuantityRowVMs,
  ] = React.useState<
    ProductionQuantityRowVM[]
  >([]);

  // ==========================
  // 管理情報（担当者など）
  // ==========================
  const [
    assignee,
    setAssignee,
  ] = React.useState("未設定");

  const [
    assigneeId,
    setAssigneeId,
  ] = React.useState<string | null>(null);

  const [createdAt] = React.useState(
    () =>
      new Date().toLocaleDateString(
        "ja-JP",
      ),
  );

  // ==========================
  // 戻る
  // ==========================
  const handleBack =
    React.useCallback(() => {
      navigate("/production");
    }, [navigate]);

  // ==========================
  // ブランド一覧
  // ==========================
  const [
    brands,
    setBrands,
  ] = React.useState<Brand[]>([]);

  React.useEffect(() => {
    loadBrands()
      .then((items: Brand[]) =>
        setBrands(items),
      )
      .catch(() => setBrands([]));
  }, []);

  const brandOptions =
    React.useMemo(
      () => buildBrandOptions(brands),
      [brands],
    );

  // ==========================
  // 商品設計一覧取得
  // ==========================
  React.useEffect(() => {
    loadProductBlueprints()
      .then(
        (
          rows: ProductBlueprintManagementRow[],
        ) =>
          setAllProductBlueprints(
            rows,
          ),
      )
      .catch(() =>
        setAllProductBlueprints([]),
      );
  }, []);

  // ブランドでフィルタ
  const filteredBlueprints =
    React.useMemo(
      () =>
        filterProductBlueprintsByBrand(
          allProductBlueprints,
          selectedBrand,
        ),
      [
        allProductBlueprints,
        selectedBrand,
      ],
    );

  const productRows =
    React.useMemo(
      () =>
        buildProductRows(
          filteredBlueprints,
        ),
      [filteredBlueprints],
    );

  // 選択中の行
  const selectedMgmtRow =
    React.useMemo(
      () =>
        allProductBlueprints.find(
          (productBlueprint) =>
            productBlueprint.id ===
            selectedId,
        ) ?? null,
      [
        allProductBlueprints,
        selectedId,
      ],
    );

  // ==========================
  // 詳細＋modelVariations＋modelIndex
  // ==========================
  React.useEffect(() => {
    if (!selectedId) {
      setSelectedDetail(null);
      setModelVariations([]);
      setModelIndex({});
      setQuantityRowVMs([]);
      return;
    }

    const productBlueprintId =
      selectedId;

    let cancelled = false;

    async function loadSelectedDetail() {
      try {
        const {
          detail,
          models,
        } = await loadDetailAndModels(
          productBlueprintId,
        );

        if (cancelled) {
          return;
        }

        const safeModels =
          Array.isArray(models)
            ? (
                models as ModelVariationResponse[]
              )
            : [];

        setSelectedDetail(detail);
        setModelVariations(
          safeModels,
        );

        setModelIndex(
          buildModelIndexFromVariations(
            safeModels as any,
          ),
        );
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
  // detail.modelRefs＋modelVariations → VM rows
  // - create画面では/models/by-blueprint/{id}/variationsを行の母数として扱う
  // - modelRefsはdisplayOrder補正にのみ利用する
  // - builderはBackendのproduction.Models形式
  //   （ModelID／Quantity／DisplayOrder）を正として読む
  // ==========================
  React.useEffect(() => {
    if (!selectedId) {
      setQuantityRowVMs([]);
      return;
    }

    const safeModels:
      ModelVariationResponse[] =
      Array.isArray(modelVariations)
        ? modelVariations
        : [];

    const refs = Array.isArray(
      selectedDetail?.modelRefs,
    )
      ? (
          (
            selectedDetail.modelRefs as ProductBlueprintModelRef[]
          ) ?? []
        )
      : [];

    const orderByModelId =
      new Map<string, number>();

    for (const ref of refs) {
      const modelId = String(
        ref?.modelId ?? "",
      ).trim();

      if (!modelId) {
        continue;
      }

      const displayOrderNumber =
        typeof ref?.displayOrder ===
        "number"
          ? ref.displayOrder
          : Number(
              ref?.displayOrder,
            );

      if (
        !Number.isFinite(
          displayOrderNumber,
        )
      ) {
        continue;
      }

      orderByModelId.set(
        modelId,
        displayOrderNumber,
      );
    }

    const fallbackModels =
      safeModels
        .map(
          (
            model: any,
            index: number,
          ) => {
            const modelId = String(
              model?.id ?? "",
            ).trim();

            if (!modelId) {
              return null;
            }

            const order =
              orderByModelId.get(
                modelId,
              );

            return {
              ModelID: modelId,
              Quantity: 0,
              DisplayOrder:
                typeof order ===
                  "number" &&
                Number.isFinite(order)
                  ? order
                  : index + 1,
            };
          },
        )
        .filter(
          (
            model,
          ): model is {
            ModelID: string;
            Quantity: number;
            DisplayOrder: number;
          } => model !== null,
        );

    const pseudoModels =
      fallbackModels;

    const viewModels =
      buildProductionQuantityRowVMs(
        pseudoModels,
        modelIndex,
      );

    setQuantityRowVMs(
      viewModels,
    );
  }, [
    selectedId,
    modelVariations,
    selectedDetail,
    modelIndex,
  ]);

  // ==========================
  // ProductBlueprintCard表示用データ
  // ==========================
  const selectedProductBlueprintForCard:
    ProductBlueprintForCard =
    React.useMemo(
      () =>
        buildSelectedForCard(
          selectedDetail,
          selectedMgmtRow,
        ),
      [
        selectedDetail,
        selectedMgmtRow,
      ],
    );

  const hasSelectedProductBlueprint =
    selectedDetail !== null ||
    selectedMgmtRow !== null;

  // ==========================
  // 担当者候補
  // ==========================
  const [
    assigneeCandidates,
    setAssigneeCandidates,
  ] = React.useState<Member[]>([]);

  const [
    loadingMembers,
    setLoadingMembers,
  ] = React.useState(false);

  React.useEffect(() => {
    let cancelled = false;

    async function loadMembers() {
      try {
        setLoadingMembers(true);

        const members: Member[] =
          await loadAssigneeCandidates();

        if (!cancelled) {
          setAssigneeCandidates(
            members,
          );
        }
      } catch {
        if (!cancelled) {
          setAssigneeCandidates(
            [],
          );
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

  const assigneeOptions =
    React.useMemo(
      () =>
        buildAssigneeOptions(
          assigneeCandidates,
        ) as Array<{
          id: string;
          name: string;
        }>,
      [assigneeCandidates],
    );

  const handleSelectAssignee =
    React.useCallback(
      (id: string) => {
        const selected =
          assigneeOptions.find(
            (option: {
              id: string;
              name: string;
            }) =>
              option.id === id,
          );

        const name =
          selected?.name ??
          "未設定";

        setAssigneeId(id);
        setAssignee(name);
      },
      [assigneeOptions],
    );

  // ==========================
  // 保存（BackendへPOST）
  // ==========================
  const handleSave =
    React.useCallback(
      async () => {
        if (!selectedId) {
          alert(
            "商品設計を選択してください",
          );
          return;
        }

        if (!assigneeId) {
          alert(
            "担当者を選択してください",
          );
          return;
        }

        if (!currentMemberUid) {
          alert(
            "ログインユーザー情報を取得できませんでした",
          );
          return;
        }

        const payload =
          buildProductionPayload({
            productBlueprintId:
              selectedId,
            assigneeId,
            rows: (
              Array.isArray(
                quantityRowVMs,
              )
                ? quantityRowVMs
                : []
            ).map(
              (
                viewModel,
                index,
              ) => {
                const modelId =
                  String(
                    viewModel.modelId ??
                      "",
                  ).trim() ||
                  String(index);

                return {
                  modelId,
                  quantity:
                    viewModel.quantity ??
                    0,
                };
              },
            ),
            currentMemberUid,
          });

        try {
          const repository =
            new ProductionRepositoryHTTP();

          await createProduction(
            repository,
            payload,
          );

          alert(
            "生産計画を作成しました",
          );

          navigate("/production");
        } catch {
          alert(
            "生産計画の作成に失敗しました",
          );
        }
      },
      [
        selectedId,
        assigneeId,
        quantityRowVMs,
        currentMemberUid,
        navigate,
      ],
    );

  // ==========================
  // hook返却値
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
    onSelectAssignee:
      handleSelectAssignee,

    selectedBrand,
    brandOptions,
    selectBrand:
      setSelectedBrand,

    productRows,
    selectedProductId:
      selectedId,
    selectProductById:
      setSelectedId,

    modelVariationsForCard:
      quantityRowVMs,
    setQuantityRows:
      setQuantityRowVMs,
  };
}

export default useProductionCreate;