// frontend/productBlueprint/src/pages/productBlueprintManagement.tsx
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";
import { useMemo, useState } from "react";

type Row = {
  product: string;
  brand: "LUMINA Fashion" | "NEXUS Street";
  owner: string;
  productId: string;
  createdAtA: string;
  createdAtB: string;
};

const RAW_ROWS: Row[] = [
  {
    product: "シルクブラウス プレミアムライン",
    brand: "LUMINA Fashion",
    owner: "佐藤 美咲",
    productId: "QR",
    createdAtA: "2024/1/15",
    createdAtB: "2024/1/15",
  },
  {
    product: "デニムジャケット ヴィンテージ加工",
    brand: "NEXUS Street",
    owner: "高橋 健太",
    productId: "QR",
    createdAtA: "2024/1/10",
    createdAtB: "2024/1/10",
  },
];

const toTs = (yyyyMd: string) => {
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

export default function ProductBlueprintManagement() {
  // フィルタ（ブランド）
  const [brandFilter, setBrandFilter] = useState<string[]>([]);

  // ソート方向
  const [createdADirection, setCreatedADirection] = useState<"asc" | "desc" | undefined>(
    undefined
  );
  const [createdBDirection, setCreatedBDirection] = useState<"asc" | "desc" | undefined>(
    undefined
  );

  // フィルタ → ソート
  const rows = useMemo(() => {
    let work = RAW_ROWS.filter(
      (r) => brandFilter.length === 0 || brandFilter.includes(r.brand)
    );

    if (createdADirection) {
      work = [...work].sort((a, b) => {
        const da = toTs(a.createdAtA);
        const db = toTs(b.createdAtA);
        return createdADirection === "asc" ? da - db : db - da;
      });
    } else if (createdBDirection) {
      work = [...work].sort((a, b) => {
        const da = toTs(a.createdAtB);
        const db = toTs(b.createdAtB);
        return createdBDirection === "asc" ? da - db : db - da;
      });
    }
    return work;
  }, [brandFilter, createdADirection, createdBDirection]);

  const headers = [
    "プロダクト",

    // ▼ filterable-table-header の Option は { value, label }
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={[
        { value: "LUMINA Fashion", label: "LUMINA Fashion" },
        { value: "NEXUS Street", label: "NEXUS Street" },
      ]}
      selectedValues={brandFilter}
      onChange={(values) => setBrandFilter(values)}
    />,

    "担当者",
    "商品ID",

    // ▼ sortable-table-header は { direction, onDirectionChange }
    <SortableTableHeader
      key="createdA"
      label="作成日"
      direction={createdADirection}
      onDirectionChange={(dir: "asc" | "desc" | undefined) => {
        setCreatedADirection(dir);
        if (dir) setCreatedBDirection(undefined);
      }}
    />,

    <SortableTableHeader
      key="createdB"
      label="作成日"
      direction={createdBDirection}
      onDirectionChange={(dir: "asc" | "desc" | undefined) => {
        setCreatedBDirection(dir);
        if (dir) setCreatedADirection(undefined);
      }}
    />,
  ];

  return (
    <List
      title="商品設計"
      headerCells={headers}
      showCreateButton
      createLabel="商品設計を作成"
      onCreate={() => console.log("create")}
      showResetButton
      onReset={() => {
        setBrandFilter([]);
        setCreatedADirection(undefined);
        setCreatedBDirection(undefined);
        console.log("reset");
      }}
    >
      {rows.map((r) => (
        <tr key={`${r.product}-${r.createdAtA}`}>
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
