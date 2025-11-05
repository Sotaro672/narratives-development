// frontend/inquiry/src/pages/InquiryManagementPage.tsx
import * as React from "react";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";

type InquiryRow = {
  id: string;
  title: string;
  body: string;
  user: string;
  status: "対応中" | "未対応";
  type: "商品説明" | "交換";
  owner: string;
  inquiredAt: string; // YYYY/M/D
  answeredAt: string; // YYYY/M/D or "-"
};

const INQUIRIES: InquiryRow[] = [
  {
    id: "inquiry_002",
    title: "デニムジャケットの色落ちについて",
    body: "NEXUS Streetのデニムジャケットを洗濯したら色落ちし…",
    user: "Style Yuki",
    status: "対応中",
    type: "商品説明",
    owner: "田中 雄太",
    inquiredAt: "2024/9/24",
    answeredAt: "2024/9/25",
  },
  {
    id: "inquiry_001",
    title: "シルクブラウスのサイズ交換について",
    body: "LUMINA Fashionのプレミアムシルクブラウスを購入しまし…",
    user: "Creator Alice",
    status: "未対応",
    type: "交換",
    owner: "佐藤 美咲",
    inquiredAt: "2024/9/20",
    answeredAt: "2024/9/20",
  },
];

// "YYYY/M/D" → number（タイムスタンプ）
// "-" など空は 0（最小）扱いにして末尾になるようにしています
const toTs = (yyyyMd: string) => {
  if (!yyyyMd || yyyyMd === "-") return 0;
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

export default function InquiryManagementPage() {
  // 担当者フィルタ
  const [ownerFilter, setOwnerFilter] = React.useState<string[]>([]);

  // ソート状態（どの列を/どの方向で）
  const [sortKey, setSortKey] = React.useState<"inquiredAt" | "answeredAt" | null>(null);
  const [sortDirection, setSortDirection] = React.useState<"asc" | "desc" | null>(null);

  const rows = React.useMemo(() => {
    let data = [...INQUIRIES];

    // フィルタ
    if (ownerFilter.length > 0) {
      data = data.filter((q) => ownerFilter.includes(q.owner));
    }

    // ソート
    if (sortKey && sortDirection) {
      data.sort((a, b) => {
        const av = toTs(a[sortKey]);
        const bv = toTs(b[sortKey]);
        return sortDirection === "asc" ? av - bv : bv - av;
      });
    }

    return data;
  }, [ownerFilter, sortKey, sortDirection]);

  const headers: React.ReactNode[] = [
    "問い合わせID",
    "件名",
    "ユーザー",
    "ステータス",
    "タイプ",

    // 担当者フィルタ（ポップオーバー/チェックボックス）
    <FilterableTableHeader
      key="owner"
      label="担当者"
      options={[
        { value: "佐藤 美咲", label: "佐藤 美咲" },
        { value: "田中 雄太", label: "田中 雄太" },
      ]}
      selected={ownerFilter}
      onChange={(next: string[]) => setOwnerFilter(next)}
    />,

    // 問い合わせ日（ソート可能）
    <SortableTableHeader
      key="inquiredAt"
      label="問い合わせ日"
      sortKey="inquiredAt"
      activeKey={sortKey}
      direction={sortDirection ?? null}
      onChange={(key, dir) => {
        setSortKey(key as "inquiredAt" | "answeredAt");
        setSortDirection(dir);
      }}
    />,

    // 応答日（ソート可能）
    <SortableTableHeader
      key="answeredAt"
      label="応答日"
      sortKey="answeredAt"
      activeKey={sortKey}
      direction={sortDirection ?? null}
      onChange={(key, dir) => {
        setSortKey(key as "inquiredAt" | "answeredAt");
        setSortDirection(dir);
      }}
    />,
  ];

  return (
    <div className="p-0">
      <List
        title="問い合わせ管理"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={() => {
          setOwnerFilter([]);
          setSortKey(null);
          setSortDirection(null);
          console.log("問い合わせ一覧を更新");
        }}
      >
        {rows.map((q) => (
          <tr key={q.id}>
            <td>
              <a href="#" className="text-blue-600 hover:underline">
                {q.id}
              </a>
            </td>
            <td>
              <div className="font-medium">{q.title}</div>
              <div style={{ color: "#6b7280", fontSize: "0.85rem" }}>{q.body}</div>
            </td>
            <td>{q.user}</td>
            <td>
              {q.status === "未対応" ? (
                <span
                  style={{
                    display: "inline-flex",
                    alignItems: "center",
                    gap: 6,
                    background: "#ef4444",
                    color: "#fff",
                    fontSize: "0.75rem",
                    fontWeight: 700,
                    padding: "0.25rem 0.6rem",
                    borderRadius: 9999,
                  }}
                >
                  <span
                    style={{
                      width: 12,
                      height: 12,
                      borderRadius: 9999,
                      background: "#111827",
                      display: "inline-block",
                    }}
                  />
                  未対応
                </span>
              ) : (
                <span
                  style={{
                    display: "inline-flex",
                    alignItems: "center",
                    gap: 6,
                    background: "#0b0f1a",
                    color: "#fff",
                    fontSize: "0.75rem",
                    fontWeight: 700,
                    padding: "0.25rem 0.6rem",
                    borderRadius: 9999,
                  }}
                >
                  <span
                    style={{
                      width: 12,
                      height: 12,
                      borderRadius: 9999,
                      background: "#111827",
                      display: "inline-block",
                    }}
                  />
                  対応中
                </span>
              )}
            </td>
            <td>
              <span
                style={{
                  display: "inline-block",
                  background: "#0b0f1a",
                  color: "#fff",
                  fontSize: "0.75rem",
                  fontWeight: 700,
                  padding: "0.25rem 0.6rem",
                  borderRadius: 9999,
                }}
              >
                {q.type}
              </span>
            </td>
            <td>{q.owner}</td>
            <td>{q.inquiredAt}</td>
            <td>{q.answeredAt}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
