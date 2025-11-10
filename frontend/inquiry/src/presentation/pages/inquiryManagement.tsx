// frontend/inquiry/src/presentation/pages/InquiryManagement.tsx

import * as React from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import "../styles/inquiry.css";
import { INQUIRIES } from "../../infrastructure/mockdata/mockdata";
import type {
  Inquiry,
  InquiryStatus,
  InquiryType,
} from "../../../../shell/src/shared/types/inquiry";

// ISO8601 → number（タイムスタンプ）
// 不正 or 未設定は 0 扱い（= 並び替え時に末尾側へ）
const toTs = (iso?: string | null): number => {
  if (!iso) return 0;
  const t = Date.parse(iso);
  return Number.isNaN(t) ? 0 : t;
};

const formatDate = (iso?: string | null): string => {
  if (!iso) return "-";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "-";
  const y = d.getFullYear();
  const m = d.getMonth() + 1;
  const day = d.getDate();
  return `${y}/${m}/${day}`;
};

const statusLabel = (status: InquiryStatus): string => {
  switch (status) {
    case "pending":
      return "未対応";
    case "in_progress":
      return "対応中";
    case "resolved":
      return "対応完了";
    case "closed":
      return "クローズ";
    default:
      return String(status);
  }
};

const typeLabel = (t: InquiryType): string => {
  switch (t) {
    case "product_description":
      return "商品説明";
    case "exchange":
      return "交換";
    default:
      return "その他";
  }
};

export default function InquiryManagementPage() {
  const navigate = useNavigate();

  // 詳細ページへ遷移
  const goDetail = (inquiryId: string) => {
    // ルートは /inquiry/:inquiryId を想定
    navigate(`/inquiry/${encodeURIComponent(inquiryId)}`);
  };

  // 担当者（assigneeId）フィルタ
  const [assigneeFilter, setAssigneeFilter] = React.useState<string[]>([]);

  // ソート状態
  type SortKey = "createdAt" | "updatedAt" | null;
  const [sortKey, setSortKey] = React.useState<SortKey>(null);
  const [sortDirection, setSortDirection] =
    React.useState<"asc" | "desc" | null>(null);

  // 担当者候補は assigneeId からユニーク抽出（null/空は除外）
  const assigneeOptions = React.useMemo(
    () =>
      Array.from(
        new Set(
          INQUIRIES.map((q) => q.assigneeId?.trim())
            .filter((v): v is string => !!v),
        ),
      ).map((v) => ({
        value: v,
        label: v,
      })),
    [],
  );

  const rows = React.useMemo<Inquiry[]>(() => {
    let data = [...INQUIRIES];

    // フィルタ（assigneeId）
    if (assigneeFilter.length > 0) {
      data = data.filter(
        (q) =>
          q.assigneeId &&
          assigneeFilter.includes(q.assigneeId),
      );
    }

    // ソート（createdAt / updatedAt）
    if (sortKey && sortDirection) {
      data.sort((a, b) => {
        const av = toTs(a[sortKey]);
        const bv = toTs(b[sortKey]);
        return sortDirection === "asc" ? av - bv : bv - av;
      });
    }

    return data;
  }, [assigneeFilter, sortKey, sortDirection]);

  const headers: React.ReactNode[] = [
    "問い合わせID",
    "件名",
    "ユーザー(ID)", // avatarId をそのまま表示
    "ステータス",
    "タイプ",

    // 担当者フィルタ（assigneeId）
    <FilterableTableHeader
      key="assignee"
      label="担当者 (memberId)"
      options={assigneeOptions}
      selected={assigneeFilter}
      onChange={(next: string[]) => setAssigneeFilter(next)}
    />,

    // 問い合わせ日（createdAt, ソート可能）
    <SortableTableHeader
      key="createdAt"
      label="問い合わせ日"
      sortKey="createdAt"
      activeKey={sortKey}
      direction={sortDirection ?? null}
      onChange={(key, dir) => {
        setSortKey(key as SortKey);
        setSortDirection(dir);
      }}
    />,

    // 最終更新日（updatedAt, ソート可能 / 応答日時相当）
    <SortableTableHeader
      key="updatedAt"
      label="最終更新日"
      sortKey="updatedAt"
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
          setAssigneeFilter([]);
          setSortKey(null);
          setSortDirection(null);
          console.log("問い合わせ一覧を更新");
        }}
      >
        {rows.map((q) => (
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

            {/* 件名 + 抜粋 */}
            <td>
              <div className="font-medium">{q.subject}</div>
              <div className="inq__excerpt">{q.content}</div>
            </td>

            {/* ユーザー: avatarId を暫定表示 */}
            <td>{q.avatarId}</td>

            {/* ステータス */}
            <td>
              {q.status === "pending" ? (
                <span className="inq__badge inq__badge--danger">
                  <span className="inq__dot" />
                  {statusLabel(q.status)}
                </span>
              ) : q.status === "in_progress" ? (
                <span className="inq__badge inq__badge--neutral">
                  <span className="inq__dot" />
                  {statusLabel(q.status)}
                </span>
              ) : q.status === "resolved" ? (
                <span className="inq__badge inq__badge--success">
                  <span className="inq__dot" />
                  {statusLabel(q.status)}
                </span>
              ) : (
                <span className="inq__badge inq__badge--muted">
                  <span className="inq__dot" />
                  {statusLabel(q.status)}
                </span>
              )}
            </td>

            {/* 問い合わせタイプ */}
            <td>
              <span className="inq__chip">
                {typeLabel(q.inquiryType)}
              </span>
            </td>

            {/* 担当者 (memberId) */}
            <td>{q.assigneeId ?? "-"}</td>

            {/* createdAt / updatedAt を日付表示 */}
            <td>{formatDate(q.createdAt)}</td>
            <td>{formatDate(q.updatedAt)}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
