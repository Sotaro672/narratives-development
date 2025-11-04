// frontend/member/src/pages/memberManagement.tsx
import * as React from "react";
import List from "../../../shell/src/layout/List/List";
import { Filter } from "lucide-react";

// Lucideアイコン型エラー対策
const IconFilter = Filter as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;

type MemberRow = {
  name: string;
  email: string;
  role: string;
  brand: string[];
  taskCount: number;
  permissionCount: number;
  registeredAt: string;
};

const MEMBERS: MemberRow[] = [
  {
    name: "小林 静香",
    email: "designer.lumina@narratives.com",
    role: "生産設計責任者",
    brand: ["LUMINA Fashion"],
    taskCount: 0,
    permissionCount: 2,
    registeredAt: "2024/6/25",
  },
  {
    name: "渡辺 花子",
    email: "support.lumina@narratives.com",
    role: "問い合わせ担当者",
    brand: ["LUMINA Fashion"],
    taskCount: 1,
    permissionCount: 2,
    registeredAt: "2024/6/15",
  },
  {
    name: "中村 拓也",
    email: "token.lumina@narratives.com",
    role: "トークン管理者",
    brand: ["LUMINA Fashion"],
    taskCount: 0,
    permissionCount: 3,
    registeredAt: "2024/4/18",
  },
  {
    name: "伊藤 愛子",
    email: "support.nexus@narratives.com",
    role: "問い合わせ担当者",
    brand: ["NEXUS Street"],
    taskCount: 0,
    permissionCount: 2,
    registeredAt: "2024/4/5",
  },
  {
    name: "田中 雄太",
    email: "marketing.nexus@narratives.com",
    role: "ブランド管理者",
    brand: ["NEXUS Street"],
    taskCount: 1,
    permissionCount: 4,
    registeredAt: "2024/3/22",
  },
  {
    name: "高橋 健太",
    email: "token.nexus@narratives.com",
    role: "トークン管理者",
    brand: ["NEXUS Street"],
    taskCount: 7,
    permissionCount: 3,
    registeredAt: "2024/3/10",
  },
  {
    name: "松本 葵",
    email: "designer.nexus@narratives.com",
    role: "生産設計責任者",
    brand: ["NEXUS Street"],
    taskCount: 0,
    permissionCount: 2,
    registeredAt: "2024/3/5",
  },
  {
    name: "佐藤 美咲",
    email: "manager.lumina@narratives.com",
    role: "ブランド管理者",
    brand: ["LUMINA Fashion"],
    taskCount: 10,
    permissionCount: 4,
    registeredAt: "2024/2/20",
  },
  {
    name: "山田 太郎",
    email: "admin@narratives.com",
    role: "管理者",
    brand: ["LUMINA Fashion", "NEXUS Street"],
    taskCount: 5,
    permissionCount: 12,
    registeredAt: "2024/1/15",
  },
];

export default function MemberManagementPage() {
  const headers: React.ReactNode[] = [
    "氏名",
    "メールアドレス",
    <>
      <span className="inline-flex items-center gap-2">
        <span>ロール</span>
        <button className="lp-th-filter" aria-label="ロールで絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
    <>
      <span className="inline-flex items-center gap-2">
        <span>所属ブランド</span>
        <button className="lp-th-filter" aria-label="ブランドで絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
    "担当数",
    "権限数",
    "登録日",
  ];

  return (
    <div className="p-0">
      <List
        title="メンバー管理"
        headerCells={headers}
        showCreateButton
        createLabel="メンバー追加"
        showResetButton
        onReset={() => console.log("メンバーリスト更新")}
      >
        {MEMBERS.map((m) => (
          <tr key={m.email}>
            <td>{m.name}</td>
            <td>{m.email}</td>
            <td>
              <span
                style={{
                  display: "inline-block",
                  backgroundColor:
                    m.role === "管理者"
                      ? "#ef4444"
                      : m.role.includes("ブランド")
                      ? "#0b0f1a"
                      : m.role.includes("生産")
                      ? "#0b0f1a"
                      : m.role.includes("トークン")
                      ? "#e5e7eb"
                      : "#e5e7eb",
                  color:
                    m.role === "管理者"
                      ? "#fff"
                      : m.role.includes("トークン")
                      ? "#111"
                      : "#fff",
                  fontSize: "0.75rem",
                  fontWeight: 700,
                  padding: "0.3rem 0.6rem",
                  borderRadius: "9999px",
                }}
              >
                {m.role}
              </span>
            </td>
            <td>
              {m.brand.map((b) => (
                <span key={b} className="lp-brand-pill" style={{ marginRight: 6 }}>
                  {b}
                </span>
              ))}
            </td>
            <td>{m.taskCount}</td>
            <td>{m.permissionCount}</td>
            <td>{m.registeredAt}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
