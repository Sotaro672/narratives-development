// frontend/console/shell/src/pages/brandManagement.tsx

import React from "react";
import { useNavigate } from "react-router-dom";

import { useBrandManagement } from "../features/brand/presentation/hook/useBrandManagement";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../layout/List/List";
import { TableCell, TableRow } from "../shared/ui/table";

import "../styles/brand.css";

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

  const headers: React.ReactNode[] = [
    "ブランド名",

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
          <TableRow
            key={b.id}
            role="button"
            tabIndex={0}
            className="cursor-pointer"
            onClick={() => goDetail(b.id)}
            onKeyDown={(e: React.KeyboardEvent<HTMLTableRowElement>) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                goDetail(b.id);
              }
            }}
          >
            <TableCell>{b.name}</TableCell>
            <TableCell>{b.memberName ?? ""}</TableCell>
            <TableCell>{b.registeredAt}</TableCell>
            <TableCell>{b.updatedAt}</TableCell>
          </TableRow>
        ))}
      </List>
    </div>
  );
}