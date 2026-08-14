// frontend/console/shell/src/features/inquiry/presentation/hooks/useInquiryManagementPage.ts

import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type Dispatch,
  type SetStateAction,
} from "react";
import { useNavigate } from "react-router-dom";

import { listInquiriesHTTP } from "../../infrastructure/inquiryRepositoryHTTP";
import { getInquiryStatusLabel } from "../utils/inquiryStatus";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";

import type { InquiryManagementItem } from "../../../../shared/types/inquiry";

const CURRENT_COMPANY_ID_ROUTE_PLACEHOLDER = "current";
const INQUIRY_DETAIL_ROUTE_BASE = "/inquiry";

export type InquiryManagementSortKey =
  | "createdAt"
  | "updatedAt"
  | null;

export type InquiryManagementSortDirection =
  | "asc"
  | "desc"
  | null;

export type InquiryManagementFilterOption = {
  value: string;
  label: string;
};

export type InquiryManagementRowViewModel = {
  inquiryId: string;
  subject: string;
  customerName: string;
  status: string;
  productName: string;
  brandName: string;
  createdAt: string;
  updatedAt: string;
};

export type UseInquiryManagementPageResult = {
  loading: boolean;
  isResetting: boolean;
  errorMessage: string | null;
  rows: InquiryManagementRowViewModel[];
  statusFilter: string[];
  productNameFilter: string[];
  brandNameFilter: string[];
  statusOptions: InquiryManagementFilterOption[];
  productNameOptions: InquiryManagementFilterOption[];
  brandNameOptions: InquiryManagementFilterOption[];
  sortKey: InquiryManagementSortKey;
  sortDirection: InquiryManagementSortDirection;
  setStatusFilter: Dispatch<SetStateAction<string[]>>;
  setProductNameFilter: Dispatch<SetStateAction<string[]>>;
  setBrandNameFilter: Dispatch<SetStateAction<string[]>>;
  handleSortChange: (
    key: string,
    direction: InquiryManagementSortDirection,
  ) => void;
  handleRefresh: () => Promise<void>;
  handleClickRow: (inquiryId: string) => void;
};

function textOrDash(
  value: string | null | undefined,
): string {
  const valueText = String(value ?? "").trim();

  return valueText || "-";
}

function getInquiryId(
  item: InquiryManagementItem,
): string {
  return item.inquiry.id;
}

function getSubject(
  item: InquiryManagementItem,
): string {
  return textOrDash(item.inquiry.subject);
}

function getCustomerName(
  item: InquiryManagementItem,
): string {
  return textOrDash(item.userFullName);
}

function getStatus(
  item: InquiryManagementItem,
): string {
  return getInquiryStatusLabel(
    item.inquiry.status,
    item.inquiry.isRead,
  );
}

function getProductName(
  item: InquiryManagementItem,
): string {
  return textOrDash(item.productName);
}

function getBrandName(
  item: InquiryManagementItem,
): string {
  return textOrDash(item.brandName);
}

function getCreatedAt(
  item: InquiryManagementItem,
): string {
  return safeDateTimeLabelJa(
    item.inquiry.createdAt,
    "-",
  );
}

function getUpdatedAt(
  item: InquiryManagementItem,
): string {
  return safeDateTimeLabelJa(
    item.inquiry.updatedAt,
    "-",
  );
}

function createFilterOptions(
  values: string[],
): InquiryManagementFilterOption[] {
  const seen = new Set<string>();
  const options: InquiryManagementFilterOption[] = [];

  for (const value of values) {
    if (value === "-" || seen.has(value)) {
      continue;
    }

    seen.add(value);

    options.push({
      value,
      label: value,
    });
  }

  return options;
}

function compareDateValues(
  firstValue: string,
  secondValue: string,
  direction: InquiryManagementSortDirection,
): number {
  const firstTimestamp = Date.parse(firstValue);
  const secondTimestamp = Date.parse(secondValue);

  return direction === "asc"
    ? firstTimestamp - secondTimestamp
    : secondTimestamp - firstTimestamp;
}

function toRowViewModel(
  item: InquiryManagementItem,
): InquiryManagementRowViewModel {
  return {
    inquiryId: getInquiryId(item),
    subject: getSubject(item),
    customerName: getCustomerName(item),
    status: getStatus(item),
    productName: getProductName(item),
    brandName: getBrandName(item),
    createdAt: getCreatedAt(item),
    updatedAt: getUpdatedAt(item),
  };
}

export function useInquiryManagementPage(): UseInquiryManagementPageResult {
  const navigate = useNavigate();

  const mountedRef = useRef(false);
  const latestRequestIdRef = useRef(0);

  const [items, setItems] = useState<InquiryManagementItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [isResetting, setIsResetting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<string[]>([]);
  const [productNameFilter, setProductNameFilter] = useState<string[]>([]);
  const [brandNameFilter, setBrandNameFilter] = useState<string[]>([]);
  const [sortKey, setSortKey] =
    useState<InquiryManagementSortKey>("createdAt");
  const [sortDirection, setSortDirection] =
    useState<InquiryManagementSortDirection>("desc");

  const loadInquiries = useCallback(
    async (
      showLoading: boolean,
    ): Promise<void> => {
      const requestId =
        latestRequestIdRef.current + 1;

      latestRequestIdRef.current = requestId;

      if (showLoading) {
        setLoading(true);
      }

      setErrorMessage(null);

      try {
        const result = await listInquiriesHTTP({
          /*
           * backend側ではmiddlewareのcompanyIdを正として使う。
           * route互換のためURLには空でないplaceholderを渡す。
           */
          companyId:
            CURRENT_COMPANY_ID_ROUTE_PLACEHOLDER,
        });

        if (
          !mountedRef.current ||
          requestId !== latestRequestIdRef.current
        ) {
          return;
        }

        setItems(result.items);
      } catch (error) {
        if (
          !mountedRef.current ||
          requestId !== latestRequestIdRef.current
        ) {
          return;
        }

        const message =
          error instanceof Error
            ? error.message
            : "問い合わせ一覧の取得に失敗しました";

        setErrorMessage(message);
        setItems([]);
      } finally {
        if (
          showLoading &&
          mountedRef.current &&
          requestId === latestRequestIdRef.current
        ) {
          setLoading(false);
        }
      }
    },
    [],
  );

  useEffect(() => {
    mountedRef.current = true;

    void loadInquiries(true);

    return () => {
      mountedRef.current = false;
      latestRequestIdRef.current += 1;
    };
  }, [loadInquiries]);

  const statusOptions = useMemo(() => {
    return createFilterOptions(
      items.map((item) => getStatus(item)),
    );
  }, [items]);

  const productNameOptions = useMemo(() => {
    return createFilterOptions(
      items.map((item) => getProductName(item)),
    );
  }, [items]);

  const brandNameOptions = useMemo(() => {
    return createFilterOptions(
      items.map((item) => getBrandName(item)),
    );
  }, [items]);

  const filteredItems = useMemo(() => {
    let nextItems = items.filter((item) => {
      const status = getStatus(item);
      const productName = getProductName(item);
      const brandName = getBrandName(item);

      const matchesStatus =
        statusFilter.length === 0 ||
        statusFilter.includes(status);

      const matchesProductName =
        productNameFilter.length === 0 ||
        productNameFilter.includes(productName);

      const matchesBrandName =
        brandNameFilter.length === 0 ||
        brandNameFilter.includes(brandName);

      return (
        matchesStatus &&
        matchesProductName &&
        matchesBrandName
      );
    });

    if (sortKey && sortDirection) {
      nextItems = [...nextItems].sort(
        (firstItem, secondItem) => {
          if (sortKey === "createdAt") {
            return compareDateValues(
              firstItem.inquiry.createdAt,
              secondItem.inquiry.createdAt,
              sortDirection,
            );
          }

          if (sortKey === "updatedAt") {
            return compareDateValues(
              firstItem.inquiry.updatedAt,
              secondItem.inquiry.updatedAt,
              sortDirection,
            );
          }

          return 0;
        },
      );
    }

    return nextItems;
  }, [
    items,
    statusFilter,
    productNameFilter,
    brandNameFilter,
    sortKey,
    sortDirection,
  ]);

  const rows = useMemo(() => {
    return filteredItems.map(toRowViewModel);
  }, [filteredItems]);

  const handleSortChange = useCallback(
    (
      key: string,
      direction:
        InquiryManagementSortDirection,
    ) => {
      if (
        key !== "createdAt" &&
        key !== "updatedAt"
      ) {
        setSortKey(null);
        setSortDirection(null);
        return;
      }

      setSortKey(key);
      setSortDirection(direction);
    },
    [],
  );

  const handleRefresh =
    useCallback(async (): Promise<void> => {
      setIsResetting(true);

      try {
        await loadInquiries(false);
      } finally {
        if (mountedRef.current) {
          setIsResetting(false);
        }
      }
    }, [loadInquiries]);

  const handleClickRow = useCallback(
    (inquiryId: string): void => {
      navigate(
        `${INQUIRY_DETAIL_ROUTE_BASE}/${encodeURIComponent(
          inquiryId,
        )}`,
      );
    },
    [navigate],
  );

  return {
    loading,
    isResetting,
    errorMessage,
    rows,
    statusFilter,
    productNameFilter,
    brandNameFilter,
    statusOptions,
    productNameOptions,
    brandNameOptions,
    sortKey,
    sortDirection,
    setStatusFilter,
    setProductNameFilter,
    setBrandNameFilter,
    handleSortChange,
    handleRefresh,
    handleClickRow,
  };
}