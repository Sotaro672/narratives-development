// frontend/inventory/src/pages/inventoryManagement.tsx
import List from "../../../shell/src/layout/List/List";
import { Filter } from "lucide-react";

type InventoryRow = {
  product: string;
  brand: string;
  token: string;
  total: number;
};

const INVENTORIES: InventoryRow[] = [
  {
    product: "シルクブラウス プレミアムライン",
    brand: "LUMINA Fashion",
    token: "LUMINA VIP Token",
    total: 221,
  },
  {
    product: "デニムジャケット ヴィンテージ加工",
    brand: "NEXUS Street",
    token: "NEXUS Community Token",
    total: 222,
  },
];

export default function InventoryManagementPage() {
  const headers = [
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
        <span>トークン</span>
        <button className="lp-th-filter" aria-label="トークンで絞り込む">
          <Filter size={16} />
        </button>
      </span>
    </>,
    "総在庫数",
  ];

  return (
    <div className="p-0">
      <List
        title="在庫管理"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={() => console.log("在庫リストを更新")}
      >
        {INVENTORIES.map((row, i) => (
          <tr key={i}>
            <td>{row.product}</td>
            <td>{row.brand}</td>
            <td>
              <span className="lp-brand-pill">{row.token}</span>
            </td>
            <td>
              {/* 濃色ピル風（専用CSSが無ければインラインで近似） */}
              <span
                style={{
                  display: "inline-block",
                  background: "#0b0f1a",
                  color: "#fff",
                  fontSize: "0.8rem",
                  fontWeight: 600,
                  padding: "0.2rem 0.6rem",
                  borderRadius: "9999px",
                  minWidth: "2.25rem",
                  textAlign: "center",
                }}
              >
                {row.total}
              </span>
            </td>
          </tr>
        ))}
      </List>
    </div>
  );
}
