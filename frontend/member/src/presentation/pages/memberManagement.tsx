// frontend/member/src/presentation/pages/memberManagement.tsx

import React, { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import "../styles/member.css";
import type { Member } from "../../domain/entity/member";
import { useMemberList } from "../../hooks/useMemberList";

// Utility: "YYYY/MM/DD" → timestamp
const toTs = (yyyyMd: string) => {
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

type SortKey = "taskCount" | "permissionCount" | "registeredAt" | null;

type MemberRow = {
  id: string;
  name: string;
  email: string;
  role: string; // 表示用（日本語）
  brands: string[];
  taskCount: number;
  permissionCount: number;
  registeredAt: string; // "YYYY/MM/DD"
};

// Role表示用に MemberRole をマッピング
const toDisplayRole = (role: Member["role"]): string => {
  switch (role) {
    case "admin":
      return "管理者";
    case "brand-manager":
      return "ブランド管理者";
    case "token-manager":
      return "トークン管理者";
    case "inquiry-handler":
      return "問い合わせ担当者";
    case "production-designer":
      return "生産設計責任者";
    default:
      return role;
  }
};

// Member → 一覧表示用 MemberRow へ変換
const toMemberRow = (m: Member): MemberRow => {
  const name =
    `${m.lastName ?? ""} ${m.firstName ?? ""}`.trim() ||
    m.email ||
    m.id;

  const registeredAt =
    m.createdAt && m.createdAt.length >= 10
      ? m.createdAt.slice(0, 10).replace(/-/g, "/")
      : "";

  return {
    id: m.id,
    name,
    email: m.email ?? "",
    role: toDisplayRole(m.role),
    brands: m.assignedBrands ?? [],
    taskCount: 0,
    permissionCount: m.permissions?.length ?? 0,
    registeredAt,
  };
};

export default function MemberManagementPage() {
  const navigate = useNavigate();

  // Firestoreからメンバー一覧を取得（useMemberListフック）
  const { members, loading, error, reload } = useMemberList();

  // FirestoreデータをMemberRowに変換
  const baseRows = useMemo<MemberRow[]>(() => {
    return members.map(toMemberRow);
  }, [members]);

  // Filters
  const [roleFilter, setRoleFilter] = useState<string[]>([]);
  const [brandFilter, setBrandFilter] = useState<string[]>([]);

  const roleOptions = useMemo(
    () =>
      Array.from(new Set(baseRows.map((m) => m.role))).map((v) => ({
        value: v,
        label: v,
      })),
    [baseRows]
  );

  const brandOptions = useMemo(
    () =>
      Array.from(new Set(baseRows.flatMap((m) => m.brands))).map((v) => ({
        value: v,
        label: v,
      })),
    [baseRows]
  );

  // Sort
  const [activeKey, setActiveKey] = useState<SortKey>("registeredAt");
  const [direction, setDirection] = useState<"asc" | "desc" | null>("desc");

  // Filter + Sort 適用後の行
  const rows = useMemo(() => {
    let data = baseRows.filter(
      (m) =>
        (roleFilter.length === 0 || roleFilter.includes(m.role)) &&
        (brandFilter.length === 0 ||
          m.brands.some((b) => brandFilter.includes(b)))
    );

    if (activeKey && direction) {
      data = [...data].sort((a, b) => {
        if (activeKey === "registeredAt") {
          const av = toTs(a.registeredAt);
          const bv = toTs(b.registeredAt);
          return direction === "asc" ? av - bv : bv - av;
        }
        const av = a[activeKey] as number;
        const bv = b[activeKey] as number;
        return direction === "asc" ? av - bv : bv - av;
      });
    }

    return data;
  }, [baseRows, roleFilter, brandFilter, activeKey, direction]);

  const headers: React.ReactNode[] = [
    "氏名",
    "メールアドレス",
    <FilterableTableHeader
      key="role"
      label="ロール"
      options={roleOptions}
      selected={roleFilter}
      onChange={setRoleFilter}
    />,
    <FilterableTableHeader
      key="brand"
      label="所属ブランド"
      options={brandOptions}
      selected={brandFilter}
      onChange={setBrandFilter}
    />,
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

  const goDetail = (id: string) => {
    if (!id) return;
    navigate(`/member/${encodeURIComponent(id)}`);
  };

  if (loading) return <div className="p-4">読み込み中...</div>;
  if (error)
    return (
      <div className="p-4 text-red-500">
        データ取得エラー: {error.message}
      </div>
    );

  return (
    <div className="p-0">
      <List
        title="メンバー管理"
        headerCells={headers}
        showCreateButton
        createLabel="メンバー追加"
        showResetButton
        // ✅ メンバー追加ボタン押下時の遷移
        onCreate={() => navigate("/member/create")}
        onReset={() => {
          setRoleFilter([]);
          setBrandFilter([]);
          setActiveKey("registeredAt");
          setDirection("desc");
          reload();
        }}
      >
        {rows.map((m) => (
          <tr
            key={m.id}
            role="button"
            tabIndex={0}
            className="cursor-pointer"
            onClick={() => goDetail(m.id)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                goDetail(m.id);
              }
            }}
          >
            <td>{m.name}</td>
            <td>{m.email}</td>
            <td>
              <span className={roleClass(m.role)}>{m.role}</span>
            </td>
            <td>
              {m.brands.map((b) => (
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
