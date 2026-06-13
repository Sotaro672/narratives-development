// frontend/transaction/src/presentation/pages/transactionList.tsx

import React from "react";
import List from "../../../../shell/src/layout/List/List";
import "../styles/transaction.css";

export default function TransactionListPage() {
  const headers: React.ReactNode[] = [
    "日時",
    "ブランド",
    "種別",
    "説明",
    "金額",
    "取引先",
  ];

  return (
    <div className="p-0">
      <List
        title="取引履歴"
        headerCells={headers}
        showCreateButton={false}
        showResetButton={false}
      >
        <tr>
          <td colSpan={6}>
            <div className="transaction-empty">
              試作品では取引履歴機能は未実装です。
            </div>
          </td>
        </tr>
      </List>
    </div>
  );
}