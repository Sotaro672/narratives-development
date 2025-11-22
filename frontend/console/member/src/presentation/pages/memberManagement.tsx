// frontend/console/member/src/presentation/pages/memberManagement.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import "../styles/member.css";
import { useMemberList } from "../hooks/useMemberList";

// ★ ページネーション（バックエンドページング表示用）
import Pagination from "../../../../shell/src/shared/ui/pagination";

export default function MemberManagementPage() {
  const navigate = useNavigate();

  const {
    // 一覧（フィルタ＆ソート済み）
    members,
    loading,
    error,

    // フィルタ用データ
    brandMap,
    brandFilterOptions,
    permissionFilterOptions,
    selectedBrandIds,
    setSelectedBrandIds,
    selectedPermissionCats,
    setSelectedPermissionCats,
    extractPermissionCategories,

    // ソート状態
    sortKey,
    sortDirection,
    handleSortChange,

    // Reset ボタン
    handleReset,

    // ページング（バックエンド）
    page,
    setPageNumber,

    // 表示用氏名
    resolvedNames,

    // 日付フォーマッタ
    formatYmd,
  } = useMemberList();

  if (loading) return <div className="p-4">読み込み中...</div>;
  if (error)
    return (
      <div className="p-4 text-red-500">データ取得エラー: {error.message}</div>
    );

  const goDetail = (id: string) => {
    if (!id) return;
    navigate(`/member/${encodeURIComponent(id)}`);
  };

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
        onReset={handleReset} // ← Reset ボタンでフィルタ＆ソート＆ページを初期化
      >
        {members.map((m) => {
          const nameFromMap = resolvedNames[m.id];
          const fallbackInline = `${m.lastName ?? ""} ${m.firstName ?? ""}`.trim();
          const name = nameFromMap || fallbackInline || m.email || m.id;

          const assigned = m.assignedBrands ?? [];
          const categories = extractPermissionCategories(
            (m.permissions ?? []) as string[],
          );

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
              <td>{formatYmd((m as any).createdAt)}</td>

              {/* 更新日 */}
              <td>{formatYmd((m as any).updatedAt)}</td>
            </tr>
          );
        })}
      </List>

      {/* ★ バックエンドページング用の UI（今後 totalPages を活用可能） */}
      <Pagination
        currentPage={page.number}
        totalPages={page.totalPages ?? 1}
        onPageChange={(p) => setPageNumber(p)}
        className="mt-4"
      />
    </div>
  );
}
