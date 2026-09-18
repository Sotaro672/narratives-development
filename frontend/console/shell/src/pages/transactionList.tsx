// frontend/console/shell/src/pages/transactionList.tsx

import React from "react";

import List from "../layout/List/List";
import { useTransactionList } from "../features/transaction/presentation/hook/useTransactionList";
import { Badge, type BadgeVariant } from "../shared/ui/badge";

function getTransactionTypeBadgeVariant(
  type: "receive" | "send",
): BadgeVariant {
  switch (type) {
    case "receive":
      return "success";
    case "send":
      return "danger";
  }
}

function getTransactionStatusBadgeVariant(
  status: string,
): BadgeVariant {
  switch (status) {
    case "transferred":
    case "reversed":
      return "success";
    default:
      return "default";
  }
}

export default function TransactionListPage() {
  const {
    transactions,
    loading,
    isResetting,
    error,
    reload,
  } = useTransactionList();

  if (loading) {
    return (
      <div className="p-4">
        読み込み中...
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 text-red-500">
        データ取得エラー: {error.message}
      </div>
    );
  }

  const headers: React.ReactNode[] = [
    "日時",
    "口座",
    "種別",
    "説明",
    "金額",
    "ステータス",
  ];

  return (
    <div className="p-0">
      <List
        title="取引履歴"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        isResetting={isResetting}
        onReset={reload}
      >
        {transactions.length === 0 ? (
          <tr>
            <td colSpan={6}>
              <div className="py-4 text-center text-sm text-slate-500">
                取引履歴はありません。
              </div>
            </td>
          </tr>
        ) : (
          transactions.map((transaction) => (
            <tr key={transaction.id}>
              <td>{transaction.timestampLabel}</td>
              <td>{transaction.accountLabel}</td>
              <td>
                <Badge variant={getTransactionTypeBadgeVariant(transaction.type)}>
                  {transaction.typeLabel}
                </Badge>
              </td>
              <td>{transaction.description}</td>
              <td>{transaction.amountLabel}</td>
              <td>
                <Badge variant={getTransactionStatusBadgeVariant(transaction.status)}>
                  {transaction.statusLabel}
                </Badge>
              </td>
            </tr>
          ))
        )}
      </List>
    </div>
  );
}