// frontend/console/brand/src/presentation/pages/brandManagement.tsx
import React from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import "../styles/brand.css";

import { useBrandManagement } from "../hook/useBrandManagement";

// memberID → 「姓 名」を解決するフック
import { useMemberList } from "../../../../member/src/presentation/hooks/useMemberList";

// managerId から非同期で名前を取得して表示するセル
function ManagerNameCell({
  managerId,
  getNameLastFirstByID,
}: {
  managerId?: string | null;
  getNameLastFirstByID: (id: string) => Promise<string>;
}) {
  const [name, setName] = React.useState("");

  React.useEffect(() => {
    let cancelled = false;

    const load = async () => {
      const id = (managerId ?? "").trim();
      if (!id) {
        setName("");
        return;
      }
      try {
        const disp = await getNameLastFirstByID(id);
        if (!cancelled) setName(disp);
      } catch (e) {
        console.error("[ManagerNameCell] name resolve error:", e);
        if (!cancelled) setName("");
      }
    };

    void load();
    return () => {
      cancelled = true;
    };
  }, [managerId, getNameLastFirstByID]);

  return <>{name}</>;
}

export default function BrandManagementPage() {
  const navigate = useNavigate();

  const {
    rows,
    statusOptions,
    managerOptions,        // ★ ownerOptions → managerOptions に変更

    statusFilter,
    managerFilter,         // ★ ownerFilter → managerFilter
    activeKey,
    direction,

    setStatusFilter,
    setManagerFilter,      // ★ setOwnerFilter → setManagerFilter
    setActiveKey,
    setDirection,

    statusBadgeClass,
    resetFilters,
  } = useBrandManagement();

  // member 用フックから ID → 氏名変換関数を利用
  const { getNameLastFirstByID } = useMemberList();

  const handleCreateBrand = () => {
    navigate("/brand/create");
  };

  const goDetail = (brandId: string) => {
    navigate(`/brand/${encodeURIComponent(brandId)}`);
  };

  // ---------- テーブルヘッダー ----------
  const headers: React.ReactNode[] = [
    "ブランド名",
    <FilterableTableHeader
      key="status"
      label="ステータス"
      options={statusOptions}
      selected={statusFilter}
      onChange={(values) => setStatusFilter(values as any)}
    />,
    // ★ owner → manager に置き換え
    <FilterableTableHeader
      key="manager"
      label="責任者"
      options={managerOptions}
      selected={managerFilter}
      onChange={setManagerFilter}
    />,
    <SortableTableHeader
      key="registeredAt"
      label="登録日"
      sortKey="registeredAt"
      activeKey={activeKey}
      direction={direction}
      onChange={(key, dir) => {
        setActiveKey(key as any);
        setDirection(dir);
      }}
    />,
  ];

  return (
    <div className="p-0">
      <List
        title="ブランド管理"
        headerCells={headers}
        showCreateButton
        createLabel="ブランド追加"
        onCreate={handleCreateBrand}
        showResetButton
        onReset={resetFilters}
      >
        {rows.map((b) => (
          <tr
            key={b.id}
            role="button"
            tabIndex={0}
            className="cursor-pointer hover:bg-slate-50 transition-colors"
            onClick={() => goDetail(b.id)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                goDetail(b.id);
              }
            }}
          >
            <td>{b.name}</td>

            <td>
              <span className={statusBadgeClass(b.isActive)}>
                {b.isActive ? "アクティブ" : "停止"}
              </span>
            </td>

            {/* ★ ManagerName 表示 */}
            <td>
              <ManagerNameCell
                managerId={b.managerId}
                getNameLastFirstByID={getNameLastFirstByID}
              />
            </td>

            <td>{b.registeredAt}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
