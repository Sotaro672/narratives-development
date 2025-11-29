// frontend/console/production/src/presentation/hook/useProductionCreate.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";

import { fetchAllBrandsForCompany } from "../../../../brand/src/infrastructure/query/brandQuery";
import type { Brand } from "../../../../brand/src/domain/entity/brand";

import {
  fetchProductBlueprintManagementRows,
  type ProductBlueprintManagementRow,
} from "../../../../productBlueprint/src/infrastructure/query/productBlueprintQuery";

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
  name: string;
  brand?: string;
};

export function useProductionCreate() {
  const navigate = useNavigate();

  // ★ currentMember から fullName / companyId を利用
  const { currentMember } = useAuth();
  const creator = currentMember?.fullName ?? "-";
  const companyId = currentMember?.companyId?.trim() ?? "";

  // ==========================
  // 商品設計一覧（backend から取得）
  // ==========================
  const [allProductBlueprints, setAllProductBlueprints] =
    React.useState<ProductBlueprintManagementRow[]>([]);

  // ==========================
  // 選択中の商品設計・ブランド
  // ==========================
  const [selectedId, setSelectedId] = React.useState<string | null>(null);
  const [selectedBrand, setSelectedBrand] = React.useState<string | null>(null);

  // ==========================
  // Colors（後で API 連携予定）
  // ==========================
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
  // ブランド一覧（API）
  // ==========================
  const [brands, setBrands] = React.useState<Brand[]>([]);

  React.useEffect(() => {
    fetchAllBrandsForCompany("", true)
      .then((items) => setBrands(items))
      .catch((e) => {
        console.error("ブランド取得失敗:", e);
        setBrands([]);
      });
  }, []);

  const brandOptions = React.useMemo(
    () => brands.map((b) => b.name).filter(Boolean),
    [brands],
  );

  // ==========================
  // 商品設計一覧取得（API）
  // ==========================
  React.useEffect(() => {
    fetchProductBlueprintManagementRows()
      .then((rows) => {
        console.log(
          "[ProductionCreate] fetched productBlueprints:",
          rows,
        );
        setAllProductBlueprints(rows);
      })
      .catch((e) => {
        console.error("商品設計一覧取得失敗:", e);
        setAllProductBlueprints([]);
      });
  }, []);

  // ==========================
  // ブランドで商品設計を絞る
  // ==========================
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

  // ==========================
  // 選択中の商品設計（カード表示用）
  // ==========================
  const selectedMgmtRow = React.useMemo(
    () => allProductBlueprints.find((pb) => pb.id === selectedId) ?? null,
    [allProductBlueprints, selectedId],
  );

  const selectedForCard: ProductBlueprintForCard = selectedMgmtRow
    ? {
        id: selectedMgmtRow.id,
        name: selectedMgmtRow.productName,
        brand: selectedMgmtRow.brandName,
      }
    : {
        id: "",
        name: "",
        brand: "",
      };

  const hasSelected = selectedMgmtRow != null;

  // ==========================
  // 担当者候補一覧（MemberRepository 経由）
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
        // companyId でスコープしたフィルタ（active メンバー想定）
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
        console.error("担当者候補一覧の取得に失敗しました:", e);
        setAssigneeCandidates([]);
      } finally {
        setLoadingMembers(false);
      }
    })();
  }, [companyId]);

  // UI で使いやすい形にマッピング（id + 表示名 = fullName優先）
  const assigneeOptions = React.useMemo(
    () =>
      assigneeCandidates.map((m) => {
        const full = getMemberFullName(m); // lastName → firstName
        return {
          id: m.id,
          name: full || m.email || m.id,
        };
      }),
    [assigneeCandidates],
  );

  // 担当者選択時：assignee を更新
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
    });

    alert("生産計画を作成しました（ダミー）");
    navigate("/production");
  }, [navigate, selectedId, colors, creator, assignee]);

  return {
    // PageStyle 用
    onBack: handleBack,
    onSave: handleSave,

    // 左カラム
    hasSelectedProductBlueprint: hasSelected,
    selectedProductBlueprintForCard: selectedForCard,

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
  };
}
