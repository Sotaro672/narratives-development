// frontend/console/shell/src/features/order/presentation/hooks/useOrderManagement.tsx    
    
import React, { useCallback, useEffect, useMemo, useState } from "react";    
import { useNavigate } from "react-router-dom";    
import {    
  FilterableTableHeader,    
  SortableTableHeader,    
} from "../../../../layout/List/List";    
import {    
  createOrderRepository,    
  type OrderItemInventoryRowDTO,    
} from "../../infrastructure/repository";    
import {    
  filterOrderRows,    
  getOrderManagementStatus,    
} from "../../application/orderManagementFilter";    
import {    
  sortOrderRows,    
  type SortDir,    
  type SortKey,    
} from "../../application/orderManagementSort";    
    
export function useOrderManagement() {    
  const navigate = useNavigate();    
  const repo = useMemo(() => createOrderRepository(), []);    
    
  const [activeKey, setActiveKey] = useState<SortKey>("createdAt");    
  const [direction, setDirection] = useState<SortDir>("desc");    
  const [rowsRaw, setRowsRaw] = useState<OrderItemInventoryRowDTO[]>([]);    
  const [listIdFilter, setListIdFilter] = useState<string[]>([]);    
  const [productNameFilter, setProductNameFilter] = useState<string[]>([]);    
  const [tokenNameFilter, setTokenNameFilter] = useState<string[]>([]);    
  const [statusFilter, setStatusFilter] = useState<string[]>([]);    
  const [errorMsg, setErrorMsg] = useState<string | null>(null);    
  const [isResetting, setIsResetting] = useState(false);    
    
  const fetchRows = useCallback(async () => {    
    setIsResetting(true);    
    setErrorMsg(null);    
    
    try {    
      const res = await repo.listItemInventoryRows({ page: 1, perPage: 200 });    
      setRowsRaw(res.items);    
    } catch (e) {    
      setRowsRaw([]);    
      setErrorMsg(e instanceof Error ? e.message : "failed_to_fetch_orders");    
    } finally {    
      setIsResetting(false);    
    }    
  }, [repo]);    
    
  useEffect(() => {    
    void fetchRows();    
  }, [fetchRows]);    
    
  const listIdOptions = useMemo(() => {    
    return Array.from(    
      new Set(    
        rowsRaw    
          .map((row) => row.listReadableId)    
          .filter((value): value is string => Boolean(value)),    
      ),    
    ).map((value) => ({ value, label: value }));    
  }, [rowsRaw]);    
    
  const productNameOptions = useMemo(() => {    
    return Array.from(    
      new Set(    
        rowsRaw    
          .map((row) => row.productName)    
          .filter((value): value is string => Boolean(value)),    
      ),    
    ).map((value) => ({ value, label: value }));    
  }, [rowsRaw]);    
    
  const tokenNameOptions = useMemo(() => {    
    return Array.from(    
      new Set(    
        rowsRaw    
          .map((row) => row.tokenName)    
          .filter((value): value is string => Boolean(value)),    
      ),    
    ).map((value) => ({ value, label: value }));    
  }, [rowsRaw]);    
    
  const statusOptions = useMemo(() => {    
    return Array.from(    
      new Set(    
        rowsRaw.map((row) => getOrderManagementStatus(row)),    
      ),    
    ).map((value) => ({ value, label: value }));    
  }, [rowsRaw]);    
    
  const rows = useMemo(() => {    
    const filteredRows = filterOrderRows(rowsRaw, {    
      listIds: listIdFilter,    
      productNames: productNameFilter,    
      tokenNames: tokenNameFilter,    
      statuses: statusFilter,    
    });    
    
    return sortOrderRows(filteredRows, activeKey, direction);    
  }, [    
    rowsRaw,    
    listIdFilter,    
    productNameFilter,    
    tokenNameFilter,    
    statusFilter,    
    activeKey,    
    direction,    
  ]);    
    
  const headers = useMemo<React.ReactNode[]>(    
    () => [    
      "注文ID",    
      <FilterableTableHeader    
        key="listId"    
        label="リストID"    
        options={listIdOptions}    
        selected={listIdFilter}    
        onChange={setListIdFilter}    
      />,    
      <FilterableTableHeader    
        key="productName"    
        label="商品名"    
        options={productNameOptions}    
        selected={productNameFilter}    
        onChange={setProductNameFilter}    
      />,    
      <FilterableTableHeader    
        key="tokenName"    
        label="トークン名"    
        options={tokenNameOptions}    
        selected={tokenNameFilter}    
        onChange={setTokenNameFilter}    
      />,    
      "ユーザー名",    
      <SortableTableHeader    
        key="createdAt"    
        label="注文日"    
        sortKey="createdAt"    
        activeKey={activeKey}    
        direction={activeKey === "createdAt" ? direction : null}    
        onChange={(key, dir) => {    
          setActiveKey(key as SortKey);    
          setDirection(dir as SortDir);    
        }}    
      />,    
      <FilterableTableHeader    
        key="status"    
        label="ステータス"    
        options={statusOptions}    
        selected={statusFilter}    
        onChange={setStatusFilter}    
      />,    
    ],    
    [    
      listIdOptions,    
      productNameOptions,    
      tokenNameOptions,    
      statusOptions,    
      listIdFilter,    
      productNameFilter,    
      tokenNameFilter,    
      statusFilter,    
      activeKey,    
      direction,    
    ],    
  );    
    
  const goDetail = useCallback(    
    (id: string) => {    
      navigate(`/order/${encodeURIComponent(id)}`);    
    },    
    [navigate],    
  );    
    
  const reset = useCallback(() => {    
    setListIdFilter([]);    
    setProductNameFilter([]);    
    setTokenNameFilter([]);    
    setStatusFilter([]);    
    setActiveKey("createdAt");    
    setDirection("desc");    
    void fetchRows();    
  }, [fetchRows]);    
    
  return {    
    rows,    
    headers,    
    errorMsg,    
    isResetting,    
    goDetail,    
    reset,    
  };    
}