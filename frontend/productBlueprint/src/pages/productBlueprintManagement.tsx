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
  createdAtA: string; // 例: "2024/1/15"
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

  // ソート状態を 1 組に統一
  const [sortedKey, setSortedKey] = useState<"createdAtA" | "createdAtB" | null>(null);
  const [sortedDir, setSortedDir] = useState<"asc" | "desc" | null>(null);

  // フィルタ → ソート
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

    // 作成日A（左）
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

    // 作成日B（右）
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
        setSortedKey(null);
        setSortedDir(null);
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
