// frontend/transaction/src/pages/transactionList.tsx
import * as React from "react";
import List from "../../../shell/src/layout/List/List";
import { Filter, ArrowDownLeft, ArrowUpRight } from "lucide-react";

// Lucide型エラー対策
const IconFilter = Filter as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;
const IconIn = ArrowDownLeft as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;
const IconOut = ArrowUpRight as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;

type Transaction = {
  datetime: string;
  brand: string;
  type: "受取" | "送金";
  description: string;
  amount: number;
  counterparty: string;
};

const TRANSACTIONS: Transaction[] = [
  { datetime: "2024/3/15 23:32:00", brand: "LUMINA Fashion", type: "受取", description: "商品購入代金", amount: 125000, counterparty: "株式会社○○商事" },
  { datetime: "2024/3/15 22:15:00", brand: "NEXUS Street", type: "送金", description: "サプライヤー支払い", amount: -89000, counterparty: "△△サプライヤー" },
  { datetime: "2024/3/15 20:45:00", brand: "LUMINA Fashion", type: "受取", description: "VIP会員購入", amount: 230000, counterparty: "VIP会員様" },
  { datetime: "2024/3/15 19:20:00", brand: "NEXUS Street", type: "受取", description: "トークン販売", amount: 54000, counterparty: "NFT購入者" },
  { datetime: "2024/3/15 18:10:00", brand: "NEXUS Street", type: "受取", description: "コラボ商品売上", amount: 156000, counterparty: "コラボ先企業" },
  { datetime: "2024/3/15 01:20:00", brand: "LUMINA Fashion", type: "送金", description: "製造委託費用", amount: -175000, counterparty: "製造パートナー" },
  { datetime: "2024/3/15 00:10:00", brand: "LUMINA Fashion", type: "受取", description: "EC売上", amount: 98000, counterparty: "オンラインストア" },
  { datetime: "2024/3/14 23:30:00", brand: "NEXUS Street", type: "送金", description: "広告宣伝費", amount: -65000, counterparty: "広告代理店" },
  { datetime: "2024/3/14 20:25:00", brand: "LUMINA Fashion", type: "受取", description: "店頭売上", amount: 315000, counterparty: "百貨店" },
  { datetime: "2024/3/14 19:15:00", brand: "NEXUS Street", type: "送金", description: "材料仕入れ", amount: -42000, counterparty: "素材サプライヤー" },
  { datetime: "2024/3/14 18:00:00", brand: "LUMINA Fashion", type: "受取", description: "キャンペーン売上", amount: 76000, counterparty: "直営店" },
  { datetime: "2024/3/14 17:40:00", brand: "NEXUS Street", type: "送金", description: "配送費用", amount: -12000, counterparty: "配送会社" },
];

export default function TransactionListPage() {
  const headers: React.ReactNode[] = [
    "日時",
    <>
      <span className="inline-flex items-center gap-2">
        ブランド
        <button className="lp-th-filter" aria-label="ブランドを絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
    <>
      <span className="inline-flex items-center gap-2">
        種別
        <button className="lp-th-filter" aria-label="種別を絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
    "説明",
    "金額",
    <>
      <span className="inline-flex items-center gap-2">
        取引先
        <button className="lp-th-filter" aria-label="取引先を絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
  ];

  const renderTypeBadge = (type: Transaction["type"]) => {
    const isReceive = type === "受取";
    const color = isReceive ? "#0a8a4b" : "#d72e2e";
    const bg = isReceive ? "#e6f9ee" : "#ffecec";
    const Icon = isReceive ? IconIn : IconOut;

    return (
      <span
        style={{
          display: "inline-flex",
          alignItems: "center",
          gap: 6,
          background: bg,
          color,
          padding: "0.2rem 0.6rem",
          borderRadius: 9999,
          fontWeight: 600,
          fontSize: "0.85rem",
        }}
      >
        <Icon width={16} height={16} />
        {type}
      </span>
    );
  };

  const renderAmount = (amt: number) => {
    const isPlus = amt >= 0;
    const n = Math.abs(amt).toLocaleString();
    return (
      <span
        style={{
          fontWeight: 700,
          color: isPlus ? "#0a8a4b" : "#d72e2e",
        }}
      >
        {isPlus ? "+" : "-"}
        {n} 円
      </span>
    );
  };

  return (
    <div className="p-0">
      <List
        title="取引履歴"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={() => console.log("取引履歴更新")}
      >
        {TRANSACTIONS.map((t, idx) => {
          const [date, time] = t.datetime.split(" ");
          return (
            <tr key={idx}>
              <td>
                <div>{date}</div>
                <div style={{ fontSize: "0.85rem", color: "#6b7280" }}>{time}</div>
              </td>
              <td>
                <span className="lp-brand-pill">{t.brand}</span>
              </td>
              <td>{renderTypeBadge(t.type)}</td>
              <td>{t.description}</td>
              <td>{renderAmount(t.amount)}</td>
              <td>{t.counterparty}</td>
            </tr>
          );
        })}
      </List>
    </div>
  );
}
