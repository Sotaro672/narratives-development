// frontend/ad/src/pages/adManagement.tsx
import * as React from "react";
import List from "../../../shell/src/layout/List/List";
import { Filter } from "lucide-react";

// lucide-react の型ずれ対策用キャスト
const IconFilter = Filter as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;

type AdRow = {
  campaign: string;
  brand: string;
  owner: string;
  format: string;
  period: string;
  status: "実行中";
  spendRate: string; // "66.0%" など
  spend: string;     // "¥198,000"
  budget: string;    // "¥300,000"
};

const ADS: AdRow[] = [
  {
    campaign: "NEXUS Street デニムジャケット新作",
    brand: "NEXUS Street",
    owner: "渡辺 花子",
    format: "動画広告",
    period: "2024/3/10 - 2024/4/10",
    status: "実行中",
    spendRate: "66.0%",
    spend: "¥198,000",
    budget: "¥300,000",
  },
  {
    campaign: "LUMINA Fashion 春コレクション",
    brand: "LUMINA Fashion",
    owner: "佐藤 美咲",
    format: "カルーセル広告",
    period: "2024/3/1 - 2024/3/31",
    status: "実行中",
    spendRate: "68.4%",
    spend: "¥342,000",
    budget: "¥500,000",
  },
];

export default function AdManagementPage() {
  const [sortAsc, setSortAsc] = React.useState(false);

  const rows = React.useMemo(() => {
    const data = [...ADS];
    // 期間の開始日でソート（YYYY/M/D 形式前提）
    const startDate = (p: string) => p.split("-")[0].trim();
    data.sort((a, b) =>
      sortAsc
        ? startDate(a.period).localeCompare(startDate(b.period))
        : startDate(b.period).localeCompare(startDate(a.period))
    );
    return data;
  }, [sortAsc]);

  const headers: React.ReactNode[] = [
    "キャンペーン名",
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
        <span>担当者</span>
        <button className="lp-th-filter" aria-label="担当者で絞り込む">
          <IconFilter width={16} height={16} />
        </button>
      </span>
    </>,
    "広告形式",
    <>
      <span className="inline-flex items-center gap-2">
        <span>広告期間</span>
        <button
          className="lp-th-filter"
          aria-label={`広告期間でソート（${sortAsc ? "昇順" : "降順"}）`}
          onClick={() => setSortAsc((v) => !v)}
        >
          {sortAsc ? "▲" : "▼"}
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
    "消化率",
    "消化",
    "予算",
  ];

  return (
    <div className="p-0">
      <List
        title="広告管理"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={() => console.log("広告一覧を更新")}
      >
        {rows.map((ad) => (
          <tr key={ad.campaign}>
            <td>{ad.campaign}</td>
            <td>
              <span className="lp-brand-pill">{ad.brand}</span>
            </td>
            <td>{ad.owner}</td>
            <td>{ad.format}</td>
            <td>{ad.period}</td>
            <td>
              <span
                style={{
                  display: "inline-block",
                  background: "#d1fae5",
                  color: "#065f46",
                  fontSize: "0.75rem",
                  fontWeight: 700,
                  padding: "0.25rem 0.6rem",
                  borderRadius: 9999,
                }}
              >
                {ad.status}
              </span>
            </td>
            <td>{ad.spendRate}</td>
            <td>{ad.spend}</td>
            <td>{ad.budget}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
