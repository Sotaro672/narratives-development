// frontend/console/shell/src/pages/inquiryManagement.tsx

import {
  useMemo,
  type KeyboardEvent,
} from "react";
import { useNavigate } from "react-router-dom";

import { useInquiryManagementPage } from "../features/inquiry/presentation/hooks/useInquiryManagementPage";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../layout/List/List";
import Empty from "../shared/ui/empty";
import { ErrorMessage } from "../shared/ui/error";
import {
  TableCell,
  TableRow,
} from "../shared/ui/table";
import Text from "../shared/ui/text";

export default function InquiryManagementPage() {
  const navigate = useNavigate();

  const {
    loading,
    isResetting,
    errorMessage,
    rows,
    statusFilter,
    productNameFilter,
    brandNameFilter,
    statusOptions,
    productNameOptions,
    brandNameOptions,
    sortKey,
    sortDirection,
    setStatusFilter,
    setProductNameFilter,
    setBrandNameFilter,
    handleSortChange,
    handleRefresh,
    handleClickRow,
  } = useInquiryManagementPage();

  const rowElements = useMemo(() => {
    return rows.map((row) => (
      <TableRow
        key={row.inquiryId}
        role="button"
        tabIndex={0}
        className="cursor-pointer"
        onClick={() => handleClickRow(row.inquiryId)}
        onKeyDown={(event: KeyboardEvent<HTMLTableRowElement>) => {
          if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            handleClickRow(row.inquiryId);
          }
        }}
      >
        <TableCell>{row.inquiryType}</TableCell>
        <TableCell>{row.customerName}</TableCell>
        <TableCell>{row.status}</TableCell>
        <TableCell>{row.productName}</TableCell>
        <TableCell>{row.brandName}</TableCell>
        <TableCell>{row.createdAt}</TableCell>
        <TableCell>{row.updatedAt}</TableCell>
      </TableRow>
    ));
  }, [handleClickRow, rows]);

  const headers = useMemo(() => {
    return [
      "問い合わせ種別",
      "お客様名",

      <FilterableTableHeader
        key="status"
        label="ステータス"
        options={statusOptions}
        selected={statusFilter}
        onChange={setStatusFilter}
      />,

      <FilterableTableHeader
        key="productName"
        label="商品名"
        options={productNameOptions}
        selected={productNameFilter}
        onChange={setProductNameFilter}
      />,

      <FilterableTableHeader
        key="brandName"
        label="ブランド"
        options={brandNameOptions}
        selected={brandNameFilter}
        onChange={setBrandNameFilter}
      />,

      <SortableTableHeader
        key="createdAt"
        label="問い合わせ日"
        sortKey="createdAt"
        activeKey={sortKey}
        direction={sortDirection}
        onChange={handleSortChange}
      />,

      <SortableTableHeader
        key="updatedAt"
        label="最終更新日"
        sortKey="updatedAt"
        activeKey={sortKey}
        direction={sortDirection}
        onChange={handleSortChange}
      />,
    ];
  }, [
    statusOptions,
    statusFilter,
    setStatusFilter,
    productNameOptions,
    productNameFilter,
    setProductNameFilter,
    brandNameOptions,
    brandNameFilter,
    setBrandNameFilter,
    sortKey,
    sortDirection,
    handleSortChange,
  ]);

  return (
    <div className="p-0">
      <List
        title="問い合わせ管理"
        headerCells={headers}
        showCreateButton
        createLabel="AMOLに問い合わせ"
        onCreate={() => navigate("/inquiry/create")}
        showResetButton
        onReset={handleRefresh}
        isResetting={isResetting}
      >
        {loading ? (
          <TableRow>
            <TableCell colSpan={headers.length}>
              <Text as="div" tone="muted">
                問い合わせ一覧を読み込み中です。
              </Text>
            </TableCell>
          </TableRow>
        ) : errorMessage ? (
          <TableRow>
            <TableCell colSpan={headers.length}>
              <ErrorMessage>
                {errorMessage}
              </ErrorMessage>
            </TableCell>
          </TableRow>
        ) : rowElements.length > 0 ? (
          rowElements
        ) : (
          <TableRow>
            <TableCell colSpan={headers.length}>
              <Empty description="問い合わせはありません。" />
            </TableCell>
          </TableRow>
        )}
      </List>
    </div>
  );
}