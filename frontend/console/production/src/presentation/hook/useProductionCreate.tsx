// frontend/console/production/src/presentation/hook/useProductionCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";

// ★ currentMember.fullName, companyId, id 取得
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

// ★ Infrastructure(API) から取得系を import（application からは参照しない）
import {
  loadBrands,
  loadProductBlueprints,
  loadDetailAndModels,
  loadAssigneeCandidates,
} from "../../infrastructure/api/productionCreateApi";

// ★ Presentation(UI) 変換・UI型
import {
  buildBrandOptions,
  filterProductBlueprintsByBrand,
  buildProductRows,
  buildSelectedForCard,
  buildAssigneeOptions,
  mapModelVariationsToRows,
} from "../create/mappers";

import type {
  ProductBlueprintForCard,
  ProductionQuantityRow,
} from "../create/types";

// ★ 型（domain / other modules）
import type { Brand } from "../../../../brand/src/domain/entity/brand";
import type { Member } from "../../../../member/src/domain/entity/member";
import type { ProductBlueprintManagementRow } from "../../../../productBlueprint/src/infrastructure/query/productBlueprintQuery";
import type { ModelVariationResponse } from "../../../../productBlueprint/src/application/productBlueprintDetailService";

// ★ Application(usecase) はコマンド生成・実行のみ
import {
  buildProductionPayload,
  createProduction,
} from "../../application/create/ProductionCreateService";

// ★ Application Port 実装（HTTP Adapter）
import { ProductionRepositoryHTTP } from "../../infrastructure/http/productionRepositoryHTTP";

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

  // ==========================
  // 生産数 rows（ProductionQuantityCard 編集対象）
  // ==========================
  const [quantityRows, setQuantityRows] = React.useState<ProductionQuantityRow[]>(
    [],
  );

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
      .then((rows: ProductBlueprintManagementRow[]) => setAllProductBlueprints(rows))
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
      setQuantityRows([]);
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
        setQuantityRows([]);
      }
    })();
  }, [selectedId]);

  // models → quantityRows 初期化
  React.useEffect(() => {
    const rows: ProductionQuantityRow[] = mapModelVariationsToRows(modelVariations);
    setQuantityRows(rows);
  }, [modelVariations]);

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
    () => buildAssigneeOptions(assigneeCandidates) as Array<{ id: string; name: string }>,
    [assigneeCandidates],
  );

  const handleSelectAssignee = React.useCallback(
    (id: string) => {
      const selected = assigneeOptions.find((o: { id: string; name: string }) => o.id === id);
      const name = selected?.name ?? "未設定";

      setAssigneeId(id);
      setAssignee(name);
    },
    [assigneeOptions],
  );

  // ==========================
  // ProductionQuantityCard rows
  // ==========================
  const modelVariationsForCard = quantityRows;

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
      rows: quantityRows.map((r) => ({
        modelVariationId: r.modelVariationId,
        quantity: r.quantity ?? 0,
      })),
      currentMemberId,
    });

    try {
      // Application の usecase は repo 注入
      const repo = new ProductionRepositoryHTTP();
      await createProduction(repo, payload);

      alert("生産計画を作成しました");
      navigate("/production");
    } catch {
      alert("生産計画の作成に失敗しました");
    }
  }, [selectedId, assigneeId, quantityRows, currentMemberId, navigate]);

  // ==========================
  // hook 返却値
  // ==========================
  return {
    // PageStyle
    onBack: handleBack,
    onSave: handleSave,

    // 左カラム
    hasSelectedProductBlueprint,
    selectedProductBlueprintForCard,

    // 管理カード
    assignee,
    creator,
    createdAt,
    assigneeOptions,
    loadingMembers,
    onSelectAssignee: handleSelectAssignee,

    // ブランド選択
    selectedBrand,
    brandOptions,
    selectBrand: setSelectedBrand,

    // 商品設計一覧
    productRows,
    selectedProductId: selectedId,
    selectProductById: setSelectedId,

    // ProductionQuantityCard
    modelVariationsForCard,
    setQuantityRows,
  };
}
