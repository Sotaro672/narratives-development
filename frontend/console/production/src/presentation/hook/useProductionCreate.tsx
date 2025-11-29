// frontend/console/production/src/presentation/hook/useProductionCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";

import { fetchAllBrandsForCompany } from "../../../../brand/src/infrastructure/query/brandQuery";
import type { Brand } from "../../../../brand/src/domain/entity/brand";

import {
  fetchProductBlueprintManagementRows,
  type ProductBlueprintManagementRow,
} from "../../../../productBlueprint/src/infrastructure/query/productBlueprintQuery";

// ★ 商品設計 詳細取得サービス + ModelVariation 一覧
import {
  getProductBlueprintDetail,
  listModelVariationsByProductBlueprintId,
  type ModelVariationResponse,
} from "../../../../productBlueprint/src/application/productBlueprintDetailService";

// ★ itemType / fit の型
import type {
  ItemType,
  Fit,
} from "../../../../productBlueprint/src/domain/entity/catalog";

// ★ currentMember.fullName, companyId 取得
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";

// ★ 担当者候補一覧（MemberRepository）
import type { Member } from "../../../../member/src/domain/entity/member";
import {
  scopedFilterByCompanyId,
  type MemberSort,
} from "../../../../member/src/domain/repository/memberRepository";
import { MemberRepositoryHTTP } from "../../../../member/src/infrastructure/http/memberRepositoryHTTP";
import { getMemberFullName } from "../../../../member/src/domain/entity/member";

type ProductBlueprintForCard = {
  id: string;
  productName: string;
  brand?: string;

  itemType?: ItemType;
  fit?: Fit;
  materials?: string;
  weight?: number;
  washTags?: string[];
  productIdTag?: string;
};

export function useProductionCreate() {
  const navigate = useNavigate();

  const { currentMember } = useAuth();
  const creator = currentMember?.fullName ?? "-";
  const companyId = currentMember?.companyId?.trim() ?? "";

  const [allProductBlueprints, setAllProductBlueprints] =
    React.useState<ProductBlueprintManagementRow[]>([]);

  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [selectedBrand, setSelectedBrand] = React.useState<string | null>(null);

  const [selectedDetail, setSelectedDetail] = React.useState<any | null>(null);

  const [modelVariations, setModelVariations] = React.useState<
    ModelVariationResponse[]
  >([]);

  const [colors] = React.useState<string[]>([]);

  const [assignee, setAssignee] = React.useState("未設定");
  const [createdAt] = React.useState(() =>
    new Date().toLocaleDateString("ja-JP"),
  );

  const handleBack = React.useCallback(() => {
    navigate("/production");
  }, [navigate]);

  const [brands, setBrands] = React.useState<Brand[]>([]);

  React.useEffect(() => {
    fetchAllBrandsForCompany("", true)
      .then((items) => setBrands(items))
      .catch(() => setBrands([]));
  }, []);

  const brandOptions = React.useMemo(
    () => brands.map((b) => b.name).filter(Boolean),
    [brands],
  );

  React.useEffect(() => {
    fetchProductBlueprintManagementRows()
      .then((rows) => {
        setAllProductBlueprints(rows);
      })
      .catch(() => setAllProductBlueprints([]));
  }, []);

  const filteredBlueprints = React.useMemo(() => {
    if (!selectedBrand) return [];
    return allProductBlueprints.filter(
      (pb) => pb.brandName === selectedBrand,
    );
  }, [allProductBlueprints, selectedBrand]);

  const productRows = React.useMemo(
    () =>
      filteredBlueprints.map((pb) => ({
        id: pb.id,
        name: pb.productName,
      })),
    [filteredBlueprints],
  );

  const selectedMgmtRow = React.useMemo(
    () => allProductBlueprints.find((pb) => pb.id === selectedId) ?? null,
    [allProductBlueprints, selectedId],
  );

  React.useEffect(() => {
    if (!selectedId) {
      setSelectedDetail(null);
      setModelVariations([]);
      return;
    }

    (async () => {
      try {
        const [detail, models] = await Promise.all([
          getProductBlueprintDetail(selectedId),
          listModelVariationsByProductBlueprintId(selectedId),
        ]);

        setSelectedDetail(detail as any);
        setModelVariations(models);

        console.log(
          "[ProductionCreate] fetched model variations for productBlueprintId:",
          selectedId,
          models,
        );
      } catch (e) {
        console.error("商品設計詳細/モデル一覧の取得に失敗しました:", e);
        setSelectedDetail(null);
        setModelVariations([]);
      }
    })();
  }, [selectedId]);

  const selectedForCard: ProductBlueprintForCard = selectedDetail
    ? {
        id: selectedDetail.id,
        productName: selectedDetail.productName,
        brand: selectedDetail.brandName ?? "",
        itemType: selectedDetail.itemType as ItemType,
        fit: selectedDetail.fit as Fit,
        materials: selectedDetail.material,
        weight: selectedDetail.weight,
        washTags: selectedDetail.qualityAssurance ?? [],
        productIdTag: selectedDetail.productIdTag?.type ?? "",
      }
    : selectedMgmtRow
      ? {
          id: selectedMgmtRow.id,
          productName: selectedMgmtRow.productName,
          brand: selectedMgmtRow.brandName,
        }
      : {
          id: "",
          productName: "",
          brand: "",
        };

  const hasSelected = selectedDetail != null || selectedMgmtRow != null;

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
        const filter = scopedFilterByCompanyId(companyId, {
          status: "active",
        });

        const sort: MemberSort = {
          column: "name",
          order: "asc",
        };

        const page: any = { number: 1, perPage: 200 };

        const repo = new MemberRepositoryHTTP();
        const result = await repo.list(page, filter);

        setAssigneeCandidates(result.items ?? []);
      } catch (e) {
        setAssigneeCandidates([]);
      } finally {
        setLoadingMembers(false);
      }
    })();
  }, [companyId]);

  const assigneeOptions = React.useMemo(
    () =>
      assigneeCandidates.map((m) => {
        const full = getMemberFullName(m);
        return {
          id: m.id,
          name: full || m.email || m.id,
        };
      }),
    [assigneeCandidates],
  );

  const handleSelectAssignee = React.useCallback(
    (id: string) => {
      const target = assigneeCandidates.find((m) => m.id === id);
      if (!target) return;
      const full = getMemberFullName(target);
      const name = full || target.email || target.id;
      setAssignee(name);
    },
    [assigneeCandidates],
  );

  // ==================================================
  // ★★★ ProductionQuantityCard 用 rows 変換（InventoryCard と同形式）
  // ==================================================
  const modelVariationsForCard = React.useMemo(() => {
    return modelVariations.map((mv) => ({
      modelCode: mv.modelNumber,
      size: mv.size,
      colorName: mv.color?.name ?? "",
      colorCode: mv.color?.rgb
        ? `#${mv.color.rgb.toString(16).padStart(6, "0")}`
        : "#FFFFFF",
      stock: 0,
    }));
  }, [modelVariations]);

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
    onBack: handleBack,
    onSave: handleSave,

    hasSelectedProductBlueprint: hasSelected,
    selectedProductBlueprintForCard: selectedForCard,

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

    modelVariations,
    modelVariationsForCard, // ← ★★★ 新しく返す！
  };
}
