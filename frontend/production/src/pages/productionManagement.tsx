// frontend/production/src/pages/productionManagement.tsx
import React, { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";

type Production = {
  id: string;
  product: string;
  brand: string;
  manager: string;
  quantity: number;
  productId: string;
  printedAt: string; // YYYY/M/D or "-"
  createdAt: string; // YYYY/M/D
};

const PRODUCTIONS: Production[] = [
  {
    id: "production_001",
    product: "シルクブラウス プレミアムライン",
    brand: "LUMINA Fashion",
    manager: "佐藤 美咲",
    quantity: 10,
    productId: "QR",
    printedAt: "2025/11/3",
    createdAt: "2025/11/5",
  },
  {
    id: "production_002",
    product: "デニムジャケット ヴィンテージ加工",
    brand: "NEXUS Street",
    manager: "高橋 健太",
    quantity: 9,
    productId: "QR",
    printedAt: "2025/11/4",
    createdAt: "2025/11/5",
  },
  {
    id: "production_003",
    product: "シルクブラウス プレミアムライン",
    brand: "LUMINA Fashion",
    manager: "佐藤 美咲",
    quantity: 7,
    productId: "QR",
    printedAt: "2025/11/1",
    createdAt: "2025/10/31",
  },
  {
    id: "production_004",
    product: "デニムジャケット ヴィンテージ加工",
    brand: "NEXUS Street",
    manager: "高橋 健太",
    quantity: 4,
    productId: "QR",
    printedAt: "2025/10/30",
    createdAt: "2025/10/29",
  },
  {
    id: "production_005",
    product: "シルクブラウス プレミアムライン",
    brand: "LUMINA Fashion",
    manager: "佐藤 美咲",
    quantity: 2,
    productId: "QR",
    printedAt: "-",
    createdAt: "2025/11/4",
  },
];

// 日付を数値へ変換（"-" は 0 として最小扱い）
const toTs = (yyyyMd: string) => {
  if (!yyyyMd || yyyyMd === "-") return 0;
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

export default function ProductionManagement() {
  const navigate = useNavigate();

  // ===== フィルタ状態 =====
  const [productFilter, setProductFilter] = useState<string[]>([]);
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [managerFilter, setManagerFilter] = useState<string[]>([]);
  const [productIdFilter, setProductIdFilter] = useState<string[]>([]);

  // ===== ソート状態 =====
  type SortKey = "printedAt" | "createdAt" | "quantity" | null;
  const [sortKey, setSortKey] = useState<SortKey>(null);
  const [sortDir, setSortDir] = useState<"asc" | "desc" | null>(null);

  // ===== オプション =====
  const productOptions = useMemo(
    () =>
      Array.from(new Set(PRODUCTIONS.map((p) => p.product))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );
  const brandOptions = useMemo(
    () =>
      Array.from(new Set(PRODUCTIONS.map((p) => p.brand))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );
  const managerOptions = useMemo(
    () =>
      Array.from(new Set(PRODUCTIONS.map((p) => p.manager))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );
  const productIdOptions = useMemo(
    () =>
      Array.from(new Set(PRODUCTIONS.map((p) => p.productId))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );

  // ===== データ生成（フィルタ → ソート） =====
  const rows = useMemo(() => {
    let data = PRODUCTIONS.filter(
      (p) =>
        (productFilter.length === 0 || productFilter.includes(p.product)) &&
        (brandFilter.length === 0 || brandFilter.includes(p.brand)) &&
        (managerFilter.length === 0 || managerFilter.includes(p.manager)) &&
        (productIdFilter.length === 0 || productIdFilter.includes(p.productId))
    );

    if (sortKey && sortDir) {
      data = [...data].sort((a, b) => {
        if (sortKey === "quantity") {
          const av = a.quantity;
          const bv = b.quantity;
          return sortDir === "asc" ? av - bv : bv - av;
        }
        const av = toTs(a[sortKey]);
        const bv = toTs(b[sortKey]);
        return sortDir === "asc" ? av - bv : bv - av;
      });
    }

    return data;
  }, [productFilter, brandFilter, managerFilter, productIdFilter, sortKey, sortDir]);

  // ===== ヘッダー =====
  const headers: React.ReactNode[] = [
    "生産計画",
    <FilterableTableHeader
      key="product"
      label="プロダクト"
      options={productOptions}
      selected={productFilter}
      onChange={setProductFilter}
    />,
    <FilterableTableHeader
      key="brand"
      label="ブランド"
      options={brandOptions}
      selected={brandFilter}
      onChange={setBrandFilter}
    />,
    <FilterableTableHeader
      key="manager"
      label="担当者"
      options={managerOptions}
      selected={managerFilter}
      onChange={setManagerFilter}
    />,
    <SortableTableHeader
      key="quantity"
      label="生産数"
      sortKey="quantity"
      activeKey={sortKey}
      direction={sortDir ?? null}
      onChange={(key, dir) => {
        setSortKey(key as SortKey);
        setSortDir(dir);
      }}
    />,
    <FilterableTableHeader
      key="productId"
      label="商品ID"
      options={productIdOptions}
      selected={productIdFilter}
      onChange={setProductIdFilter}
    />,
    <SortableTableHeader
      key="printedAt"
      label="印刷日"
      sortKey="printedAt"
      activeKey={sortKey}
      direction={sortDir ?? null}
      onChange={(key, dir) => {
        setSortKey(key as SortKey);
        setSortDir(dir);
      }}
    />,
    <SortableTableHeader
      key="createdAt"
      label="作成日"
      sortKey="createdAt"
      activeKey={sortKey}
      direction={sortDir ?? null}
      onChange={(key, dir) => {
        setSortKey(key as SortKey);
        setSortDir(dir);
      }}
    />,
  ];

  return (
    <div className="p-0">
      <List
        title="商品生産"
        headerCells={headers}
        showCreateButton
        createLabel="生産計画を作成"
        showResetButton
        onCreate={() => console.log("新規生産計画作成")}
        onReset={() => {
          setProductFilter([]);
          setBrandFilter([]);
          setManagerFilter([]);
          setProductIdFilter([]);
          setSortKey(null);
          setSortDir(null);
          console.log("リスト更新");
        }}
      >
        {rows.map((p) => (
          <tr
            key={p.id}
            className="cursor-pointer hover:bg-blue-50 transition-colors"
            onClick={() => navigate(`/production/${p.id}`)} // ← クリックで詳細へ遷移
          >
            <td className="text-blue-600 underline">{p.id}</td>
            <td>{p.product}</td>
            <td>
              <span className="lp-brand-pill">{p.brand}</span>
            </td>
            <td>{p.manager}</td>
            <td>{p.quantity}</td>
            <td>
              <span className="lp-brand-pill">{p.productId}</span>
            </td>
            <td>{p.printedAt}</td>
            <td>{p.createdAt}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
