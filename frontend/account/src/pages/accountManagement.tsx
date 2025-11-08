// frontend/account/src/pages/accountManagement.tsx
import * as React from "react";
import List from "../../../shell/src/layout/List/List";
import { Filter } from "lucide-react";
import "./accountManagement.css";
import { ACCOUNTS, type Account } from "../../mockdata";

// Lucide型エラー対策
const IconFilter = Filter as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;

export default function AccountManagementPage() {
  const headers: React.ReactNode[] = [
    "アカウントID",
    "氏名",
    "メールアドレス",
    <>
      <span className="inline-flex items-center gap-2">
        ロール
        <button className="lp-th-filter" aria-label="ロールを絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
    <>
      <span className="inline-flex items-center gap-2">
        ブランド
        <button className="lp-th-filter" aria-label="ブランドを絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
    "登録日",
  ];

  const roleClass = (role: string) => {
    if (role === "管理者") return "account-role-badge is-admin";
    if (role.includes("ブランド")) return "account-role-badge is-brand";
    return "account-role-badge is-default";
  };

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
            <td>{acc.name}</td>
            <td>{acc.email}</td>
            <td>
              <span className={roleClass(acc.role)}>{acc.role}</span>
            </td>
            <td>
              <span className="lp-brand-pill">{acc.brand}</span>
            </td>
            <td>{acc.createdAt}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
