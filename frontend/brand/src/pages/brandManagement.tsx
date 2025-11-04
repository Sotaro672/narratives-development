// frontend/brand/src/pages/brandManagement.tsx
import { useMemo, useState } from "react";
import List from "../../../shell/src/layout/List/List";
import { Filter, ChevronDown, ChevronUp } from "lucide-react";

type BrandRow = {
  name: string;
  status: "active" | "inactive";
  owner: string;
  registeredAt: string; // YYYY/M/D 表示用
};

// ダミーデータ（必要に応じてAPI結果に置き換え）
const ALL_BRANDS: BrandRow[] = [
  { name: "NEXUS Street",    status: "active",   owner: "渡辺 花子", registeredAt: "2024/2/1" },
  { name: "LUMINA Fashion",  status: "active",   owner: "佐藤 美咲", registeredAt: "2024/1/1" },
];

export default function BrandManagementPage() {
  // 登録日の昇降ソート
  const [sortAsc, setSortAsc] = useState<boolean>(false);

  const sorted = useMemo(() => {
    const data = [...ALL_BRANDS];
    // 文字列日付の単純比較（表示例に合わせて文字列のまま）
    data.sort((a, b) => (sortAsc ? a.registeredAt.localeCompare(b.registeredAt) : b.registeredAt.localeCompare(a.registeredAt)));
    return data;
  }, [sortAsc]);

  const headers = [
    "ブランド名",
    <>
      <span>ステータス</span>
      <button className="lp-th-filter" aria-label="ステータスで絞り込む">
        <Filter size={16} />
      </button>
    </>,
    "責任者",
    <>
      <span>登録日</span>
      <button
        className="lp-th-filter"
        aria-label={`登録日でソート（${sortAsc ? "昇順" : "降順"}）`}
        onClick={() => setSortAsc((v) => !v)}
      >
        {sortAsc ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
      </button>
    </>,
  ];

  return (
    <div className="p-0">
      <List
        title="ブランド管理"
        headerCells={headers}
        showCreateButton
        createLabel="ブランド追加"
        onCreate={() => console.log("ブランド追加")}
        showResetButton
        onReset={() => console.log("リセット")}
      >
        {sorted.map((b) => (
          <tr key={b.name}>
            <td>{b.name}</td>
            <td>
              {/* ピル表示（既存の薄青ピルを流用） */}
              <span className="lp-brand-pill">
                {b.status === "active" ? "アクティブ" : "停止"}
              </span>
            </td>
            <td>{b.owner}</td>
            <td>{b.registeredAt}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
