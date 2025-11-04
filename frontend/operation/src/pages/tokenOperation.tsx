// frontend/operation/src/pages/tokenOperation.tsx
import { useMemo, useState } from "react";
import List from "../../../shell/src/layout/List/List";
import { Filter } from "lucide-react";

type TokenOperation = {
  tokenName: string;
  symbol: string;
  brand: string;
  linkedProducts: number;
  manager: string;
  planned: number;
  requested: number;
  issued: number;
  distributionRate: string;
};

const TOKEN_OPERATIONS: TokenOperation[] = [
  {
    tokenName: "LUMINA VIP Token",
    symbol: "LVIP",
    brand: "LUMINA Fashion",
    linkedProducts: 1,
    manager: "山田 太郎",
    planned: 0,
    requested: 0,
    issued: 10,
    distributionRate: "100.0%",
  },
  {
    tokenName: "SILK Premium Token",
    symbol: "SILK",
    brand: "LUMINA Fashion",
    linkedProducts: 1,
    manager: "佐藤 美咲",
    planned: 10,
    requested: 0,
    issued: 0,
    distributionRate: "0.0%",
  },
  {
    tokenName: "NEXUS Community Token",
    symbol: "NXCOM",
    brand: "NEXUS Street",
    linkedProducts: 1,
    manager: "佐藤 美咲",
    planned: 0,
    requested: 0,
    issued: 12,
    distributionRate: "100.0%",
  },
  {
    tokenName: "NEXUS Street Token",
    symbol: "NEXUS",
    brand: "NEXUS Street",
    linkedProducts: 1,
    manager: "高橋 健太",
    planned: 0,
    requested: 5,
    issued: 0,
    distributionRate: "0.0%",
  },
  {
    tokenName: "SILK Limited Edition",
    symbol: "SLKED",
    brand: "LUMINA Fashion",
    linkedProducts: 0,
    manager: "高橋 健太",
    planned: 0,
    requested: 0,
    issued: 0,
    distributionRate: "0.0%",
  },
];

export default function TokenOperationPage() {
  const [sortKey, setSortKey] = useState<keyof TokenOperation>("tokenName");
  const [sortAsc, setSortAsc] = useState(true);

  const sortedRows = useMemo(() => {
    const data = [...TOKEN_OPERATIONS];
    data.sort((a, b) => {
      const valA = a[sortKey];
      const valB = b[sortKey];
      if (typeof valA === "number" && typeof valB === "number")
        return sortAsc ? valA - valB : valB - valA;
      return sortAsc
        ? String(valA).localeCompare(String(valB))
        : String(valB).localeCompare(String(valA));
    });
    return data;
  }, [sortKey, sortAsc]);

  const headers = [
    "トークン名",
    "シンボル",
    <>
      <span className="inline-flex items-center gap-2">
        <span>ブランド</span>
        <button
          className="lp-th-filter"
          aria-label="ブランドで絞り込む"
          onClick={() => setSortKey("brand")}
        >
          <Filter size={16} />
        </button>
      </span>
    </>,
    "連携商品種類数",
    <>
      <span className="inline-flex items-center gap-2">
        <span>担当者</span>
        <button
          className="lp-th-filter"
          aria-label="担当者で絞り込む"
          onClick={() => setSortKey("manager")}
        >
          <Filter size={16} />
        </button>
      </span>
    </>,
    "計画量",
    "申請量",
    "発行量",
    "配布率",
  ];

  return (
    <div className="p-0">
      <List
        title="トークン運用"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={() => console.log("リスト更新")}
      >
        {sortedRows.map((t, i) => (
          <tr key={i}>
            <td>{t.tokenName}</td>
            <td>{t.symbol}</td>
            <td>
              <span className="lp-brand-pill">{t.brand}</span>
            </td>
            <td>{t.linkedProducts}</td>
            <td>{t.manager}</td>
            <td>{t.planned}</td>
            <td>{t.requested}</td>
            <td>{t.issued}</td>
            <td>{t.distributionRate}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
