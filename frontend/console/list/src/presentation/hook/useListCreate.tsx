// frontend/console/list/src/presentation/hook/useListCreate.tsx
import * as React from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import type { AdminAssigneeCandidate } from "../../../../admin/src/presentation/components/AdminCard";

export type ListCreateStatusJa = "出品中" | "停止中" | "";

export type CandidateRow = {
  id: string;
  name: string;
};

function s(v: unknown): string {
  return String(v ?? "").trim();
}

export function useListCreate() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();

  // ✅ ルート param と query の両方に対応
  // - /listings/create/:inventoryId
  // - /listings/create/:productBlueprintId/:tokenBlueprintId
  // - /listings/create?inventoryId=xxx
  // - /listings/create?productBlueprintId=...&tokenBlueprintId=...
  const params = useParams<{
    inventoryId?: string;
    productBlueprintId?: string;
    tokenBlueprintId?: string;
  }>();

  const inventoryId = React.useMemo(() => {
    return s(params.inventoryId || searchParams.get("inventoryId"));
  }, [params.inventoryId, searchParams]);

  const productBlueprintId = React.useMemo(() => {
    return s(params.productBlueprintId || searchParams.get("productBlueprintId"));
  }, [params.productBlueprintId, searchParams]);

  const tokenBlueprintId = React.useMemo(() => {
    return s(params.tokenBlueprintId || searchParams.get("tokenBlueprintId"));
  }, [params.tokenBlueprintId, searchParams]);

  // ──────────────────────────────────────────────
  // 入力フォーム状態（プリフィルは空）
  // ──────────────────────────────────────────────
  const [product, setProduct] = React.useState("");
  const [brand, setBrand] = React.useState("");
  const [token, setToken] = React.useState("");
  const [stock, setStock] = React.useState<number | "">("");
  const [manager, setManager] = React.useState("");
  const [status, setStatus] = React.useState<ListCreateStatusJa>("");

  // ──────────────────────────────────────────────
  // 管理情報（AdminCard）
  // ※ モック削除：API/Query で取得する前提。未設定だけは固定で残す。
  // ──────────────────────────────────────────────
  const [assigneeOptions] = React.useState<AdminAssigneeCandidate[]>([
    { id: "unassigned", name: "未設定" },
  ]);

  const [assigneeId, setAssigneeId] = React.useState<string>("unassigned");

  const assigneeName = React.useMemo(() => {
    return assigneeOptions.find((c) => c.id === assigneeId)?.name ?? "未設定";
  }, [assigneeId, assigneeOptions]);

  const [loadingMembers] = React.useState(false);

  // AdminCard は id(string) を渡してくる想定
  const onSelectAssignee = React.useCallback((id: string) => {
    setAssigneeId(id);
  }, []);

  // ──────────────────────────────────────────────
  // ブランド選択（Popover）
  // ※ モック削除：API/Query で取得する前提
  // ──────────────────────────────────────────────
  const [selectedBrand, setSelectedBrand] = React.useState<string>("");

  // 既存フォームの brand と連動（どちらから変更しても同期）
  React.useEffect(() => {
    if (brand.trim() && brand.trim() !== selectedBrand) {
      setSelectedBrand(brand.trim());
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [brand]);

  React.useEffect(() => {
    if (selectedBrand.trim() && selectedBrand.trim() !== brand) {
      setBrand(selectedBrand.trim());
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedBrand]);

  const [brandOptions] = React.useState<string[]>([]);

  const selectBrand = React.useCallback((b: string) => {
    setSelectedBrand(b);
  }, []);

  // ──────────────────────────────────────────────
  // 右カラムの一覧テーブル
  // ※ モック削除：inventoryId を受け取って、それを初期選択に使う
  // ──────────────────────────────────────────────
  const [candidateRows] = React.useState<CandidateRow[]>([]);

  const [selectedCandidateId, setSelectedCandidateId] =
    React.useState<string>("");

  // ✅ inventoryId が渡されたら初期選択として反映
  React.useEffect(() => {
    if (!selectedCandidateId && inventoryId) {
      setSelectedCandidateId(inventoryId);
    }
  }, [inventoryId, selectedCandidateId]);

  const selectCandidateById = React.useCallback((id: string) => {
    setSelectedCandidateId(id);
  }, []);

  // ──────────────────────────────────────────────
  // ハンドラ
  // ──────────────────────────────────────────────
  const onCreate = React.useCallback(() => {
    // TODO: 実装（API へ作成リクエスト等）
    navigate("/list");
  }, [navigate]);

  const onBack = React.useCallback(() => navigate(-1), [navigate]);

  return {
    // ✅ incoming
    inventoryId,
    productBlueprintId,
    tokenBlueprintId,

    // page actions
    onBack,
    onCreate,

    // left form
    product,
    setProduct,
    brand,
    setBrand,
    token,
    setToken,
    stock,
    setStock,
    manager,
    setManager,
    status,
    setStatus,

    // admin card
    assigneeId,
    assigneeName,
    assigneeOptions,
    loadingMembers,
    onSelectAssignee,

    // brand popover
    selectedBrand,
    brandOptions,
    selectBrand,

    // table
    candidateRows,
    selectedCandidateId,
    selectCandidateById,
  };
}
