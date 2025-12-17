// frontend/console/list/src/presentation/hook/useListCreate.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import type { AdminAssigneeCandidate } from "../../../../admin/src/presentation/components/AdminCard";

export type ListCreateStatusJa = "出品中" | "停止中" | "";

export type CandidateRow = {
  id: string;
  name: string;
};

export function useListCreate() {
  const navigate = useNavigate();

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
  // ──────────────────────────────────────────────
  const [assigneeOptions] = React.useState<AdminAssigneeCandidate[]>([
    { id: "unassigned", name: "未設定" },
    { id: "yamada", name: "山田 太郎" },
    { id: "sato", name: "佐藤 美咲" },
    { id: "suzuki", name: "鈴木 一郎" },
  ]);

  const [assigneeId, setAssigneeId] = React.useState<string>("unassigned");

  const assigneeName = React.useMemo(() => {
    return (
      assigneeOptions.find((c) => c.id === assigneeId)?.name ?? "未設定"
    );
  }, [assigneeId, assigneeOptions]);

  const [loadingMembers] = React.useState(false);

  // AdminCard は id(string) を渡してくる想定
  const onSelectAssignee = React.useCallback((id: string) => {
    setAssigneeId(id);
  }, []);

  // ──────────────────────────────────────────────
  // ブランド選択（Popover）
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

  const [brandOptions] = React.useState<string[]>([
    "LUMINA Fashion",
    "NARRATIVES Studio",
    "TOKYO Atelier",
  ]);

  const selectBrand = React.useCallback((b: string) => {
    setSelectedBrand(b);
  }, []);

  // ──────────────────────────────────────────────
  // 右カラムの一覧テーブル（ダミー）
  // ──────────────────────────────────────────────
  const [candidateRows] = React.useState<CandidateRow[]>([
    { id: "inv-001", name: "シルクブラウス プレミアムライン / LUM-SS25-001" },
    { id: "inv-002", name: "リネンシャツ リラックスフィット / LUM-SS25-002" },
    { id: "inv-003", name: "ウールコート クラシック / LUM-AW25-010" },
  ]);

  const [selectedCandidateId, setSelectedCandidateId] =
    React.useState<string>("");

  const selectCandidateById = React.useCallback((id: string) => {
    setSelectedCandidateId(id);
  }, []);

  // ──────────────────────────────────────────────
  // ハンドラ
  // ──────────────────────────────────────────────
  const onCreate = React.useCallback(() => {
    alert("出品情報を作成しました（ダミー）");
    navigate("/list");
  }, [navigate]);

  const onBack = React.useCallback(() => navigate(-1), [navigate]);

  return {
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
