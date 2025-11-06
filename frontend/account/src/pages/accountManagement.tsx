import * as React from "react";
import List from "../../../shell/src/layout/List/List";
import { Filter } from "lucide-react";
import "./accountManagement.css";

// Lucide型エラー対策
const IconFilter = Filter as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;

type Account = {
  id: string;
  name: string;
  email: string;
  role: string;
  brand: string;
  createdAt: string;
};

const ACCOUNTS: Account[] = [
  {
    id: "acc_001",
    name: "山田 太郎",
    email: "admin@narratives.com",
    role: "管理者",
    brand: "LUMINA Fashion",
    createdAt: "2024/05/20",
  },
  {
    id: "acc_002",
    name: "佐藤 美咲",
    email: "manager.lumina@narratives.com",
    role: "ブランド管理者",
    brand: "LUMINA Fashion",
    createdAt: "2024/06/01",
  },
];

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
        {ACCOUNTS.map((acc) => (
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
