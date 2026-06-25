// frontend/inquiry/src/presentation/pages/InquiryManagement.tsx
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import List, {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../shell/src/layout/List/List";
import { safeDateTimeLabelJa } from "../../../shell/src/shared/util/dateJa";

import {
  listInquiriesHTTP,
  type InquiryManagementItem,
} from "../../infrastructure/inquiryRepositoryHTTP";

const CURRENT_COMPANY_ID_ROUTE_PLACEHOLDER = "current";
const INQUIRY_DETAIL_ROUTE_BASE = "/inquiry";

type SortKey = "createdAt" | "updatedAt" | null;
type SortDir = "asc" | "desc" | null;

function textOrDash(value: string | null | undefined): string {
  const trimmed = String(value ?? "").trim();
  return trimmed || "-";
}

function normalizeText(value: string | null | undefined): string {
  return String(value ?? "").trim();
}

function getInquiryID(item: InquiryManagementItem): string {
  return textOrDash(item.inquiry.id);
}

function getSubject(item: InquiryManagementItem): string {
  return textOrDash(item.inquiry.subject);
}

function getCustomerName(item: InquiryManagementItem): string {
  return textOrDash(item.userFullName);
}

function getStatus(item: InquiryManagementItem): string {
  return textOrDash(item.inquiry.status);
}

function getProductName(item: InquiryManagementItem): string {
  return textOrDash(item.productName);
}

function getBrandName(item: InquiryManagementItem): string {
  return textOrDash(item.brandName);
}

function getCreatedAt(item: InquiryManagementItem): string {
  return safeDateTimeLabelJa(item.inquiry.createdAt, "-");
}

function getUpdatedAt(item: InquiryManagementItem): string {
  return safeDateTimeLabelJa(item.inquiry.updatedAt, "-");
}

function uniqueOptions(values: string[]): Array<{ value: string; label: string }> {
  const seen = new Set<string>();
  const options: Array<{ value: string; label: string }> = [];

  for (const value of values) {
    const normalized = normalizeText(value);
    if (!normalized || normalized === "-") continue;
    if (seen.has(normalized)) continue;

    seen.add(normalized);
    options.push({
      value: normalized,
      label: normalized,
    });
  }

  return options;
}

function toTimestamp(value: string | null | undefined): number | null {
  const normalized = normalizeText(value);
  if (!normalized) return null;

  const parsed = Date.parse(normalized);
  if (!Number.isNaN(parsed)) return parsed;

  const m =
    normalized.match(
      /^(\d{4})\/(\d{1,2})\/(\d{1,2})(?:\s+(\d{1,2}):(\d{1,2})(?::(\d{1,2}))?)?$/,
    ) ?? null;

  if (!m) return null;

  const year = Number(m[1]);
  const month = Number(m[2]);
  const day = Number(m[3]);
  const hour = Number(m[4] ?? "0");
  const minute = Number(m[5] ?? "0");
  const second = Number(m[6] ?? "0");

  const date = new Date(year, month - 1, day, hour, minute, second);
  const ts = date.getTime();

  return Number.isNaN(ts) ? null : ts;
}

function compareDateValues(
  a: string | null | undefined,
  b: string | null | undefined,
  direction: SortDir,
): number {
  const av = toTimestamp(a);
  const bv = toTimestamp(b);

  if (av === null && bv === null) return 0;
  if (av === null) return 1;
  if (bv === null) return -1;

  return direction === "asc" ? av - bv : bv - av;
}

export default function InquiryManagementPage() {
  const navigate = useNavigate();

  const [items, setItems] = useState<InquiryManagementItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [isResetting, setIsResetting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const [statusFilter, setStatusFilter] = useState<string[]>([]);
  const [productNameFilter, setProductNameFilter] = useState<string[]>([]);
  const [brandNameFilter, setBrandNameFilter] = useState<string[]>([]);

  const [sortKey, setSortKey] = useState<SortKey>("createdAt");
  const [sortDir, setSortDir] = useState<SortDir>("desc");

  const fetchRows = useCallback(async () => {
    setErrorMessage(null);

    try {
      const result = await listInquiriesHTTP({
        // backend 側では middleware の companyId を正として使う。
        // route 互換のため URL には non-empty placeholder を渡す。
        companyId: CURRENT_COMPANY_ID_ROUTE_PLACEHOLDER,
      });

      setItems(Array.isArray(result.items) ? result.items : []);
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : "問い合わせ一覧の取得に失敗しました";

      setErrorMessage(message);
      setItems([]);
    }
  }, []);

  useEffect(() => {
    let active = true;

    async function load() {
      setLoading(true);
      setErrorMessage(null);

      try {
        const result = await listInquiriesHTTP({
          // backend 側では middleware の companyId を正として使う。
          // route 互換のため URL には non-empty placeholder を渡す。
          companyId: CURRENT_COMPANY_ID_ROUTE_PLACEHOLDER,
        });

        if (!active) return;

        setItems(Array.isArray(result.items) ? result.items : []);
      } catch (error) {
        if (!active) return;

        const message =
          error instanceof Error
            ? error.message
            : "問い合わせ一覧の取得に失敗しました";

        setErrorMessage(message);
        setItems([]);
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }

    void load();

    return () => {
      active = false;
    };
  }, []);

  const statusOptions = useMemo(() => {
    return uniqueOptions(items.map((item) => getStatus(item)));
  }, [items]);

  const productNameOptions = useMemo(() => {
    return uniqueOptions(items.map((item) => getProductName(item)));
  }, [items]);

  const brandNameOptions = useMemo(() => {
    return uniqueOptions(items.map((item) => getBrandName(item)));
  }, [items]);

  const filteredItems = useMemo(() => {
    let next = items.filter((item) => {
      const status = getStatus(item);
      const productName = getProductName(item);
      const brandName = getBrandName(item);

      const statusOk =
        statusFilter.length === 0 || statusFilter.includes(status);
      const productNameOk =
        productNameFilter.length === 0 ||
        productNameFilter.includes(productName);
      const brandNameOk =
        brandNameFilter.length === 0 || brandNameFilter.includes(brandName);

      return statusOk && productNameOk && brandNameOk;
    });

    if (sortKey && sortDir) {
      next = [...next].sort((a, b) => {
        if (sortKey === "createdAt") {
          return compareDateValues(
            a.inquiry.createdAt,
            b.inquiry.createdAt,
            sortDir,
          );
        }

        if (sortKey === "updatedAt") {
          return compareDateValues(
            a.inquiry.updatedAt,
            b.inquiry.updatedAt,
            sortDir,
          );
        }

        return 0;
      });
    }

    return next;
  }, [
    items,
    statusFilter,
    productNameFilter,
    brandNameFilter,
    sortKey,
    sortDir,
  ]);

  const handleClickRow = useCallback(
    (inquiryID: string) => {
      const trimmed = normalizeText(inquiryID);
      if (!trimmed || trimmed === "-") return;

      navigate(`${INQUIRY_DETAIL_ROUTE_BASE}/${encodeURIComponent(trimmed)}`);
    },
    [navigate],
  );

  const rows = useMemo(() => {
    return filteredItems.map((item) => {
      const inquiryID = getInquiryID(item);

      return (
        <tr
          key={inquiryID}
          role="button"
          tabIndex={0}
          style={{ cursor: "pointer" }}
          onClick={() => handleClickRow(inquiryID)}
          onKeyDown={(event) => {
            if (event.key === "Enter" || event.key === " ") {
              event.preventDefault();
              handleClickRow(inquiryID);
            }
          }}
        >
          <td>{getSubject(item)}</td>
          <td>{getCustomerName(item)}</td>
          <td>{getStatus(item)}</td>
          <td>{getProductName(item)}</td>
          <td>{getBrandName(item)}</td>
          <td>{getCreatedAt(item)}</td>
          <td>{getUpdatedAt(item)}</td>
        </tr>
      );
    });
  }, [filteredItems, handleClickRow]);

  const headers = useMemo(() => {
    return [
      "件名",
      "お客様名",
      <FilterableTableHeader
        key="status"
        label="ステータス"
        options={statusOptions}
        selected={statusFilter}
        onChange={setStatusFilter}
      />,
      <FilterableTableHeader
        key="productName"
        label="商品名"
        options={productNameOptions}
        selected={productNameFilter}
        onChange={setProductNameFilter}
      />,
      <FilterableTableHeader
        key="brandName"
        label="ブランド"
        options={brandNameOptions}
        selected={brandNameFilter}
        onChange={setBrandNameFilter}
      />,
      <SortableTableHeader
        key="createdAt"
        label="問い合わせ日"
        sortKey="createdAt"
        activeKey={sortKey}
        direction={sortDir ?? null}
        onChange={(key, dir) => {
          setSortKey(key as SortKey);
          setSortDir(dir);
        }}
      />,
      <SortableTableHeader
        key="updatedAt"
        label="最終更新日"
        sortKey="updatedAt"
        activeKey={sortKey}
        direction={sortDir ?? null}
        onChange={(key, dir) => {
          setSortKey(key as SortKey);
          setSortDir(dir);
        }}
      />,
    ];
  }, [
    statusOptions,
    statusFilter,
    productNameOptions,
    productNameFilter,
    brandNameOptions,
    brandNameFilter,
    sortKey,
    sortDir,
  ]);

  const handleRefresh = useCallback(async () => {
    setIsResetting(true);
    try {
      await fetchRows();
    } finally {
      setIsResetting(false);
    }
  }, [fetchRows]);

  return (
    <div className="p-0">
      <List
        title="問い合わせ管理"
        headerCells={headers}
        showCreateButton={false}
        showResetButton
        onReset={handleRefresh}
        isResetting={isResetting}
      >
        {loading ? (
          <tr>
            <td colSpan={7}>
              <div className="inq__empty">問い合わせ一覧を読み込み中です。</div>
            </td>
          </tr>
        ) : errorMessage ? (
          <tr>
            <td colSpan={7}>
              <div className="inq__empty">{errorMessage}</div>
            </td>
          </tr>
        ) : rows.length > 0 ? (
          rows
        ) : (
          <tr>
            <td colSpan={7}>
              <div className="inq__empty">問い合わせはありません。</div>
            </td>
          </tr>
        )}
      </List>
    </div>
  );
}