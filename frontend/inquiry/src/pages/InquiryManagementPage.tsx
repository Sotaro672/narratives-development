// frontend/inquiry/src/pages/InquiryManagementPage.tsx
import * as React from "react";
import List from "../../../shell/src/layout/List/List";
import { Filter, ChevronDown, ChevronUp } from "lucide-react";

// Lucide を React の汎用 SVG コンポーネントにキャスト（型不整合の暫定回避）
const IconFilter = Filter as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;
const IconChevronDown = ChevronDown as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;
const IconChevronUp = ChevronUp as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;

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

export default function InquiryManagementPage() {
  const [sortAsc, setSortAsc] = React.useState(false);

  const rows = React.useMemo(() => {
    const data = [...INQUIRIES];
    data.sort((a, b) =>
      sortAsc
        ? a.inquiredAt.localeCompare(b.inquiredAt)
        : b.inquiredAt.localeCompare(a.inquiredAt)
    );
    return data;
  }, [sortAsc]);

  const headers: React.ReactNode[] = [
    "問い合わせID",
    "件名",
    "ユーザー",
    <>ステータス</>,
    <>タイプ</>,
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
        <span>問い合わせ日</span>
        <button
          className="lp-th-filter"
          aria-label={`問い合わせ日でソート（${sortAsc ? "昇順" : "降順"}）`}
          onClick={() => setSortAsc((v) => !v)}
        >
          {sortAsc ? (
            <IconChevronUp width={16} height={16} />
          ) : (
            <IconChevronDown width={16} height={16} />
          )}
        </button>
      </span>
    </>,
    "応答日",
  ];

  return (
    <div className="p-0">
      <List
        title="問い合わせ管理"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={() => console.log("問い合わせ一覧を更新")}
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
