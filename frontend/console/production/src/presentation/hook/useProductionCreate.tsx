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
      .then((rows) => setAllProductBlueprints(rows))
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

  React.useEffect(() => {
    if (!companyId) {
      setAssigneeCandidates([]);
      return;
    }

    (async () => {
      try {
        // companyId でスコープしたフィルタ（active メンバー想定）
        const filter = scopedFilterByCompanyId(companyId, {
          status: "active",
        });

        const sort: MemberSort = {
          column: "name",
          order: "asc",
        };

        // Page 型の詳細は共通定義に依存するため any で渡す
        const page: any = { number: 1, perPage: 200 };

        // ★ クラスのインスタンスを生成して list を呼ぶ
        const repo = new MemberRepositoryHTTP();
        const result = await repo.list(page, filter);
        // list の戻り値は PageResult<Member> を想定
        setAssigneeCandidates(result.items ?? []);
      } catch (e) {
        console.error("担当者候補一覧の取得に失敗しました:", e);
        setAssigneeCandidates([]);
      }
    })();
  }, [companyId]);

  // UI で使いやすい形にマッピング（id + 表示名）
  const assigneeOptions = React.useMemo(
    () =>
      assigneeCandidates.map((m) => ({
        id: m.id,
        name: m.fullName || m.email || m.id,
      })),
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
    setAssignee,

    // 担当者候補一覧（今後 Popover 等で使う想定）
    assigneeOptions,

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
