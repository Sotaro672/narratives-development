// frontend/console/tokenBlueprint/src/presentation/hook/useTokenBlueprintManagement.tsx

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
 * ISO8601 → yyyy/MM/dd 形式に整形
 */
function formatDateYYYYMMDD(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return iso; // パースできなければそのまま返す
  }
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}/${m}/${day}`;
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

    // eslint-disable-next-line no-console
    console.log("[useTokenBlueprintManagement] effect start", {
      hasCurrentMember: !!currentMember,
      companyId: companyId ?? "",
    });

    if (!companyId) return;

    (async () => {
      try {
        // eslint-disable-next-line no-console
        console.log("[useTokenBlueprintManagement] fetch start", { companyId });

        const result = await fetchTokenBlueprintsForCompany(companyId);

        // eslint-disable-next-line no-console
        console.log("[useTokenBlueprintManagement] fetch success", {
          companyId,
          count: result?.length ?? 0,
          sample: (result ?? []).slice(0, 3).map((r) => ({
            id: r.id,
            name: r.name,
            symbol: r.symbol,
            brandId: r.brandId,
            brandName: (r as any).brandName ?? "",
            assigneeId: r.assigneeId,
            assigneeName: (r as any).assigneeName ?? "",
            minted: (r as any).minted,
            iconId: (r as any).iconId,
          })),
        });

        setRows(result);
      } catch (e) {
        // eslint-disable-next-line no-console
        console.error("[useTokenBlueprintManagement] fetch failed", {
          companyId,
          error: e,
        });
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

    // eslint-disable-next-line no-console
    console.log("[useTokenBlueprintManagement] options built", {
      rowsCount: rows.length,
      brandOptionsCount: brandOptions.length,
      assigneeOptionsCount: assigneeOptions.length,
      brandOptionsSample: brandOptions.slice(0, 10),
      assigneeOptionsSample: assigneeOptions.slice(0, 10),
    });

    return { brandOptions, assigneeOptions };
  }, [rows]);

  // フィルタ + ソート適用後の行: service に委譲
  const filteredRows: TokenBlueprint[] = useMemo(() => {
    const out = filterAndSortTokenBlueprints(rows, {
      brandFilter,
      assigneeFilter,
      sortKey,
      sortDir,
    });

    // eslint-disable-next-line no-console
    console.log("[useTokenBlueprintManagement] filter/sort applied", {
      inCount: rows.length,
      outCount: out.length,
      brandFilter,
      assigneeFilter,
      sortKey,
      sortDir,
      sample: out.slice(0, 5).map((r) => ({
        id: r.id,
        name: r.name,
        brandId: r.brandId,
        assigneeId: r.assigneeId,
      })),
    });

    return out;
  }, [rows, brandFilter, assigneeFilter, sortKey, sortDir]);

  // createdAt / updatedAt を yyyy/MM/dd 形式に整形して UI へ渡す
  const displayRows: TokenBlueprint[] = useMemo(() => {
    const out = filteredRows.map((tb) => ({
      ...tb,
      createdAt: tb.createdAt ? formatDateYYYYMMDD(tb.createdAt) : tb.createdAt,
      updatedAt: tb.updatedAt ? formatDateYYYYMMDD(tb.updatedAt) : tb.updatedAt,
    }));

    // eslint-disable-next-line no-console
    console.log("[useTokenBlueprintManagement] displayRows built", {
      count: out.length,
      sample: out.slice(0, 5).map((r) => ({
        id: r.id,
        name: r.name,
        createdAt: r.createdAt,
        updatedAt: r.updatedAt,
      })),
    });

    return out;
  }, [filteredRows]);

  // 行クリックで詳細へ（id を使用）
  const handleRowClick = useCallback(
    (id: string) => {
      // eslint-disable-next-line no-console
      console.log("[useTokenBlueprintManagement] row click", { id });
      navigate(`/tokenBlueprint/${encodeURIComponent(id)}`);
    },
    [navigate],
  );

  const handleCreate = useCallback(() => {
    // eslint-disable-next-line no-console
    console.log("[useTokenBlueprintManagement] create click");
    navigate("/tokenBlueprint/create");
  }, [navigate]);

  const handleReset = useCallback(() => {
    // eslint-disable-next-line no-console
    console.log("[useTokenBlueprintManagement] reset filters/sort");
    setBrandFilter([]);
    setAssigneeFilter([]);
    setSortKey(null);
    setSortDir(null);
  }, []);

  const handleChangeBrandFilter = useCallback((vals: string[]) => {
    // eslint-disable-next-line no-console
    console.log("[useTokenBlueprintManagement] brandFilter change", { vals });
    setBrandFilter(vals);
  }, []);

  const handleChangeAssigneeFilter = useCallback((vals: string[]) => {
    // eslint-disable-next-line no-console
    console.log("[useTokenBlueprintManagement] assigneeFilter change", { vals });
    setAssigneeFilter(vals);
  }, []);

  const handleChangeSort = useCallback((key: string | null, dir: SortDir) => {
    // eslint-disable-next-line no-console
    console.log("[useTokenBlueprintManagement] sort change", { key, dir });
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
