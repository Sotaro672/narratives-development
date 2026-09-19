// frontend/console/shell/src/pages/accountManagement.tsx

import { useNavigate } from "react-router-dom";

import { useAccountManagement } from "../features/account/presentation/hook/useAccountManagement";
import List from "../layout/List/List";
import { ErrorMessage } from "../shared/ui/error";
import { TableCell, TableRow } from "../shared/ui/table";

import "../styles/account.css";

export default function AccountManagementPage() {
  const navigate = useNavigate();

  const {
    accounts,
    loading,
    isResetting,
    error,
    reload,
  } = useAccountManagement();

  if (loading) {
    return <div className="p-4">読み込み中...</div>;
  }

  if (error) {
    return (
      <div className="p-4">
        <ErrorMessage>
          データ取得エラー: {error.message}
        </ErrorMessage>
      </div>
    );
  }

  const headers = [
    "銀行名",
    "支店名",
    "口座番号",
    "種別",
    "ブランド",
    "ステータス",
    "登録日",
  ];

  return (
    <div className="p-0">
      <List
        title="口座管理"
        headerCells={headers}
        showCreateButton
        createLabel="口座追加"
        showResetButton
        isResetting={isResetting}
        onCreate={() => navigate("/account/connect")}
        onReset={reload}
      >
        {accounts.length === 0 ? (
          <TableRow>
            <TableCell colSpan={headers.length}>
              <div className="account-empty">
                登録されている口座はありません。
              </div>
            </TableCell>
          </TableRow>
        ) : (
          accounts.map((account) => (
            <TableRow key={account.id}>
              <TableCell>{account.bankName}</TableCell>
              <TableCell>{account.branchName}</TableCell>
              <TableCell>{account.accountNumberLabel}</TableCell>
              <TableCell>{account.accountTypeLabel}</TableCell>
              <TableCell>
                {account.connectedBrands.map((brand) => (
                  <span
                    key={brand.id}
                    className="lp-brand-pill account-brand-tag"
                  >
                    {brand.name}
                  </span>
                ))}
              </TableCell>
              <TableCell>{account.statusLabel}</TableCell>
              <TableCell>{account.registeredAt}</TableCell>
            </TableRow>
          ))
        )}
      </List>
    </div>
  );
}