// frontend/list/src/pages/listManagement.tsx
import * as React from "react";
import List from "../../../shell/src/layout/List/List";
import { Filter } from "lucide-react";

// Lucideアイコンの型をReact汎用コンポーネントにキャスト（TS2786対策）
const IconFilter = Filter as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;

type ListingRow = {
  id: string;
  product: string;
  brand: string;
  token: string;
  stock: number;
  manager: string;
  status: "出品中" | "停止中";
};

const LISTINGS: ListingRow[] = [
  {
    id: "list_001",
    product: "シルクブラウス プレミアムライン",
    brand: "LUMINA Fashion",
    token: "LUMINA VIP Token",
    stock: 221,
    manager: "山田 太郎",
    status: "出品中",
  },
  {
    id: "list_002",
    product: "デニムジャケット ヴィンテージ加工",
    brand: "NEXUS Street",
    token: "NEXUS Community Token",
    stock: 222,
    manager: "佐藤 美咲",
    status: "出品中",
  },
  {
    id: "list_003",
    product: "シルクブラウス プレミアムライン",
    brand: "LUMINA Fashion",
    token: "LUMINA VIP Token",
    stock: 221,
    manager: "山田 太郎",
    status: "停止中",
  },
];

export default function ListManagementPage() {
  const [sortAsc, setSortAsc] = React.useState(true);

  const rows = React.useMemo(() => {
    const data = [...LISTINGS];
    data.sort((a, b) =>
      sortAsc ? a.id.localeCompare(b.id) : b.id.localeCompare(a.id)
    );
    return data;
  }, [sortAsc]);

  const headers: React.ReactNode[] = [
    <>
      <span className="inline-flex items-center gap-2">
        <span>出品ID</span>
        <button
          className="lp-th-filter"
          aria-label="出品IDでソート"
          onClick={() => setSortAsc((v) => !v)}
        >
          {sortAsc ? "▲" : "▼"}
        </button>
      </span>
    </>,
    <>
      <span className="inline-flex items-center gap-2">
        <span>プロダクト</span>
        <button className="lp-th-filter" aria-label="プロダクトで絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
    <>
      <span className="inline-flex items-center gap-2">
        <span>ブランド</span>
        <button className="lp-th-filter" aria-label="ブランドで絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
    <>
      <span className="inline-flex items-center gap-2">
        <span>トークン</span>
        <button className="lp-th-filter" aria-label="トークンで絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
    "総在庫数",
    <>
      <span className="inline-flex items-center gap-2">
        <span>担当者</span>
        <button className="lp-th-filter" aria-label="担当者で絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
    <>
      <span className="inline-flex items-center gap-2">
        <span>ステータス</span>
        <button className="lp-th-filter" aria-label="ステータスで絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
  ];

  return (
    <div className="p-0">
      <List
        title="出品管理"
        headerCells={headers}
        showCreateButton
        createLabel="出品を作成"
        showResetButton
        onReset={() => console.log("出品リスト更新")}
      >
        {rows.map((l) => (
          <tr key={l.id}>
            <td>{l.id}</td>
            <td>{l.product}</td>
            <td>
              <span className="lp-brand-pill">{l.brand}</span>
            </td>
            <td>
              <span className="lp-brand-pill">{l.token}</span>
            </td>
            <td>
              <span
                style={{
                  display: "inline-block",
                  minWidth: 36,
                  background: "#0b0f1a",
                  color: "#fff",
                  fontWeight: 600,
                  textAlign: "center",
                  borderRadius: 12,
                  padding: "0.2rem 0.6rem",
                }}
              >
                {l.stock}
              </span>
            </td>
            <td>{l.manager}</td>
            <td>
              {l.status === "出品中" ? (
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
                  出品中
                </span>
              ) : (
                <span
                  style={{
                    display: "inline-block",
                    background: "#ef4444",
                    color: "#fff",
                    fontSize: "0.75rem",
                    fontWeight: 700,
                    padding: "0.3rem 0.6rem",
                    borderRadius: 9999,
                  }}
                >
                  停止中
                </span>
              )}
            </td>
          </tr>
        ))}
      </List>
    </div>
  );
}
