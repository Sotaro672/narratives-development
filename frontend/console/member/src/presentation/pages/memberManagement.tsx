// frontend/console/member/src/presentation/pages/memberManagement.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";
import List from "../../../../shell/src/layout/List/List";
import "../styles/member.css";
import { useMemberList } from "../hooks/useMemberList";

// ★ 追加: フィルタ付きテーブルヘッダー
import FilterableTableHeader from "../../../../shell/src/shared/ui/filterable-table-header";

export default function MemberManagementPage() {
  const navigate = useNavigate();

  // メンバー一覧 + 氏名解決関数 + brandId→brandName マップを hook から取得
  const {
    members,
    loading,
    error,
    reload,
    getNameLastFirstByID,
    brandMap,
  } = useMemberList();

  // 氏名キャッシュ（画面側で保持）
  const [resolvedNames, setResolvedNames] = React.useState<
    Record<string, string>
  >({});

  // 所属ブランド列のフィルタ状態（brandId の配列）
  const [selectedBrandIds, setSelectedBrandIds] = React.useState<string[]>([]);

  // 権限列のフィルタ状態（permission category の配列）
  const [selectedPermissionCats, setSelectedPermissionCats] = React.useState<
    string[]
  >([]);

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

  const ymd = (createdAt: any): string => {
    if (!createdAt) return "";
    if (typeof createdAt === "object" && createdAt !== null) {
      if (typeof (createdAt as any).toDate === "function") {
        return (createdAt as any)
          .toDate()
          .toISOString()
          .slice(0, 10)
          .replace(/-/g, "/");
      }
      if (typeof (createdAt as any).seconds === "number") {
        return new Date((createdAt as any).seconds * 1000)
          .toISOString()
          .slice(0, 10)
          .replace(/-/g, "/");
      }
    }
    if (typeof createdAt === "string") {
      return createdAt.slice(0, 10).replace(/-/g, "/");
    }
    return "";
  };

  // permissions からカテゴリ名（先頭の `<category>` 部分）をユニークに抽出
  const extractPermissionCategories = (perms?: string[] | null): string[] => {
    if (!perms || perms.length === 0) return [];
    const set = new Set<string>();
    for (const p of perms) {
      const name = String(p ?? "").trim();
      if (!name) continue;
      const dot = name.indexOf(".");
      const cat = dot > 0 ? name.slice(0, dot) : name;
      if (!cat) continue;
      set.add(cat);
    }
    return Array.from(set);
  };

  // ブランドフィルタの候補リスト
  const brandFilterOptions = React.useMemo(
    () =>
      Object.entries(brandMap).map(([id, label]) => ({
        value: id,
        label: label || id,
      })),
    [brandMap],
  );

  // 権限カテゴリフィルタの候補リスト（一覧中のメンバーから集計）
  const permissionFilterOptions = React.useMemo(() => {
    const set = new Set<string>();
    for (const m of members) {
      const cats = extractPermissionCategories(
        (m.permissions ?? []) as string[],
      );
      for (const c of cats) set.add(c);
    }
    return Array.from(set).map((c) => ({ value: c, label: c }));
  }, [members]);

  if (loading) return <div className="p-4">読み込み中...</div>;
  if (error)
    return (
      <div className="p-4 text-red-500">
        データ取得エラー: {error.message}
      </div>
    );

  return (
    <div className="p-0">
      <List
        title="メンバー管理"
        headerCells={[
          "氏名",
          "メールアドレス",
          // 所属ブランド列ヘッダー（フィルタ付き）
          <FilterableTableHeader
            key="brand-header"
            label="所属ブランド"
            options={brandFilterOptions}
            selected={selectedBrandIds}
            onChange={setSelectedBrandIds}
            dialogTitle="所属ブランドで絞り込み"
          />,
          // 権限列ヘッダー（フィルタ付き）
          <FilterableTableHeader
            key="perm-header"
            label="権限"
            options={permissionFilterOptions}
            selected={selectedPermissionCats}
            onChange={setSelectedPermissionCats}
            dialogTitle="権限カテゴリで絞り込み"
          />,
          "登録日",
        ]}
        showCreateButton
        createLabel="メンバー追加"
        showResetButton
        onCreate={() => navigate("/member/create")}
        onReset={() => reload()}
      >
        {members.map((m) => {
          const inline = `${m.lastName ?? ""} ${m.firstName ?? ""}`.trim();
          const name =
            resolvedNames[m.id] || inline || (m.email ?? "") || m.id;

          const assigned = m.assignedBrands ?? [];
          const categories = extractPermissionCategories(
            (m.permissions ?? []) as string[],
          );

          // ── フィルタ適用 ──
          const matchesBrandFilter =
            selectedBrandIds.length === 0 ||
            assigned.some((brandId) => selectedBrandIds.includes(brandId));

          const matchesPermissionFilter =
            selectedPermissionCats.length === 0 ||
            categories.some((cat) => selectedPermissionCats.includes(cat));

          if (!matchesBrandFilter || !matchesPermissionFilter) {
            return null;
          }

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
              <td>
                {assigned.map((brandId) => {
                  const label = brandMap[brandId] ?? brandId;
                  return (
                    <span
                      key={brandId}
                      className="lp-brand-pill mm-brand-tag"
                    >
                      {label}
                    </span>
                  );
                })}
              </td>
              <td>
                <div className="mm-permission-col">
                  {categories.length === 0 ? (
                    <span className="text-sm text-[hsl(var(--muted-foreground))]">
                      なし
                    </span>
                  ) : (
                    categories.map((cat) => (
                      <span
                        key={cat}
                        className="lp-brand-pill mm-brand-tag"
                      >
                        {cat}
                      </span>
                    ))
                  )}
                </div>
              </td>
              <td>{ymd((m as any).createdAt)}</td>
            </tr>
          );
        })}
      </List>
    </div>
  );
}
