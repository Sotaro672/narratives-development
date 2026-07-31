// frontend/console/shell/src/features/productBlueprintReview/presentation/hook/useProductBlueprintReviewManagement.tsx

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useNavigate } from "react-router-dom";

import {
  FetchProductBlueprintReviewManagementRows,
  FilterProductBlueprintReviewRows,
  type UiRow,
} from "../../application/productBlueprintReviewManagementService";

export interface UseProductBlueprintReviewManagementResult {
  Rows: UiRow[];

  BrandFilter: string[];
  AssigneeFilter: string[];

  HandleBrandFilterChange: (
    values: string[],
  ) => void;

  HandleAssigneeFilterChange: (
    values: string[],
  ) => void;

  HandleRowClick: (row: UiRow) => void;
  HandleReset: () => void;

  IsResetting: boolean;
}

export function useProductBlueprintReviewManagement(): UseProductBlueprintReviewManagementResult {
  const navigate = useNavigate();

  const [allRows, setAllRows] = useState<UiRow[]>([]);
  const [brandFilter, setBrandFilter] = useState<string[]>([]);
  const [assigneeFilter, setAssigneeFilter] = useState<string[]>([]);
  const [isResetting, setIsResetting] = useState(false);

  const load = useCallback(async () => {
    setIsResetting(true);

    try {
      const rows =
        await FetchProductBlueprintReviewManagementRows({});

      setAllRows(rows);
    } catch {
      setAllRows([]);
    } finally {
      setIsResetting(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const rows = useMemo(
    () =>
      FilterProductBlueprintReviewRows({
        AllRows: allRows,
        BrandFilter: brandFilter,
        AssigneeFilter: assigneeFilter,
      }),
    [
      allRows,
      brandFilter,
      assigneeFilter,
    ],
  );

  const handleBrandFilterChange = useCallback(
    (values: string[]) => {
      setBrandFilter(values);
    },
    [],
  );

  const handleAssigneeFilterChange = useCallback(
    (values: string[]) => {
      setAssigneeFilter(values);
    },
    [],
  );

  const handleRowClick = useCallback(
    (row: UiRow) => {
      const productBlueprintID = String(
        row.ProductBlueprintID || row.ID || "",
      );

      if (!productBlueprintID) {
        return;
      }

      navigate(
        `/productBlueprintReview/${encodeURIComponent(
          productBlueprintID,
        )}`,
        {
          state: {
            ProductName: String(
              row.ProductName || "",
            ),
            AssigneeName: String(
              row.AssigneeName || "",
            ),
          },
        },
      );
    },
    [navigate],
  );

  const handleReset = useCallback(() => {
    setBrandFilter([]);
    setAssigneeFilter([]);
    void load();
  }, [load]);

  return {
    Rows: rows,

    BrandFilter: brandFilter,
    AssigneeFilter: assigneeFilter,

    HandleBrandFilterChange:
      handleBrandFilterChange,

    HandleAssigneeFilterChange:
      handleAssigneeFilterChange,

    HandleRowClick: handleRowClick,
    HandleReset: handleReset,

    IsResetting: isResetting,
  };
}