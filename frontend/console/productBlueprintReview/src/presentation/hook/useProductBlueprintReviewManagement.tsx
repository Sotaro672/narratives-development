// frontend/console/productBlueprintReview/src/presentation/hook/useProductBlueprintReviewManagement.tsx

import { useMemo, useState, useCallback, useEffect } from "react";
import { useNavigate } from "react-router-dom";

import {
  FetchProductBlueprintReviewManagementRows,
  FilterAndSortProductBlueprintReviewRows,
  type UiRow,
  type ProductBlueprintReviewSortKey,
  type SortDirection,
} from "../../application/productBlueprintReviewManagementService";

import { safeDateTimeLabelJa } from "../../../../shell/src/shared/util/dateJa";

export interface UseProductBlueprintReviewManagementResult {
  Rows: UiRow[];

  BrandFilter: string[];
  AssigneeFilter: string[];

  HandleBrandFilterChange: (Values: string[]) => void;
  HandleAssigneeFilterChange: (Values: string[]) => void;

  HandleSortChange: (Key: string | null, Dir: "Asc" | "Desc" | null) => void;

  HandleRowClick: (Row: UiRow) => void;
  HandleReset: () => void;

  IsResetting: boolean;
}

function FormatDateTimeYYYYMMDDHHmm(V: string | null | undefined): string {
  const Label = safeDateTimeLabelJa(V, "");
  if (!Label) return "";

  const M = Label.match(/^(\d{4}\/\d{2}\/\d{2} \d{2}:\d{2})(?::\d{2})?$/);
  if (M) return M[1];

  return Label;
}

export function useProductBlueprintReviewManagement(): UseProductBlueprintReviewManagementResult {
  const Navigate = useNavigate();

  const [AllRows, SetAllRows] = useState<UiRow[]>([]);
  const [BrandFilter, SetBrandFilter] = useState<string[]>([]);
  const [AssigneeFilter, SetAssigneeFilter] = useState<string[]>([]);
  const [SortedKey, SetSortedKey] = useState<ProductBlueprintReviewSortKey>(null);
  const [SortedDir, SetSortedDir] = useState<SortDirection>(null);
  const [IsResetting, SetIsResetting] = useState<boolean>(false);

  const Load = useCallback(async () => {
    SetIsResetting(true);
    try {
      const UiRows = await FetchProductBlueprintReviewManagementRows({});
      SetAllRows(UiRows);
    } catch {
      SetAllRows([]);
    } finally {
      SetIsResetting(false);
    }
  }, []);

  useEffect(() => {
    void Load();
  }, [Load]);

  const FilteredSortedRows: UiRow[] = useMemo(
    () =>
      FilterAndSortProductBlueprintReviewRows({
        AllRows,
        BrandFilter,
        AssigneeFilter,
        SortedKey,
        SortedDir,
      }),
    [AllRows, BrandFilter, AssigneeFilter, SortedKey, SortedDir],
  );

  const Rows: UiRow[] = useMemo(() => {
    return FilteredSortedRows.map((R: any) => ({
      ...R,
      CreatedAt: FormatDateTimeYYYYMMDDHHmm(R.CreatedAt),
      UpdatedAt: FormatDateTimeYYYYMMDDHHmm(R.UpdatedAt),
    }));
  }, [FilteredSortedRows]);

  const HandleBrandFilterChange = useCallback((Values: string[]) => {
    SetBrandFilter(Values);
  }, []);

  const HandleAssigneeFilterChange = useCallback((Values: string[]) => {
    SetAssigneeFilter(Values);
  }, []);

  const HandleSortChange = useCallback(
    (Key: string | null, Dir: "Asc" | "Desc" | null) => {
      SetSortedKey((Key as ProductBlueprintReviewSortKey) ?? null);
      SetSortedDir(Dir as SortDirection);
    },
    [],
  );

  const HandleRowClick = useCallback(
    (Row: UiRow) => {
      // ✅ B案のURLに統一: /productBlueprintReview/<id>
      const ProductBlueprintID = String((Row as any).ProductBlueprintID || (Row as any).ID || "");
      const ProductName = String((Row as any).ProductName || "");
      const AssigneeName = String((Row as any).AssigneeName || "");

      Navigate(`/productBlueprintReview/${encodeURIComponent(ProductBlueprintID)}`, {
        state: {
          ProductBlueprintID,
          ProductName,
          AssigneeName,
        },
      });
    },
    [Navigate],
  );

  const HandleReset = useCallback(() => {
    SetBrandFilter([]);
    SetAssigneeFilter([]);
    SetSortedKey(null);
    SetSortedDir(null);
    void Load();
  }, [Load]);

  return {
    Rows,
    BrandFilter,
    AssigneeFilter,
    HandleBrandFilterChange,
    HandleAssigneeFilterChange,
    HandleSortChange,
    HandleRowClick,
    HandleReset,
    IsResetting,
  };
}