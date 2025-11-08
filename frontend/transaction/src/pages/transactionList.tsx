import React, { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom"; // ← 追加
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";
import { ArrowDownLeft, ArrowUpRight } from "lucide-react";
import "./transactionList.css";
import { TRANSACTIONS, type Transaction } from "../../mockdata";

// Lucide型エラー対策
const IconIn = ArrowDownLeft as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;
const IconOut = ArrowUpRight as unknown as React.ComponentType<
  React.SVGProps<SVGSVGElement>
>;

// 日時文字列 → タイムスタンプ変換
const toTs = (s: string) => new Date(s.replace(/-/g, "/")).getTime();

export default function TransactionListPage() {
  const navigate = useNavigate(); // ← 追加

  // ---- フィルタ状態 ----
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [typeFilter, setTypeFilter] = useState<string[]>([]);
  const [counterpartyFilter, setCounterpartyFilter] = useState<string[]>([]);

  // ---- ソート状態 ----
  const [sortKey, setSortKey] = useState<"datetime" | "amount">("datetime");
  const [sortDir, setSortDir] = useState<"asc" | "desc">("desc");

  // ---- オプション ----
  const brandOptions = useMemo(
    () =>
      Array.from(new Set(TRANSACTIONS.map((t) => t.brand))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );

  const typeOptions = useMemo(
    () =>
      Array.from(new Set(TRANSACTIONS.map((t) => t.type))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );

  const counterpartyOptions = useMemo(
    () =>
      Array.from(new Set(TRANSACTIONS.map((t) => t.counterparty))).map((v) => ({
        value: v,
        label: v,
      })),
    []
  );

  // ---- フィルタ + ソート ----
  const rows = useMemo(() => {
    let work = TRANSACTIONS.filter(
      (t) =>
        (brandFilter.length === 0 || brandFilter.includes(t.brand)) &&
        (typeFilter.length === 0 || typeFilter.includes(t.type)) &&
        (counterpartyFilter.length === 0 ||
          counterpartyFilter.includes(t.counterparty))
    );

    work = [...work].sort((a, b) => {
      if (sortKey === "datetime") {
        const av = toTs(a.datetime);
        const bv = toTs(b.datetime);
        return sortDir === "asc" ? av - bv : bv - av;
      }
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
      key="h-datetime"
      label="日時"
      sortKey="datetime"
      activeKey={sortKey}
      direction={sortKey === "datetime" ? sortDir : undefined}
      onChange={(_, dir) => {
        setSortKey("datetime");
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
      onChange={setTypeFilter}
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

  const typeClass = (type: Transaction["type"]) =>
    `transaction-type-badge ${type === "受取" ? "is-receive" : "is-send"}`;

  const renderTypeBadge = (type: Transaction["type"]) => {
    const Icon = type === "受取" ? IconIn : IconOut;
    return (
      <span className={typeClass(type)}>
        <Icon width={16} height={16} />
        {type}
      </span>
    );
  };

  const amountClass = (amt: number) =>
    `transaction-amount ${amt >= 0 ? "is-plus" : "is-minus"}`;

  const renderAmount = (amt: number) => {
    const isPlus = amt >= 0;
    const n = Math.abs(amt).toLocaleString();
    return (
      <span className={amountClass(amt)}>
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
        onReset={() => {
          setBrandFilter([]);
          setTypeFilter([]);
          setCounterpartyFilter([]);
          setSortKey("datetime");
          setSortDir("desc");
          console.log("取引履歴リセット");
        }}
      >
        {rows.map((t, idx) => {
          const [date, time] = t.datetime.split(" ");
          return (
            <tr
              key={`${t.datetime}-${idx}`}
              role="button"
              tabIndex={0}
              className="cursor-pointer hover:bg-slate-50 transition-colors"
              onClick={() => goDetail(t.datetime)} // ← 行クリックで遷移
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  goDetail(t.datetime);
                }
              }}
            >
              <td>
                <div>{date}</div>
                <div className="transaction-time-sub">{time}</div>
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
