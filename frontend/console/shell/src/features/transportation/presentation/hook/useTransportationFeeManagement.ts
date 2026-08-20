// frontend/console/shell/src/features/transportation/presentation/hook/useTransportationFeeManagement.ts

import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import {
  listTransportationVMs,
  type TransportationListItemVM,
} from "../../application/transportationService";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";

export type TransportationFeeManagementRow = {
  id: string;
  companyId: string;
  name: string;
  createdAt: string;
  createdBy: string;
  createdByName: string;
  updatedAt: string;
  updatedBy: string;
  updatedByName: string;
};

export type UseTransportationFeeManagementResult = {
  rows: TransportationFeeManagementRow[];
  handlers: {
    handleCreate: () => void;
    handleRowClick: (row: TransportationFeeManagementRow) => void;
    handleReset: () => void;
  };
  isResetting: boolean;
};

function toManagementRow(
  item: TransportationListItemVM,
): TransportationFeeManagementRow {
  return {
    id: item.id,
    companyId: item.companyId,
    name: item.name,
    createdAt: safeDateTimeLabelJa(item.createdAt, ""),
    createdBy: item.createdBy,
    createdByName: item.createdByName ?? "",
    updatedAt: safeDateTimeLabelJa(item.updatedAt, ""),
    updatedBy: item.updatedBy,
    updatedByName: item.updatedByName ?? "",
  };
}

export function useTransportationFeeManagement(): UseTransportationFeeManagementResult {
  const navigate = useNavigate();

  const [items, setItems] = useState<TransportationListItemVM[]>([]);
  const [isResetting, setIsResetting] = useState(false);

  const load = useCallback(async (): Promise<void> => {
    setIsResetting(true);

    try {
      const result = await listTransportationVMs();
      setItems(result);
    } catch {
      setItems([]);
    } finally {
      setIsResetting(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const rows = useMemo<TransportationFeeManagementRow[]>(
    () => items.map(toManagementRow),
    [items],
  );

  const handleCreate = useCallback(() => {
    navigate("/transportationFee/create");
  }, [navigate]);

  const handleRowClick = useCallback(
    (row: TransportationFeeManagementRow) => {
      if (!row.id) return;

      navigate(
        `/transportationFee/${encodeURIComponent(row.id)}`,
      );
    },
    [navigate],
  );

  const handleReset = useCallback(() => {
    void load();
  }, [load]);

  return {
    rows,
    handlers: {
      handleCreate,
      handleRowClick,
      handleReset,
    },
    isResetting,
  };
}