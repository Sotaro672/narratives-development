// frontend/console/shell/src/pages/locationManagement.tsx

import type { KeyboardEvent } from "react";

import { useLocationManagement } from "../features/company/presentation/hook/useLocationManagement";
import List from "../layout/List/List";
import { TableCell, TableRow } from "../shared/ui/table";

import "../styles/location.css";

export default function LocationManagement() {
  const {
    rows,
    handlers: {
      handleCreate,
      handleRowClick,
      handleReset,
    },
    isResetting,
  } = useLocationManagement();

  const headers = [
    "保管場所名",
    "住所",
    "作成者",
    "登録日",
    "更新者",
    "最終更新日",
  ];

  return (
    <List
      title="在庫保管場所"
      headerCells={headers}
      showCreateButton
      createLabel="保管場所を追加"
      onCreate={handleCreate}
      showResetButton
      isResetting={isResetting}
      onReset={handleReset}
    >
      {rows.map((row) => (
        <TableRow
          key={row.id}
          className="location-management__row"
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
          <TableCell>{row.address || "-"}</TableCell>
          <TableCell>{row.createdByName || "-"}</TableCell>
          <TableCell>{row.createdAt || "-"}</TableCell>
          <TableCell>{row.updatedByName || "-"}</TableCell>
          <TableCell>{row.updatedAt || "-"}</TableCell>
        </TableRow>
      ))}
    </List>
  );
}