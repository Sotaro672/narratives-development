// frontend/console/brand/src/presentation/pages/brandManagement.tsx
import React from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import "../styles/brand.css";
import { useBrandManagement } from "../hook/useBrandManagement";

export default function BrandManagementPage() {
  const navigate = useNavigate();

  const {
    rows,
    managerOptions,

    managerFilter,
    activeKey,
    direction,

    setManagerFilter,
    setActiveKey,
    setDirection,

    resetFilters,
    isResetting,
  } = useBrandManagement();

  const handleCreateBrand = () => {
    navigate("/brand/create");
  };

  const goDetail = (brandId: string) => {
    navigate(`/brand/${encodeURIComponent(brandId)}`);
  };

  // ---------- テーブルヘッダー ----------
  const headers: React.ReactNode[] = [
    "ブランド名",

    // 責任者フィルタ（ラベルは memberName / managerName）
    <FilterableTableHeader
      key="manager"
      label="責任者"
      options={managerOptions}
      selected={managerFilter}
      onChange={setManagerFilter}
    />,

    // 登録日（ソート可能）
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

    // 更新日（ソート可能）
    <SortableTableHeader
      key="updatedAt"
      label="更新日"
      sortKey="updatedAt"
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
        isResetting={isResetting}
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
            {/* ブランド名 */}
            <td>{b.name}</td>

            {/* 責任者名：backend の memberName をそのまま表示 */}
            <td>{b.memberName ?? ""}</td>

            {/* 登録日 */}
            <td>{b.registeredAt}</td>

            {/* 更新日 */}
            <td>{b.updatedAt}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}