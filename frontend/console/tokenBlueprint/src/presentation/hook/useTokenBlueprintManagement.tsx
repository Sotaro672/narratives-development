import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../../../../shell/src/auth/presentation/hook/useCurrentMember";
import type { TokenBlueprint } from "../../../../shell/src/shared/types/tokenBlueprint";
import {
  SortKey,
  SortDir,
  fetchTokenBlueprintsForCompany,
  buildOptionsFromTokenBlueprints,
  filterAndSortTokenBlueprints,
} from "../../application/tokenBlueprintManagementService";

export type UseTokenBlueprintManagementResult = {
  rows: TokenBlueprint[];
  brandOptions: { value: string; label: string }[];
  assigneeOptions: { value: string; label: string }[];
  brandFilter: string[];
  assigneeFilter: string[];
  sortKey: SortKey;
  sortDir: SortDir;

  handleChangeBrandFilter: (vals: string[]) => void;
  handleChangeAssigneeFilter: (vals: string[]) => void;
  handleChangeSort: (key: string | null, dir: SortDir) => void;
  handleReset: () => void;
  handleCreate: () => void;
  handleRowClick: (id: string) => void;
};

/**
 * ISO8601 → yyyy/MM/dd HH:mm 形式に整形
 * - 例: 2026/01/24 13:05
 */
function formatDateYYYYMMDDHHmm(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return iso; // パースできなければそのまま返す
  }

  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");

  // 24h 表記
  const hh = String(d.getHours()).padStart(2, "0");
  const mm = String(d.getMinutes()).padStart(2, "0");

  return `${y}/${m}/${day} ${hh}:${mm}`;
}

/**
 * TokenBlueprint Management ページ用ロジック（Hook）
 * - currentMember.companyId に紐づく TokenBlueprint 一覧を service 経由で取得
 * - フィルタ / ソート / 行クリック / 作成ボタン など UI 以外の要素を集約
 */
export function useTokenBlueprintManagement(): UseTokenBlueprintManagementResult {
  const navigate = useNavigate();
  const { currentMember } = useAuth();

  // 一覧データ
  const [rows, setRows] = useState<TokenBlueprint[]>([]);

  // フィルタ状態（brandId / assigneeId ベース）
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [assigneeFilter, setAssigneeFilter] = useState<string[]>([]);

  // ソート状態
  const [sortKey, setSortKey] = useState<SortKey>(null);
  const [sortDir, setSortDir] = useState<SortDir>(null);

  // ─────────────────────────────
  // データ取得: ListByCompanyID usecase を叩く（service に委譲）
  // ─────────────────────────────
  useEffect(() => {
    const companyId = currentMember?.companyId;
    if (!companyId) return;

    (async () => {
      try {
        const result = await fetchTokenBlueprintsForCompany(companyId);
        setRows(result);
      } catch {
        setRows([]);
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentMember?.companyId]);

  // オプション（brandId / assigneeId から算出）:
  // value は ID、label に brandName / assigneeName を渡す
  const { brandOptions, assigneeOptions } = useMemo(() => {
    const base = buildOptionsFromTokenBlueprints(rows);

    const brandNameById = new Map<string, string>();
    const assigneeNameById = new Map<string, string>();

    rows.forEach((r) => {
      const bid = r.brandId?.trim();
      if (bid) {
        const bname = (r as any).brandName ?? "";
        if (bname && !brandNameById.has(bid)) {
          brandNameById.set(bid, bname);
        }
      }

      const aid = r.assigneeId?.trim();
      if (aid) {
        const aname = (r as any).assigneeName ?? "";
        if (aname && !assigneeNameById.has(aid)) {
          assigneeNameById.set(aid, aname);
        }
      }
    });

    const brandOptions = base.brandOptions.map((opt) => ({
      ...opt,
      // brandName があればそれをラベルに使う
      label: brandNameById.get(opt.value) || opt.label || opt.value,
    }));

    const assigneeOptions = base.assigneeOptions.map((opt) => ({
      ...opt,
      // assigneeName があればそれをラベルに使う
      label: assigneeNameById.get(opt.value) || opt.label || opt.value,
    }));

    return { brandOptions, assigneeOptions };
  }, [rows]);

  // フィルタ + ソート適用後の行: service に委譲
  const filteredRows: TokenBlueprint[] = useMemo(() => {
    return filterAndSortTokenBlueprints(rows, {
      brandFilter,
      assigneeFilter,
      sortKey,
      sortDir,
    });
  }, [rows, brandFilter, assigneeFilter, sortKey, sortDir]);

  // createdAt / updatedAt を yyyy/MM/dd HH:mm 形式に整形して UI へ渡す
  const displayRows: TokenBlueprint[] = useMemo(() => {
    return filteredRows.map((tb) => ({
      ...tb,
      createdAt: tb.createdAt ? formatDateYYYYMMDDHHmm(tb.createdAt) : tb.createdAt,
      updatedAt: tb.updatedAt ? formatDateYYYYMMDDHHmm(tb.updatedAt) : tb.updatedAt,
    }));
  }, [filteredRows]);

  // 行クリックで詳細へ（id を使用）
  const handleRowClick = useCallback(
    (id: string) => {
      navigate(`/tokenBlueprint/${encodeURIComponent(id)}`);
    },
    [navigate],
  );

  const handleCreate = useCallback(() => {
    navigate("/tokenBlueprint/create");
  }, [navigate]);

  const handleReset = useCallback(() => {
    setBrandFilter([]);
    setAssigneeFilter([]);
    setSortKey(null);
    setSortDir(null);
  }, []);

  const handleChangeBrandFilter = useCallback((vals: string[]) => {
    setBrandFilter(vals);
  }, []);

  const handleChangeAssigneeFilter = useCallback((vals: string[]) => {
    setAssigneeFilter(vals);
  }, []);

  const handleChangeSort = useCallback((key: string | null, dir: SortDir) => {
    setSortKey((key as SortKey) ?? null);
    setSortDir(dir);
  }, []);

  return {
    rows: displayRows,
    brandOptions,
    assigneeOptions,
    brandFilter,
    assigneeFilter,
    sortKey,
    sortDir,
    handleChangeBrandFilter,
    handleChangeAssigneeFilter,
    handleChangeSort,
    handleReset,
    handleCreate,
    handleRowClick,
  };
}