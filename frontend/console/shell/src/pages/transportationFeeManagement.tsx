// frontend/console/shell/src/pages/transportationFeeManagement.tsx

import List from "../layout/List/List";
import { useTransportationFeeManagement } from "../features/transportation/presentation/hook/useTransportationFeeManagement";

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
        <tr
          key={row.id}
          className="cursor-pointer hover:bg-[rgba(0,0,0,0.03)] transition"
          role="button"
          tabIndex={0}
          onClick={() => handleRowClick(row)}
          onKeyDown={(event) => {
            if (event.key === "Enter" || event.key === " ") {
              event.preventDefault();
              handleRowClick(row);
            }
          }}
        >
          <td>{row.name || "-"}</td>
          <td>{row.createdByName || "-"}</td>
          <td>{row.createdAt || "-"}</td>
          <td>{row.updatedByName || "-"}</td>
          <td>{row.updatedAt || "-"}</td>
        </tr>
      ))}
    </List>
  );
}