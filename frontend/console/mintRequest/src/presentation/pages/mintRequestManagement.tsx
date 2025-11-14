// frontend/mintRequest/src/presentation/pages/mintRequestManagement.tsx

import React, { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../shell/src/layout/List/List";
import "../styles/mintRequest.css";
import {
  MINT_REQUESTS,
} from "../../infrastructure/mockdata/mockdata";
import type {
  MintRequest,
  MintRequestStatus,
} from "../../../../shell/src/shared/types/mintRequest";

// 日時文字列をタイムスタンプに変換（不正 or null は -1）
const toTs = (s: string | null | undefined): number => {
  if (!s) return -1;
  const t = Date.parse(s);
  return Number.isNaN(t) ? -1 : t;
};

// ステータス表示用ラベル
const statusLabel = (s: MintRequestStatus): string => {
  switch (s) {
    case "minted":
      return "Mint完了";
    case "requested":
      return "リクエスト済み";
    case "planning":
    default:
      return "計画中";
  }
};

type SortKey = "requestedAt" | "mintedAt" | "mintQuantity" | null;

export default function MintRequestManagementPage() {
  const navigate = useNavigate();

  // Filters
  const [tokenFilter, setTokenFilter] = useState<string[]>([]);
  const [productionFilter, setProductionFilter] = useState<string[]>([]);
  const [requesterFilter, setRequesterFilter] = useState<string[]>([]);
  const [statusFilter, setStatusFilter] = useState<MintRequestStatus[] | string[]>([]);

  // Sorting
  const [sortKey, setSortKey] = useState<SortKey>("requestedAt");
  const [sortDir, setSortDir] = useState<"asc" | "desc" | null>("desc");

  // Filter options
  const tokenOptions = useMemo(
    () =>
      Array.from(new Set(MINT_REQUESTS.map((r) => r.tokenBlueprintId))).map(
        (v) => ({ value: v, label: v }),
      ),
    [],
  );

  const productionOptions = useMemo(
    () =>
      Array.from(new Set(MINT_REQUESTS.map((r) => r.productionId))).map(
        (v) => ({ value: v, label: v }),
      ),
    [],
  );

  const requesterOptions = useMemo(
    () =>
      Array.from(
        new Set(
          MINT_REQUESTS.map((r) => r.requestedBy).filter(
            (v): v is string => !!v && !!v.trim(),
          ),
        ),
      ).map((v) => ({ value: v, label: v })),
    [],
  );

  const statusOptions = useMemo(
    () =>
      Array.from(new Set(MINT_REQUESTS.map((r) => r.status))).map((v) => ({
        value: v,
        label: statusLabel(v),
      })),
    [],
  );

  // Filter + sort rows
  const rows = useMemo(() => {
    let data = MINT_REQUESTS.filter((r) => {
      const tokenOk =
        tokenFilter.length === 0 ||
        tokenFilter.includes(r.tokenBlueprintId);
      const productionOk =
        productionFilter.length === 0 ||
        productionFilter.includes(r.productionId);
      const requesterOk =
        requesterFilter.length === 0 ||
        requesterFilter.includes(r.requestedBy ?? "");
      const statusOk =
        statusFilter.length === 0 ||
        statusFilter.includes(r.status);

      return tokenOk && productionOk && requesterOk && statusOk;
    });

    if (sortKey && sortDir) {
      data = [...data].sort((a, b) => {
        if (sortKey === "mintQuantity") {
          return sortDir === "asc"
            ? a.mintQuantity - b.mintQuantity
            : b.mintQuantity - a.mintQuantity;
        }

        const av = toTs(a[sortKey]);
        const bv = toTs(b[sortKey]);
        return sortDir === "asc" ? av - bv : bv - av;
      });
    }

    return data;
  }, [
    tokenFilter,
    productionFilter,
    requesterFilter,
    statusFilter,
    sortKey,
    sortDir,
  ]);

  // 行クリックで詳細へ遷移（id を利用）
  const goDetail = (requestId: string) => {
    navigate(`/mintRequest/${encodeURIComponent(requestId)}`);
  };

  const headers: React.ReactNode[] = [
    "ミント申請ID",
    <FilterableTableHeader
      key="tokenBlueprintId"
      label="トークン設計ID"
      options={tokenOptions}
      selected={tokenFilter}
      onChange={setTokenFilter}
    />,
    <FilterableTableHeader
      key="productionId"
      label="生産ID"
      options={productionOptions}
      selected={productionFilter}
      onChange={setProductionFilter}
    />,
    <SortableTableHeader
      key="mintQuantity"
      label="Mint数量"
      sortKey="mintQuantity"
      activeKey={sortKey}
      direction={sortDir ?? null}
      onChange={(key, dir) => {
        setSortKey(key as SortKey);
        setSortDir(dir);
      }}
    />,
    <FilterableTableHeader
      key="status"
      label="ステータス"
      options={statusOptions}
      selected={statusFilter}
      onChange={(next: string[]) =>
        setStatusFilter(next as MintRequestStatus[] | string[])
      }
    />,
    <FilterableTableHeader
      key="requester"
      label="リクエスト者"
      options={requesterOptions}
      selected={requesterFilter}
      onChange={setRequesterFilter}
    />,
    <SortableTableHeader
      key="requestedAt"
      label="リクエスト日時"
      sortKey="requestedAt"
      activeKey={sortKey}
      direction={sortDir ?? null}
      onChange={(key, dir) => {
        setSortKey(key as SortKey);
        setSortDir(dir);
      }}
    />,
    <SortableTableHeader
      key="mintedAt"
      label="Mint実行日時"
      sortKey="mintedAt"
      activeKey={sortKey}
      direction={sortDir ?? null}
      onChange={(key, dir) => {
        setSortKey(key as SortKey);
        setSortDir(dir);
      }}
    />,
  ];

  return (
    <div className="p-0">
      <List
        title="ミントリクエスト管理"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={() => {
          setTokenFilter([]);
          setProductionFilter([]);
          setRequesterFilter([]);
          setStatusFilter([]);
          setSortKey("requestedAt");
          setSortDir("desc");
        }}
      >
        {rows.map((r: MintRequest) => (
          <tr
            key={r.id}
            onClick={() => goDetail(r.id)}
            style={{ cursor: "pointer" }}
            tabIndex={0}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") goDetail(r.id);
            }}
            aria-label={`ミント申請 ${r.id} の詳細へ`}
          >
            <td>
              <a
                href="#"
                onClick={(e) => {
                  e.preventDefault();
                  goDetail(r.id);
                }}
                className="text-blue-600 hover:underline"
              >
                {r.id}
              </a>
            </td>
            <td>
              <span className="lp-brand-pill">{r.tokenBlueprintId}</span>
            </td>
            <td>
              <span className="lp-brand-pill">{r.productionId}</span>
            </td>
            <td>{r.mintQuantity}</td>
            <td>
              {r.status === "minted" ? (
                <span className="mint-badge is-done">
                  {statusLabel(r.status)}
                </span>
              ) : r.status === "requested" ? (
                <span className="mint-badge is-requested">
                  {statusLabel(r.status)}
                </span>
              ) : (
                <span className="mint-badge is-planned">
                  {statusLabel(r.status)}
                </span>
              )}
            </td>
            <td>{r.requestedBy ?? "-"}</td>
            <td>{r.requestedAt ?? "-"}</td>
            <td>{r.mintedAt ?? "-"}</td>
          </tr>
        ))}
      </List>
    </div>
  );
}
