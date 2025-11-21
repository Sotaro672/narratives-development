// frontend/console/member/src/presentation/pages/memberManagement.tsx
import * as React from "react";
import { useNavigate } from "react-router-dom";
import List from "../../../../shell/src/layout/List/List";
import "../styles/member.css";
import { useMemberList } from "../hooks/useMemberList";

// ★ フィルタ付きテーブルヘッダー
import FilterableTableHeader from "../../../../shell/src/shared/ui/filterable-table-header";
// ★ ソート可能テーブルヘッダー
import SortableTableHeader, {
  type SortDirection,
} from "../../../../shell/src/shared/ui/sortable-table-header";

// ★ ページネーション
import Pagination from "../../../../shell/src/shared/ui/pagination";

export default function MemberManagementPage() {
  const navigate = useNavigate();

  const {
    members,
    loading,
    error,
    getNameLastFirstByID,
    brandMap,
    brandFilterOptions,
    permissionFilterOptions,
    selectedBrandIds,
    setSelectedBrandIds,
    selectedPermissionCats,
    setSelectedPermissionCats,
    extractPermissionCategories,

    // ★ useMemberList からページ制御パラメータを取得
    page,
    setPageNumber,
  } = useMemberList();

  const [resolvedNames, setResolvedNames] = React.useState<
    Record<string, string>
  >({});

  // ▼ ソート状態
  const [sortKey, setSortKey] = React.useState<string | null>(null);
  const [sortDirection, setSortDirection] =
    React.useState<Exclude<SortDirection, null>>("desc");

  const handleSortChange = React.useCallback(
    (key: string, nextDirection: Exclude<SortDirection, null>) => {
      setSortKey(key);
      setSortDirection(nextDirection);
    },
    [],
  );

  // ▼ Reset ボタン押下時の処理
  const handleReset = React.useCallback(() => {
    setSelectedBrandIds([]);
    setSelectedPermissionCats([]);
    setSortKey(null);
    setSortDirection("desc");
    setPageNumber(1);
  }, [setSelectedBrandIds, setSelectedPermissionCats, setPageNumber]);

  // -------------------------
  //  氏名補完
  // -------------------------
  React.useEffect(() => {
    let disposed = false;

    (async () => {
      const entries = await Promise.all(
        members.map(async (m) => {
          const inline = `${m.lastName ?? ""} ${m.firstName ?? ""}`.trim();
          if (inline) return [m.id, inline] as const;

          const resolved = await getNameLastFirstByID(m.id);
          return [m.id, resolved] as const;
        }),
      );

      if (!disposed) {
        const next: Record<string, string> = {};
        for (const [id, name] of entries) {
          if (name) next[id] = name;
        }
        setResolvedNames(next);
      }
    })();

    return () => {
      disposed = true;
    };
  }, [members, getNameLastFirstByID]);

  const goDetail = (id: string) => {
    if (!id) return;
    navigate(`/member/${encodeURIComponent(id)}`);
  };

  const ymd = (date: any): string => {
    if (!date) return "";
    if (typeof date === "object" && date !== null) {
      if (typeof (date as any).toDate === "function") {
        return (date as any)
          .toDate()
          .toISOString()
          .slice(0, 10)
          .replace(/-/g, "/");
      }
      if (typeof (date as any).seconds === "number") {
        return new Date((date as any).seconds * 1000)
          .toISOString()
          .slice(0, 10)
          .replace(/-/g, "/");
      }
    }
    if (typeof date === "string") {
      return date.slice(0, 10).replace(/-/g, "/");
    }
    return "";
  };

  // ▼ ソート用：createdAt / updatedAt を number に変換
  const getDateValue = React.useCallback(
    (m: any): number => {
      const raw =
        sortKey === "updatedAt" ? (m as any).updatedAt : (m as any).createdAt;
      if (!raw) return 0;

      if (typeof raw === "object" && raw !== null) {
        if (typeof raw.toDate === "function") return raw.toDate().getTime();
        if (typeof raw.seconds === "number") return raw.seconds * 1000;
      }
      if (typeof raw === "string") {
        const t = new Date(raw).getTime();
        return Number.isNaN(t) ? 0 : t;
      }
      return 0;
    },
    [sortKey],
  );

  const sortedMembers = React.useMemo(() => {
    if (!sortKey) return members;

    return [...members].sort((a, b) => {
      const av = getDateValue(a);
      const bv = getDateValue(b);
      return sortDirection === "asc" ? av - bv : bv - av;
    });
  }, [members, sortKey, sortDirection, getDateValue]);

  if (loading) return <div className="p-4">読み込み中...</div>;
  if (error)
    return (
      <div className="p-4 text-red-500">データ取得エラー: {error.message}</div>
    );

  return (
    <div className="p-0">
      <List
        title="メンバー管理"
        headerCells={[
          "氏名",
          "メールアドレス",
          <FilterableTableHeader
            key="brand-header"
            label="所属ブランド"
            options={brandFilterOptions}
            selected={selectedBrandIds}
            onChange={setSelectedBrandIds}
            dialogTitle="所属ブランドで絞り込み"
          />,
          <FilterableTableHeader
            key="perm-header"
            label="権限"
            options={permissionFilterOptions}
            selected={selectedPermissionCats}
            onChange={setSelectedPermissionCats}
            dialogTitle="権限カテゴリで絞り込み"
          />,
          <SortableTableHeader
            key="createdAt-header"
            label="登録日"
            sortKey="createdAt"
            activeKey={sortKey}
            direction={sortDirection}
            onChange={handleSortChange}
          />,
          <SortableTableHeader
            key="updatedAt-header"
            label="更新日"
            sortKey="updatedAt"
            activeKey={sortKey}
            direction={sortDirection}
            onChange={handleSortChange}
          />,
        ]}
        showCreateButton
        createLabel="メンバー追加"
        showResetButton
        onCreate={() => navigate("/member/create")}
        onReset={handleReset}
      >
        {sortedMembers.map((m) => {
          const inline = `${m.lastName ?? ""} ${m.firstName ?? ""}`.trim();
          const name =
            resolvedNames[m.id] || inline || (m.email ?? "") || m.id;

          const assigned = m.assignedBrands ?? [];
          const categories = extractPermissionCategories(
            (m.permissions ?? []) as string[],
          );

          // ▼ フィルタ適用
          const matchesBrandFilter =
            selectedBrandIds.length === 0 ||
            assigned.some((brandId) => selectedBrandIds.includes(brandId));

          const matchesPermissionFilter =
            selectedPermissionCats.length === 0 ||
            categories.some((cat) => selectedPermissionCats.includes(cat));

          if (!matchesBrandFilter || !matchesPermissionFilter) return null;

          return (
            <tr
              key={m.id}
              role="button"
              tabIndex={0}
              className="cursor-pointer"
              onClick={() => goDetail(m.id)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  goDetail(m.id);
                }
              }}
            >
              <td>{name || "招待中"}</td>
              <td>{m.email ?? ""}</td>

              {/* 所属ブランド */}
              <td>
                {assigned.map((brandId) => {
                  const label = brandMap[brandId] ?? brandId;
                  return (
                    <span key={brandId} className="lp-brand-pill mm-brand-tag">
                      {label}
                    </span>
                  );
                })}
              </td>

              {/* 権限 */}
              <td className="mm-permission-col">
                {categories.length === 0 ? (
                  <span className="text-sm text-[hsl(var(--muted-foreground))]">
                    なし
                  </span>
                ) : (
                  categories.map((cat) => (
                    <span key={cat} className="lp-brand-pill mm-brand-tag">
                      {cat}
                    </span>
                  ))
                )}
              </td>

              {/* 登録日 */}
              <td>{ymd((m as any).createdAt)}</td>

              {/* 更新日 */}
              <td>{ymd((m as any).updatedAt)}</td>
            </tr>
          );
        })}
      </List>

      {/* ★ Pagination：現状は totalPages=1（今後 API から総件数を受け取るよう拡張可能） */}
      <Pagination
        currentPage={page.number}
        totalPages={1}
        onPageChange={(p) => setPageNumber(p)}
        className="mt-4"
      />
    </div>
  );
}
