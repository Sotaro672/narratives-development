// frontend/mintRequest/src/pages/mintRequestManagement.tsx
import React, { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";
import "./mintRequestManagement.css";

type MintStatus = "リクエスト済み" | "Mint完了" | "計画中";

type Row = {
  planId: string;
  tokenDesign: string;
  productName: string;
  quantity: number;
  status: MintStatus;
  requester: string;
  requestAt: string;   // "YYYY/MM/DD HH:mm:ss"
  executedAt: string;  // same or "-"
};

const ROWS: Row[] = [
  {
    planId: "production_002",
    tokenDesign: "NEXUS Street Token",
    productName: "production_002",
    quantity: 5,
    status: "リクエスト済み",
    requester: "佐藤 美咲",
    requestAt: "2025/11/3 11:05:08",
    executedAt: "-",
  },
  {
    planId: "production_003",
    tokenDesign: "LUMINA VIP Token",
    productName: "production_003",
    quantity: 10,
    status: "Mint完了",
    requester: "高橋 健太",
    requestAt: "2025/10/26 11:05:08",
    executedAt: "2025/10/28 11:05:08",
  },
  {
    planId: "production_004",
    tokenDesign: "NEXUS Community Token",
    productName: "production_004",
    quantity: 12,
    status: "Mint完了",
    requester: "山田 太郎",
    requestAt: "2025/10/21 11:05:08",
    executedAt: "2025/10/24 11:05:08",
  },
  {
    planId: "production_001",
    tokenDesign: "SILK Premium Token",
    productName: "production_001",
    quantity: 10,
    status: "計画中",
    requester: "山田 太郎",
    requestAt: "1970/1/1 9:00:00",
    executedAt: "-",
  },
];

const toTs = (s: string) => (s === "-" ? -1 : new Date(s.replace(/-/g, "/")).getTime());

export default function MintRequestManagementPage() {
  const navigate = useNavigate();

  // ── Filters ───────────────────────────────────────────────
  const [tokenFilter, setTokenFilter] = useState<string[]>([]);
  const [productFilter, setProductFilter] = useState<string[]>([]);
  const [requesterFilter, setRequesterFilter] = useState<string[]>([]);
  const [statusFilter, setStatusFilter] = useState<string[]>([]);

  // ── Sorting ───────────────────────────────────────────────
  const [sortKey, setSortKey] =
    useState<"requestAt" | "executedAt" | "quantity" | null>("requestAt");
  const [sortDir, setSortDir] = useState<"asc" | "desc" | null>("desc");

  // options for filters
  const tokenOptions = useMemo(
    () => Array.from(new Set(ROWS.map((r) => r.tokenDesign))).map((v) => ({ value: v, label: v })),
    []
  );
  const productOptions = useMemo(
    () => Array.from(new Set(ROWS.map((r) => r.productName))).map((v) => ({ value: v, label: v })),
    []
  );
  const requesterOptions = useMemo(
    () => Array.from(new Set(ROWS.map((r) => r.requester))).map((v) => ({ value: v, label: v })),
    []
  );
  const statusOptions = useMemo(
    () => Array.from(new Set(ROWS.map((r) => r.status))).map((v) => ({ value: v, label: v })),
    []
  );

  // filter + sort
  const rows = useMemo(() => {
    let data = ROWS.filter(
      (r) =>
        (tokenFilter.length === 0 || tokenFilter.includes(r.tokenDesign)) &&
        (productFilter.length === 0 || productFilter.includes(r.productName)) &&
        (requesterFilter.length === 0 || requesterFilter.includes(r.requester)) &&
        (statusFilter.length === 0 || statusFilter.includes(r.status))
    );

    if (sortKey && sortDir) {
      data = [...data].sort((a, b) => {
        if (sortKey === "quantity") {
          return sortDir === "asc" ? a.quantity - b.quantity : b.quantity - a.quantity;
        }
        const av = toTs(a[sortKey]);
        const bv = toTs(b[sortKey]);
        return sortDir === "asc" ? av - bv : bv - av;
      });
    }

    return data;
  }, [tokenFilter, productFilter, requesterFilter, statusFilter, sortKey, sortDir]);

  const headers: React.ReactNode[] = [
    "生産計画ID",
    <FilterableTableHeader
      key="token"
      label="トークン設計"
      options={tokenOptions}
      selected={tokenFilter}
      onChange={setTokenFilter}
    />,
    <FilterableTableHeader
      key="product"
      label="商品名"
      options={productOptions}
      selected={productFilter}
      onChange={setProductFilter}
    />,
    <SortableTableHeader
      key="quantity"
      label="Mint数量"
      sortKey="quantity"
      activeKey={sortKey}
      direction={sortDir ?? null}
      onChange={(key, dir) => {
        setSortKey(key as "quantity");
        setSortDir(dir);
      }}
    />,
    <FilterableTableHeader
      key="status"
      label="ステータス"
      options={statusOptions}
      selected={statusFilter}
      onChange={setStatusFilter}
    />,
    <FilterableTableHeader
      key="requester"
      label="リクエスト者"
      options={requesterOptions}
      selected={requesterFilter}
      onChange={setRequesterFilter}
    />,
    <SortableTableHeader
      key="requestAt"
      label="リクエスト日時"
      sortKey="requestAt"
      activeKey={sortKey}
      direction={sortDir ?? null}
      onChange={(key, dir) => {
        setSortKey(key as "requestAt");
        setSortDir(dir);
      }}
    />,
    <SortableTableHeader
      key="executedAt"
      label="Mint実行日時"
      sortKey="executedAt"
      activeKey={sortKey}
      direction={sortDir ?? null}
      onChange={(key, dir) => {
        setSortKey(key as "executedAt");
        setSortDir(dir);
      }}
    />,
  ];

  // 行クリックで詳細へ遷移
  const goDetail = (requestId: string) => {
    // ルートは /mintRequest/:requestId を想定（mintRequestDetail.tsx で useParams を使用）
    navigate(`/mintRequest/${encodeURIComponent(requestId)}`);
  };

  return (
    <div className="p-0">
      <List
        title="ミントリクエスト管理"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={() => {
          setTokenFilter([]);
          setProductFilter([]);
          setRequesterFilter([]);
          setStatusFilter([]);
          setSortKey("requestAt");
          setSortDir("desc");
        }}
      >
        {rows.map((r) => (
          <tr
            key={r.planId}
            onClick={() => goDetail(r.planId)}
            style={{ cursor: "pointer" }}
            tabIndex={0}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") goDetail(r.planId);
            }}
            aria-label={`ミント申請 ${r.planId} の詳細へ`}
          >
            <td>
              <a
                href="#"
                onClick={(e) => {
                  e.preventDefault();
                  goDetail(r.planId);
                }}
                className="text-blue-600 hover:underline"
              >
                {r.planId}
              </a>
            </td>
            <td>
              <span className="lp-brand-pill">{r.tokenDesign}</span>
            </td>
            <td>
              <span className="lp-brand-pill">{r.productName}</span>
            </td>
            <td>{r.quantity}</td>
            <td>
              {r.status === "Mint完了" ? (
                <span className="mint-badge is-done">Mint完了</span>
              ) : r.status === "リクエスト済み" ? (
                <span className="mint-badge is-requested">リクエスト済み</span>
              ) : (
                <span className="mint-badge is-planned">計画中</span>
              )}
            </td>
            <td>{r.requester}</td>
            <td>{r.requestAt}</td>
            <td>{r.executedAt}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
