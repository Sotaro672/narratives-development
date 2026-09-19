// frontend/console/shell/src/pages/memberManagement.tsx

import type { KeyboardEvent } from "react";
import { useNavigate } from "react-router-dom";

import { useMemberList } from "../features/member/presentation/hooks/useMemberList";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../layout/List/List";
import { ErrorMessage } from "../shared/ui/error";
import Pagination from "../shared/ui/pagination";
import { TableCell, TableRow } from "../shared/ui/table";

import "../styles/member.css";

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

  if (loading) {
    return (
      <div className="member-management__message">
        読み込み中...
      </div>
    );
  }

  if (error) {
    return (
      <ErrorMessage className="member-management__message">
        データ取得エラー: {error.message}
      </ErrorMessage>
    );
  }

  const goDetail = (memberId: string) => {
    if (!memberId) {
      return;
    }

    navigate(`/member/${encodeURIComponent(memberId)}`);
  };

  return (
    <div className="member-management">
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
          const name = m.displayName || m.email;
          const assigned = m.assignedBrands ?? [];
          const categories = extractPermissionCategories(
            (m.permissions ?? []) as string[],
          );

          return (
            <TableRow
              key={m.id}
              role="button"
              tabIndex={0}
              className="table__row--clickable"
              onClick={() => goDetail(m.id)}
              onKeyDown={(event: KeyboardEvent<HTMLTableRowElement>) => {
                if (event.key === "Enter" || event.key === " ") {
                  event.preventDefault();
                  goDetail(m.id);
                }
              }}
            >
              <TableCell>{name}</TableCell>
              <TableCell>{m.email}</TableCell>

              <TableCell>
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
              </TableCell>

              <TableCell className="mm-permission-col">
                {categories.length === 0 ? (
                  <span className="member-management__empty-permission">
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
              </TableCell>

              <TableCell>{formatYmd((m as any).createdAt)}</TableCell>
              <TableCell>{formatYmd((m as any).updatedAt)}</TableCell>
            </TableRow>
          );
        })}
      </List>

      <Pagination
        currentPage={page.number}
        totalPages={page.totalPages ?? 1}
        onPageChange={(p) => setPageNumber(p)}
      />
    </div>
  );
}