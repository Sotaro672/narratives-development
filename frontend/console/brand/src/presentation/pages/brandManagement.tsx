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

// managerId から非同期で表示名を取得して表示するセル
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

    const run = async () => {
      const id = (managerId ?? "").trim();
      if (!id) {
        setName("");
        return;
      }
      try {
        const disp = await getNameLastFirstByID(id);
        if (!cancelled) setName(disp);
      } catch (e) {
        console.error("[ManagerNameCell] failed to resolve name:", e);
        if (!cancelled) setName("");
      }
    };

    void run();

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
    ownerOptions,

    statusFilter,
    ownerFilter,
    activeKey,
    direction,

    setStatusFilter,
    setOwnerFilter,
    setActiveKey,
    setDirection,

    statusBadgeClass,
    resetFilters,
  } = useBrandManagement();

  // member 用フックから ID→表示名 関数だけ借りる
  const { getNameLastFirstByID } = useMemberList();

  // ブランド追加ボタン押下 → /brand/create へ遷移
  const handleCreateBrand = () => {
    navigate("/brand/create");
  };

  // 行クリック → /brand/:id へ遷移
  const goDetail = (brandId: string) => {
    navigate(`/brand/${encodeURIComponent(brandId)}`);
  };

  // ヘッダー
  const headers: React.ReactNode[] = [
    "ブランド名",
    <FilterableTableHeader
      key="status"
      label="ステータス"
      options={statusOptions}
      selected={statusFilter}
      onChange={(values) => setStatusFilter(values as any)}
    />,
    <FilterableTableHeader
      key="owner"
      label="責任者"
      options={ownerOptions}
      selected={ownerFilter}
      onChange={setOwnerFilter}
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
