// frontend/console/member/src/presentation/pages/memberManagement.tsx
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import "../styles/member.css";
import { useMemberList } from "../hooks/useMemberList";

import Pagination from "../../../../shell/src/shared/ui/pagination";

export default function MemberManagementPage() {
  const navigate = useNavigate();

  const {
    members,
    loading,
    error,
    isResetting,
    brandMap,
    brandFilterOptions,
    permissionFilterOptions,
    selectedBrandIds,
    setSelectedBrandIds,
    selectedPermissionCats,
    setSelectedPermissionCats,
    extractPermissionCategories,
    sortKey,
    sortDirection,
    handleSortChange,
    handleReset,
    page,
    setPageNumber,
    formatYmd,
  } = useMemberList();

  if (loading) return <div className="p-4">読み込み中...</div>;

  if (error) {
    return (
      <div className="p-4 text-red-500">データ取得エラー: {error.message}</div>
    );
  }

  const goDetail = (uid?: string | null) => {
    const trimmedUid = String(uid ?? "").trim();
    if (!trimmedUid) {
      console.warn("[MemberManagementPage] member uid is empty");
      return;
    }

    navigate(`/member/${encodeURIComponent(trimmedUid)}`);
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
        isResetting={isResetting}
        onCreate={() => navigate("/member/create")}
        onReset={handleReset}
      >
        {members.map((m) => {
          const fallbackInline = `${m.lastName ?? ""} ${m.firstName ?? ""}`.trim();

          const name =
            String(m.displayName ?? "").trim() ||
            fallbackInline ||
            m.email ||
            "招待中";

          const assigned = m.assignedBrands ?? [];
          const categories = extractPermissionCategories(
            (m.permissions ?? []) as string[],
          );

          const memberUid = String(m.uid ?? "").trim();
          const canOpenDetail = memberUid.length > 0;

          return (
            <tr
              key={m.id}
              role="button"
              tabIndex={0}
              className={
                canOpenDetail
                  ? "cursor-pointer"
                  : "cursor-not-allowed opacity-60"
              }
              onClick={() => goDetail(memberUid)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  goDetail(memberUid);
                }
              }}
            >
              <td>{name}</td>
              <td>{m.email ?? ""}</td>

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

              <td>{formatYmd((m as any).createdAt)}</td>
              <td>{formatYmd((m as any).updatedAt)}</td>
            </tr>
          );
        })}
      </List>

      <Pagination
        currentPage={page.number}
        totalPages={page.totalPages ?? 1}
        onPageChange={(p) => setPageNumber(p)}
        className="mt-4"
      />
    </div>
  );
}