// frontend/tokenBlueprint/src/pages/tokenBlueprintManagement.tsx
import { useMemo, useState } from "react";
import List from "../../../shell/src/layout/List/List";
import { Filter, ChevronDown, ChevronUp } from "lucide-react";

type TokenBlueprint = {
  name: string;
  symbol: string;
  brand: string;
  manager: string;
  createdAt: string; // YYYY/M/D
};

const TOKEN_BLUEPRINTS: TokenBlueprint[] = [
  { name: "SILK Premium Token",   symbol: "SILK",  brand: "LUMINA Fashion",  manager: "佐藤 美咲", createdAt: "2024/1/20" },
  { name: "NEXUS Street Token",   symbol: "NEXUS", brand: "NEXUS Street",    manager: "高橋 健太", createdAt: "2024/1/18" },
  { name: "LUMINA VIP Token",     symbol: "LVIP",  brand: "LUMINA Fashion",  manager: "山田 太郎", createdAt: "2024/1/15" },
  { name: "NEXUS Community Token",symbol: "NXCOM", brand: "NEXUS Street",    manager: "佐藤 美咲", createdAt: "2024/1/12" },
  { name: "SILK Limited Edition", symbol: "SLKED", brand: "LUMINA Fashion",  manager: "高橋 健太", createdAt: "2024/1/10" },
];

export default function TokenBlueprintManagementPage() {
  // 作成日の昇降ソート
  const [sortAsc, setSortAsc] = useState(false);

  const rows = useMemo(() => {
    const data = [...TOKEN_BLUEPRINTS];
    data.sort((a, b) =>
      sortAsc
        ? a.createdAt.localeCompare(b.createdAt)
        : b.createdAt.localeCompare(a.createdAt)
    );
    return data;
  }, [sortAsc]);

  const headers = [
    "トークン名",
    "シンボル",
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
        <span>作成日</span>
        <button
          className="lp-th-filter"
          aria-label={`作成日でソート（${sortAsc ? "昇順" : "降順"}）`}
          onClick={() => setSortAsc((v) => !v)}
        >
          {sortAsc ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
        </button>
      </span>
    </>,
  ];

  return (
    <div className="p-0">
      <List
        title="トークン設計"
        headerCells={headers}
        showCreateButton
        createLabel="トークン設計を作成"
        showResetButton
        onCreate={() => console.log("トークン設計の新規作成")}
        onReset={() => console.log("リスト更新")}
      >
        {rows.map((t) => (
          <tr key={`${t.symbol}-${t.createdAt}`}>
            <td>{t.name}</td>
            <td>{t.symbol}</td>
            <td>
              <span className="lp-brand-pill">{t.brand}</span>
            </td>
            <td>{t.manager}</td>
            <td>{t.createdAt}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
