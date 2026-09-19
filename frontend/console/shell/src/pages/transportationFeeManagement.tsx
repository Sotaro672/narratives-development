// frontend/console/shell/src/pages/transportationFeeManagement.tsx

import type { KeyboardEvent } from "react";

import { useTransportationFeeManagement } from "../features/transportation/presentation/hook/useTransportationFeeManagement";
import List from "../layout/List/List";
import { TableCell, TableRow } from "../shared/ui/table";

export default function TransportationFeeManagement() {
  const {
    rows,
    handlers: {
      handleCreate,
      handleRowClick,
      handleReset,
    },
    isResetting,
  } = useTransportationFeeManagement();

  const headers = [
    "料金設定名",
    "作成者",
    "作成日",
    "更新者",
    "最終更新日",
  ];

  return (
    <List
      title="配送料金"
      headerCells={headers}
      showCreateButton
      createLabel="配送料金を作成"
      onCreate={handleCreate}
      showResetButton
      isResetting={isResetting}
      onReset={handleReset}
    >
      {rows.map((row) => (
        <TableRow
          key={row.id}
          className="cursor-pointer"
          role="button"
          tabIndex={0}
          onClick={() => handleRowClick(row)}
          onKeyDown={(event: KeyboardEvent<HTMLTableRowElement>) => {
            if (event.key === "Enter" || event.key === " ") {
              event.preventDefault();
              handleRowClick(row);
            }
          }}
        >
          <TableCell>{row.name || "-"}</TableCell>
          <TableCell>{row.createdByName || "-"}</TableCell>
          <TableCell>{row.createdAt || "-"}</TableCell>
          <TableCell>{row.updatedByName || "-"}</TableCell>
          <TableCell>{row.updatedAt || "-"}</TableCell>
        </TableRow>
      ))}
    </List>
  );
}