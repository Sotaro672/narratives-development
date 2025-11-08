// frontend/inquiry/src/pages/InquiryManagement.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";
import "./InquiryManagement.css";
import { INQUIRIES, type InquiryRow } from "../../mockdata";

// "YYYY/M/D" → number（タイムスタンプ）
// "-" など空は 0（最小）扱いにして末尾になるようにしています
const toTs = (yyyyMd: string) => {
  if (!yyyyMd || yyyyMd === "-") return 0;
  const [y, m, d] = yyyyMd.split("/").map((v) => parseInt(v, 10));
  return new Date(y, (m || 1) - 1, d || 1).getTime();
};

export default function InquiryManagementPage() {
  const navigate = useNavigate();

  // 詳細ページへ遷移
  const goDetail = (inquiryId: string) => {
    // ルートは /inquiry/:inquiryId を想定（inquiryDetail.tsx 側で useParams を使用）
    navigate(`/inquiry/${encodeURIComponent(inquiryId)}`);
  };

  // 担当者フィルタ
  const [ownerFilter, setOwnerFilter] = React.useState<string[]>([]);

  // ソート状態
  type SortKey = "inquiredAt" | "answeredAt" | null;
  const [sortKey, setSortKey] = React.useState<SortKey>(null);
  const [sortDirection, setSortDirection] =
    React.useState<"asc" | "desc" | null>(null);

  // 担当者候補はモックデータからユニーク抽出
  const ownerOptions = React.useMemo(
    () =>
      Array.from(new Set(INQUIRIES.map((q) => q.owner))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );

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

    // 担当者フィルタ
    <FilterableTableHeader
      key="owner"
      label="担当者"
      options={ownerOptions}
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
        setSortKey(key as SortKey);
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
        setSortKey(key as SortKey);
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
        {rows.map((q: InquiryRow) => (
          <tr
            key={q.id}
            onClick={() => goDetail(q.id)}
            style={{ cursor: "pointer" }}
            tabIndex={0}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") goDetail(q.id);
            }}
            aria-label={`問い合わせ ${q.id} の詳細へ`}
          >
            <td>
              <a
                href="#"
                className="inq__link"
                onClick={(e) => {
                  e.preventDefault();
                  goDetail(q.id);
                }}
              >
                {q.id}
              </a>
            </td>
            <td>
              <div className="font-medium">{q.title}</div>
              <div className="inq__excerpt">{q.body}</div>
            </td>
            <td>{q.user}</td>
            <td>
              {q.status === "未対応" ? (
                <span className="inq__badge inq__badge--danger">
                  <span className="inq__dot" />
                  未対応
                </span>
              ) : (
                <span className="inq__badge inq__badge--neutral">
                  <span className="inq__dot" />
                  対応中
                </span>
              )}
            </td>
            <td>
              <span className="inq__chip">{q.type}</span>
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
