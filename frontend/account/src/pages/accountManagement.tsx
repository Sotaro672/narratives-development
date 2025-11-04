// frontend/account/src/pages/accountManagement.tsx
import * as React from "react";
import List from "../../../shell/src/layout/List/List";
import { Filter } from "lucide-react";

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

  const renderRoleBadge = (role: string) => {
    let bg = "#e5e7eb";
    let color = "#111";

    if (role === "管理者") {
      bg = "#ef4444";
      color = "#fff";
    } else if (role.includes("ブランド")) {
      bg = "#0b0f1a";
      color = "#fff";
    }

    return (
      <span
        style={{
          display: "inline-block",
          backgroundColor: bg,
          color,
          fontSize: "0.75rem",
          fontWeight: 700,
          padding: "0.3rem 0.6rem",
          borderRadius: "9999px",
        }}
      >
        {role}
      </span>
    );
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
            <td>{renderRoleBadge(acc.role)}</td>
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
