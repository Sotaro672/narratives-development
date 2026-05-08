// frontend/transaction/src/presentation/pages/transactionList.tsx

import React, { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import { ArrowDownLeft, ArrowUpRight } from "lucide-react";
import "../styles/transaction.css";
import { TRANSACTIONS } from "../../infrastructure/mockdata/mockdata";
import type {
  Transaction,
  TransactionType,
} from "../../../../shell/src/shared/types/transaction";

// Lucide型エラー対策
const IconIn = ArrowDownLeft as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;
const IconOut = ArrowUpRight as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;

// ISO8601 → タイムスタンプ
const toTs = (s: string) => new Date(s).getTime();

// 取引先表示用: 受取 = fromAccount / 送金 = toAccount
const getCounterparty = (t: Transaction): string =>
  t.type === "receive" ? t.fromAccount : t.toAccount;

// 一覧表示用: type -> 日本語ラベル
const getTypeLabel = (type: TransactionType): string =>
  type === "receive" ? "受取" : "送金";

type SortKey = "timestamp" | "amount";
type SortDir = "asc" | "desc";

export default function TransactionListPage() {
  const navigate = useNavigate();

  // ---- フィルタ状態 ----
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [typeFilter, setTypeFilter] = useState<TransactionType[]>([]);
  const [counterpartyFilter, setCounterpartyFilter] = useState<string[]>([]);

  // ---- ソート状態 ----
  const [sortKey, setSortKey] = useState<SortKey>("timestamp");
  const [sortDir, setSortDir] = useState<SortDir>("desc");

  // ---- オプション ----
  const brandOptions = useMemo(
    () =>
      Array.from(new Set(TRANSACTIONS.map((t) => t.brandName))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );

  const typeOptions = useMemo(
    () =>
      Array.from(
        new Set<TransactionType>(TRANSACTIONS.map((t) => t.type))
      ).map((v) => ({
        value: v,
        label: getTypeLabel(v),
      })),
    []
  );

  const counterpartyOptions = useMemo(
    () =>
      Array.from(
        new Set(TRANSACTIONS.map((t) => getCounterparty(t)))
      ).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );

  // ---- フィルタ + ソート ----
  const rows = useMemo(() => {
    let work = TRANSACTIONS.filter((t) => {
      const cp = getCounterparty(t);
      return (
        (brandFilter.length === 0 || brandFilter.includes(t.brandName)) &&
        (typeFilter.length === 0 || typeFilter.includes(t.type)) &&
        (counterpartyFilter.length === 0 || counterpartyFilter.includes(cp))
      );
    });

    work = [...work].sort((a, b) => {
      if (sortKey === "timestamp") {
        const av = toTs(a.timestamp);
        const bv = toTs(b.timestamp);
        return sortDir === "asc" ? av - bv : bv - av;
      }
      // amount: 値は常に正。direction は type を問わず数値でソート。
      const av = a.amount;
      const bv = b.amount;
      return sortDir === "asc" ? av - bv : bv - av;
    });

    return work;
  }, [brandFilter, typeFilter, counterpartyFilter, sortKey, sortDir]);

  // ---- 行クリック時の遷移処理 ----
  const goDetail = (transactionId: string) => {
    navigate(`/transaction/${encodeURIComponent(transactionId)}`);
  };

  const headers: React.ReactNode[] = [
    <SortableTableHeader
      key="h-timestamp"
      label="日時"
      sortKey="timestamp"
      activeKey={sortKey}
      direction={sortKey === "timestamp" ? sortDir : undefined}
      onChange={(_, dir) => {
        setSortKey("timestamp");
        setSortDir(dir);
      }}
    />,
    <FilterableTableHeader
      key="h-brand"
      label="ブランド"
      options={brandOptions}
      selected={brandFilter}
      onChange={setBrandFilter}
    />,
    <FilterableTableHeader
      key="h-type"
      label="種別"
      options={typeOptions}
      selected={typeFilter}
      onChange={(vals) => setTypeFilter(vals as TransactionType[])}
    />,
    "説明",
    <SortableTableHeader
      key="h-amount"
      label="金額"
      sortKey="amount"
      activeKey={sortKey}
      direction={sortKey === "amount" ? sortDir : undefined}
      onChange={(_, dir) => {
        setSortKey("amount");
        setSortDir(dir);
      }}
    />,
    <FilterableTableHeader
      key="h-counterparty"
      label="取引先"
      options={counterpartyOptions}
      selected={counterpartyFilter}
      onChange={setCounterpartyFilter}
    />,
  ];

  const typeClass = (type: TransactionType) =>
    `transaction-type-badge ${
      type === "receive" ? "is-receive" : "is-send"
    }`;

  const renderTypeBadge = (type: TransactionType) => {
    const Icon = type === "receive" ? IconIn : IconOut;
    return (
      <span className={typeClass(type)}>
        <Icon width={16} height={16} />
        {getTypeLabel(type)}
      </span>
    );
  };

  const amountClass = (isPlus: boolean) =>
    `transaction-amount ${isPlus ? "is-plus" : "is-minus"}`;

  const renderAmount = (t: Transaction) => {
    const isPlus = t.type === "receive";
    const sign = isPlus ? "+" : "-";
    const n = t.amount.toLocaleString();
    return (
      <span className={amountClass(isPlus)}>
        {sign}
        {n} 円
      </span>
    );
  };

  const formatDateTime = (iso: string) => {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return { date: iso, time: "" };
    const yyyy = d.getFullYear();
    const mm = String(d.getMonth() + 1).padStart(2, "0");
    const dd = String(d.getDate()).padStart(2, "0");
    const hh = String(d.getHours()).padStart(2, "0");
    const mi = String(d.getMinutes()).padStart(2, "0");
    const ss = String(d.getSeconds()).padStart(2, "0");
    return { date: `${yyyy}/${mm}/${dd}`, time: `${hh}:${mi}:${ss}` };
  };

  return (
    <div className="p-0">
      <List
        title="取引履歴"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={() => {
          setBrandFilter([]);
          setTypeFilter([]);
          setCounterpartyFilter([]);
          setSortKey("timestamp");
          setSortDir("desc");
          console.log("取引履歴リセット");
        }}
      >
        {rows.map((t) => {
          const { date, time } = formatDateTime(t.timestamp);
          const counterparty = getCounterparty(t);
          return (
            <tr
              key={t.id}
              role="button"
              tabIndex={0}
              className="cursor-pointer hover:bg-slate-50 transition-colors"
              onClick={() => goDetail(t.id)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  goDetail(t.id);
                }
              }}
            >
              <td>
                <div>{date}</div>
                <div className="transaction-time-sub">{time}</div>
              </td>
              <td>
                <span className="lp-brand-pill">{t.brandName}</span>
              </td>
              <td>{renderTypeBadge(t.type)}</td>
              <td>{t.description}</td>
              <td>{renderAmount(t)}</td>
              <td>{counterparty}</td>
            </tr>
          );
        })}
      </List>
    </div>
  );
}
