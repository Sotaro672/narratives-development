// frontend/console/shell/src/features/order/presentation/hooks/useOrderManagement.tsx   
   
import React, { useCallback, useEffect, useMemo, useState } from "react";   
import { useNavigate } from "react-router-dom";   
import {   
  SortableTableHeader,   
} from "../../../../layout/List/List";   
import {   
  createOrderRepository,   
  type OrderItemInventoryRowDTO,   
} from "../../infrastructure/repository";   
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
   
  const rows = useMemo(() => {   
    return sortOrderRows(rowsRaw, activeKey, direction);   
  }, [rowsRaw, activeKey, direction]);   
   
  const headers = useMemo<React.ReactNode[]>(   
    () => [   
      "注文ID",   
      "リストID",   
      "商品名",   
      "トークン名",   
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
      "ステータス",   
    ],   
    [activeKey, direction],   
  );   
   
  const goDetail = useCallback(   
    (id: string) => {   
      navigate(`/order/${encodeURIComponent(id)}`);   
    },   
    [navigate],   
  );   
   
  const reset = useCallback(() => {   
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