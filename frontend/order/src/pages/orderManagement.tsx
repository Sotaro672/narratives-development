// frontend/order/src/pages/orderManagement.tsx
import * as React from "react";
import List from "../../../shell/src/layout/List/List";
import { Filter } from "lucide-react";

// LucideアイコンをReactコンポーネントにキャスト（型エラー対策）
const IconFilter = Filter as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;

type OrderRow = {
  id: string;
  customerName: string;
  customerEmail: string;
  productName: string;
  additionalItems: string;
  status: "支払済" | "移譲完了";
  paymentMethod: string;
  amount: string;
  quantityInfo: string;
  purchaseLocation: string;
  orderDate: string;
};

const ORDERS: OrderRow[] = [
  {
    id: "ORD-2024-0002",
    customerName: "山本 由紀",
    customerEmail: "yuki.yamamoto@example.com",
    productName: "デニムジャケット ヴィンテージ加工",
    additionalItems: "他1点",
    status: "支払済",
    paymentMethod: "デジタルウォレット",
    amount: "¥32,700",
    quantityInfo: "3点",
    purchaseLocation: "オンライン",
    orderDate: "2024/3/21",
  },
  {
    id: "ORD-2024-0001",
    customerName: "Creator Alice",
    customerEmail: "alice@example.com",
    productName: "シルクブラウス プレミアムライン",
    additionalItems: "他1点",
    status: "移譲完了",
    paymentMethod: "クレジットカード",
    amount: "¥45,900",
    quantityInfo: "2点",
    purchaseLocation: "オンライン",
    orderDate: "2024/3/20",
  },
];

export default function OrderManagementPage() {
  const [sortAsc, setSortAsc] = React.useState(false);

  const rows = React.useMemo(() => {
    const data = [...ORDERS];
    data.sort((a, b) =>
      sortAsc
        ? a.orderDate.localeCompare(b.orderDate)
        : b.orderDate.localeCompare(a.orderDate)
    );
    return data;
  }, [sortAsc]);

  const headers: React.ReactNode[] = [
    <>
      <span className="inline-flex items-center gap-2">
        <span>注文番号</span>
        <button
          className="lp-th-filter"
          aria-label="注文番号でソート"
          onClick={() => setSortAsc((v) => !v)}
        >
          {sortAsc ? "▲" : "▼"}
        </button>
      </span>
    </>,
    "お客様",
    "商品",
    <>
      <span className="inline-flex items-center gap-2">
        <span>ステータス</span>
        <button className="lp-th-filter" aria-label="ステータスで絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
    <>
      <span className="inline-flex items-center gap-2">
        <span>支払方法</span>
        <button className="lp-th-filter" aria-label="支払方法で絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
    "金額",
    <>
      <span className="inline-flex items-center gap-2">
        <span>購入場所</span>
        <button className="lp-th-filter" aria-label="購入場所で絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
    "注文日",
  ];

  return (
    <div className="p-0">
      <List
        title="注文管理"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={() => console.log("注文一覧を更新")}
      >
        {rows.map((o) => (
          <tr key={o.id}>
            <td>{o.id}</td>
            <td>
              <div className="font-medium">{o.customerName}</div>
              <div style={{ color: "#6b7280", fontSize: "0.85rem" }}>
                {o.customerEmail}
              </div>
            </td>
            <td>
              <div>{o.productName}</div>
              <div style={{ color: "#6b7280", fontSize: "0.85rem" }}>
                {o.additionalItems}
              </div>
            </td>
            <td>
              {o.status === "支払済" ? (
                <span
                  style={{
                    display: "inline-block",
                    background: "#e5e7eb",
                    color: "#111827",
                    fontSize: "0.75rem",
                    fontWeight: 700,
                    padding: "0.3rem 0.6rem",
                    borderRadius: 9999,
                  }}
                >
                  支払済
                </span>
              ) : (
                <span
                  style={{
                    display: "inline-block",
                    background: "#0b0f1a",
                    color: "#fff",
                    fontSize: "0.75rem",
                    fontWeight: 700,
                    padding: "0.3rem 0.6rem",
                    borderRadius: 9999,
                  }}
                >
                  移譲完了
                </span>
              )}
            </td>
            <td>{o.paymentMethod}</td>
            <td>
              <div className="font-medium">{o.amount}</div>
              <div style={{ color: "#6b7280", fontSize: "0.85rem" }}>
                {o.quantityInfo}
              </div>
            </td>
            <td>{o.purchaseLocation}</td>
            <td>{o.orderDate}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
