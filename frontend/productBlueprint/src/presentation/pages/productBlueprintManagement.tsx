// frontend/productBlueprint/src/pages/productBlueprintManagement.tsx

import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  RAW_ROWS,
  type ProductBlueprintRow,
} from "../../../mockdata";

const toTs = (yyyyMd: string) => {
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

export default function ProductBlueprintManagement() {
  const navigate = useNavigate();

  // フィルタ状態
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [sortedKey, setSortedKey] = useState<"createdAtA" | "createdAtB" | null>(
    null
  );
  const [sortedDir, setSortedDir] = useState<"asc" | "desc" | null>(null);

  // フィルタとソート
  const rows = useMemo(() => {
    let work = RAW_ROWS.filter(
      (r) => brandFilter.length === 0 || brandFilter.includes(r.brand)
    );

    if (sortedKey && sortedDir) {
      work = [...work].sort((a, b) => {
        const da = toTs(a[sortedKey]);
        const db = toTs(b[sortedKey]);
        return sortedDir === "asc" ? da - db : db - da;
      });
    }
    return work;
  }, [brandFilter, sortedKey, sortedDir]);

  const headers = [
    "プロダクト",
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={[
        { value: "LUMINA Fashion", label: "LUMINA Fashion" },
        { value: "NEXUS Street", label: "NEXUS Street" },
      ]}
      selected={brandFilter}
      onChange={(values: string[]) => setBrandFilter(values)}
    />,
    "担当者",
    "商品ID",
    <SortableTableHeader
      key="createdA"
      label="作成日"
      sortKey="createdAtA"
      activeKey={sortedKey}
      direction={sortedDir ?? null}
      onChange={(key, dir) => {
        setSortedKey(key as "createdAtA" | "createdAtB");
        setSortedDir(dir);
      }}
    />,
    <SortableTableHeader
      key="createdB"
      label="作成日"
      sortKey="createdAtB"
      activeKey={sortedKey}
      direction={sortedDir ?? null}
      onChange={(key, dir) => {
        setSortedKey(key as "createdAtA" | "createdAtB");
        setSortedDir(dir);
      }}
    />,
  ];

  // 行クリックで詳細ページへ遷移
  const handleRowClick = (r: ProductBlueprintRow) => {
    navigate(
      `/productBlueprint/detail?product=${encodeURIComponent(r.product)}`
    );
  };

  // 作成ボタン押下で作成ページへ遷移
  const handleCreate = () => {
    navigate("/productBlueprint/create");
  };

  return (
    <List
      title="商品設計"
      headerCells={headers}
      showCreateButton
      createLabel="商品設計を作成"
      onCreate={handleCreate}
      showResetButton
      onReset={() => {
        setBrandFilter([]);
        setSortedKey(null);
        setSortedDir(null);
      }}
    >
      {rows.map((r) => (
        <tr
          key={`${r.product}-${r.createdAtA}`}
          className="cursor-pointer hover:bg-[rgba(0,0,0,0.03)] transition"
          onClick={() => handleRowClick(r)}
        >
          <td>{r.product}</td>
          <td>
            <span className="lp-brand-pill">{r.brand}</span>
          </td>
          <td>{r.owner}</td>
          <td>{r.productId}</td>
          <td>{r.createdAtA}</td>
          <td>{r.createdAtB}</td>
        </tr>
      ))}
    </List>
  );
}
