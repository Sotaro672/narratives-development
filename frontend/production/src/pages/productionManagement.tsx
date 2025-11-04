// frontend/production/src/pages/productionManagement.tsx
import List from "../../../shell/src/layout/List/List";
import { Filter } from "lucide-react";

type Production = {
  id: string;
  product: string;
  brand: string;
  manager: string;
  quantity: number;
  productId: string;
  printedAt: string;
  createdAt: string;
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

export default function ProductionManagement() {
  const headers = [
    "生産計画",
    <>
      <span className="inline-flex items-center gap-2">
        <span>プロダクト</span>
        <button className="lp-th-filter" aria-label="プロダクトで絞り込む">
          <Filter size={16} />
        </button>
      </span>
    </>,
    <>
      <span className="inline-flex items-center gap-2">
        <span>ブランド</span>
        <button className="lp-th-filter" aria-label="ブランドで絞り込む">
          <Filter size={16} />
        </button>
      </span>
    </>,
    <>
      <span className="inline-flex items-center gap-2">
        <span>担当者</span>
        <button className="lp-th-filter" aria-label="担当者で絞り込む">
          <Filter size={16} />
        </button>
      </span>
    </>,
    <>
      <span className="inline-flex items-center gap-2">
        <span>生産数</span>
        <button className="lp-th-filter" aria-label="生産数で絞り込む">
          <Filter size={16} />
        </button>
      </span>
    </>,
    <>
      <span className="inline-flex items-center gap-2">
        <span>商品ID</span>
        <button className="lp-th-filter" aria-label="商品IDで絞り込む">
          <Filter size={16} />
        </button>
      </span>
    </>,
    "印刷日",
    "作成日",
  ];

  return (
    <div className="p-0">
      <List
        title="商品生産"
        headerCells={headers}
        showCreateButton={true}
        createLabel="生産計画を作成"
        showResetButton={true}
        onCreate={() => console.log("新規生産計画作成")}
        onReset={() => console.log("リスト更新")}
      >
        {PRODUCTIONS.map((p) => (
          <tr key={p.id}>
            <td>
              <a href="#" className="text-blue-600 hover:underline">
                {p.id}
              </a>
            </td>
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
