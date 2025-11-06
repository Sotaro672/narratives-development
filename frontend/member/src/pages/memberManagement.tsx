import React, { useMemo, useState } from "react";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";
import "./memberManagement.css";

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

// Utility
const toTs = (yyyyMd: string) => {
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

type SortKey = "taskCount" | "permissionCount" | "registeredAt" | null;

export default function MemberManagementPage() {
  // Filters
  const [roleFilter, setRoleFilter] = useState<string[]>([]);
  const [brandFilter, setBrandFilter] = useState<string[]>([]);

  const roleOptions = useMemo(
    () => Array.from(new Set(MEMBERS.map((m) => m.role))).map((v) => ({ value: v, label: v })),
    []
  );
  const brandOptions = useMemo(
    () =>
      Array.from(new Set(MEMBERS.flatMap((m) => m.brand))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );

  // Sort
  const [activeKey, setActiveKey] = useState<SortKey>("registeredAt");
  const [direction, setDirection] = useState<"asc" | "desc" | null>("desc");

  // Data
  const rows = useMemo(() => {
    let data = MEMBERS.filter(
      (m) =>
        (roleFilter.length === 0 || roleFilter.includes(m.role)) &&
        (brandFilter.length === 0 || m.brand.some((b) => brandFilter.includes(b)))
    );

    if (activeKey && direction) {
      data = [...data].sort((a, b) => {
        if (activeKey === "registeredAt") {
          const av = toTs(a.registeredAt);
          const bv = toTs(b.registeredAt);
          return direction === "asc" ? av - bv : bv - av;
        }
        const av = a[activeKey];
        const bv = b[activeKey];
        return direction === "asc"
          ? (av as number) - (bv as number)
          : (bv as number) - (av as number);
      });
    }

    return data;
  }, [roleFilter, brandFilter, activeKey, direction]);

  // Headers
  const headers: React.ReactNode[] = [
    "氏名",
    "メールアドレス",

    // ロール（Filterable）
    <FilterableTableHeader
      key="role"
      label="ロール"
      options={roleOptions}
      selected={roleFilter}
      onChange={setRoleFilter}
    />,

    // 所属ブランド（Filterable）
    <FilterableTableHeader
      key="brand"
      label="所属ブランド"
      options={brandOptions}
      selected={brandFilter}
      onChange={setBrandFilter}
    />,

    // 担当数（Sortable）
    <SortableTableHeader
      key="taskCount"
      label="担当数"
      sortKey="taskCount"
      activeKey={activeKey}
      direction={direction}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir);
      }}
    />,

    // 権限数（Sortable）
    <SortableTableHeader
      key="permissionCount"
      label="権限数"
      sortKey="permissionCount"
      activeKey={activeKey}
      direction={direction}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir);
      }}
    />,

    // 登録日（Sortable）
    <SortableTableHeader
      key="registeredAt"
      label="登録日"
      sortKey="registeredAt"
      activeKey={activeKey}
      direction={direction}
      onChange={(key, dir) => {
        setActiveKey(key as SortKey);
        setDirection(dir);
      }}
    />,
  ];

  const roleClass = (role: string) => {
    if (role === "管理者") return "member-role-badge is-admin";
    if (role.includes("ブランド")) return "member-role-badge is-brand";
    if (role.includes("生産")) return "member-role-badge is-production";
    if (role.includes("トークン")) return "member-role-badge is-token";
    return "member-role-badge is-default";
  };

  return (
    <div className="p-0">
      <List
        title="メンバー管理"
        headerCells={headers}
        showCreateButton
        createLabel="メンバー追加"
        showResetButton
        onReset={() => {
          setRoleFilter([]);
          setBrandFilter([]);
          setActiveKey("registeredAt");
          setDirection("desc");
          console.log("メンバーリスト更新");
        }}
      >
        {rows.map((m) => (
          <tr key={m.email}>
            <td>{m.name}</td>
            <td>{m.email}</td>
            <td>
              <span className={roleClass(m.role)}>{m.role}</span>
            </td>
            <td>
              {m.brand.map((b) => (
                <span key={b} className="lp-brand-pill mm-brand-tag">
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
