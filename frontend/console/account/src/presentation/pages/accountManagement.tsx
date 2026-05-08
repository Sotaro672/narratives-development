// frontend/account/src/presentation/pages/accountManagement.tsx

import * as React from "react";
import List from "../../../../shell/src/layout/List/List";
import { Filter } from "lucide-react";
import "../styles/account.css";
import { ACCOUNTS } from "../../infrastructure/mockdata/mockdata";
import type {
  Account,
  AccountStatus,
} from "../../../../shell/src/shared/types/account";

// Lucide型エラー対策
const IconFilter = Filter as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;

const formatAccountNumber = (num: number): string =>
  num.toString().padStart(8, "0");

const formatDate = (iso: string): string => {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const yyyy = d.getFullYear();
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${yyyy}/${mm}/${dd}`;
};

const statusClass = (status: AccountStatus): string => {
  switch (status) {
    case "active":
      return "account-status-badge is-active";
    case "inactive":
      return "account-status-badge is-inactive";
    case "suspended":
      return "account-status-badge is-suspended";
    case "deleted":
      return "account-status-badge is-deleted";
    default:
      return "account-status-badge";
  }
};

export default function AccountManagementPage() {
  const headers: React.ReactNode[] = [
    "口座ID",
    "会員ID",
    <>
      <span className="inline-flex items-center gap-2">
        銀行名
        <button className="lp-th-filter" aria-label="銀行名を絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
    "支店名",
    "口座番号",
    "種別",
    "通貨",
    <>
      <span className="inline-flex items-center gap-2">
        ステータス
        <button className="lp-th-filter" aria-label="ステータスを絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
    "登録日",
  ];

  return (
    <div className="p-0">
      <List
        title="口座管理"
        headerCells={headers}
        showCreateButton
        createLabel="口座登録"
        onCreate={() => console.log("新規口座登録")}
        showResetButton
        onReset={() => console.log("リスト更新")}
      >
        {ACCOUNTS.map((acc: Account) => (
          <tr key={acc.id}>
            <td>{acc.id}</td>
            <td>{acc.memberId}</td>
            <td>{acc.bankName}</td>
            <td>{acc.branchName}</td>
            <td>{formatAccountNumber(acc.accountNumber)}</td>
            <td>{acc.accountType}</td>
            <td>{acc.currency}</td>
            <td>
              <span className={statusClass(acc.status)}>{acc.status}</span>
            </td>
            <td>{formatDate(acc.createdAt)}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
