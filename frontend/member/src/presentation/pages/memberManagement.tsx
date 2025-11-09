// frontend/member/src/pages/memberManagement.tsx

import React, { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import "../styles/member.css";
import { MEMBERS, type MemberRow } from "../../../mockdata";

// Utility
const toTs = (yyyyMd: string) => {
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

type SortKey = "taskCount" | "permissionCount" | "registeredAt" | null;

export default function MemberManagementPage() {
  const navigate = useNavigate();

  // Filters
  const [roleFilter, setRoleFilter] = useState<string[]>([]);
  const [brandFilter, setBrandFilter] = useState<string[]>([]);

  const roleOptions = useMemo(
    () =>
      Array.from(new Set(MEMBERS.map((m) => m.role))).map((v) => ({
        value: v,
        label: v,
      })),
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
        (brandFilter.length === 0 ||
          m.brand.some((b) => brandFilter.includes(b)))
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

  // 詳細ページへ遷移（email を ID として利用）
  const goDetail = (email: string) => {
    navigate(`/member/${encodeURIComponent(email)}`);
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
          <tr
            key={m.email}
            role="button"
            tabIndex={0}
            className="cursor-pointer"
            onClick={() => goDetail(m.email)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                goDetail(m.email);
              }
            }}
          >
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
