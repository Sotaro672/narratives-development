// frontend/console/shell/src/features/brand/presentation/hook/useBrandManagement.ts

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import type { SortOrder } from "../../../../shared/types/common/common";
import { safeDateLabelJa } from "../../../../shared/util/dateJa";

import type { BrandRow as BrandRowBase } from "../../application/brandService";
import { listBrands } from "../../application/brandService";

export type SortKey =
  | "registeredAt"
  | "updatedAt"
  | null;

export type StatusFilterValue =
  | "active"
  | "inactive";

export type BrandRow = BrandRowBase & {
  updatedAt: string;
};

type ManagerOption = {
  value: string;
  label: string;
};

const toTs = (value: string): number => {
  const normalizedDate = safeDateLabelJa(
    value,
    "",
  );

  if (!normalizedDate) {
    return 0;
  }

  const [year, month, day] = normalizedDate
    .split("/")
    .map((part) => parseInt(part, 10));

  if (
    !Number.isFinite(year) ||
    !Number.isFinite(month) ||
    !Number.isFinite(day)
  ) {
    return 0;
  }

  return new Date(
    year,
    month - 1,
    day,
  ).getTime();
};

export function useBrandManagement() {
  const [baseRows, setBaseRows] = useState<
    BrandRow[]
  >([]);

  const [loading, setLoading] = useState(false);

  const [error, setError] =
    useState<Error | null>(null);

  const [isResetting, setIsResetting] =
    useState(false);

  const [statusFilter, setStatusFilter] =
    useState<StatusFilterValue[]>([]);

  const [managerFilter, setManagerFilter] =
    useState<string[]>([]);

  const [activeKey, setActiveKey] =
    useState<SortKey>("registeredAt");

  const [direction, setDirection] =
    useState<SortOrder | null>("desc");

  const [reloadKey, setReloadKey] =
    useState(0);

  const [managerOptions, setManagerOptions] =
    useState<ManagerOption[]>([]);

  const statusBadgeClass = (
    isActive: boolean,
  ): string =>
    `brand-status-badge ${
      isActive ? "is-active" : "is-inactive"
    }`;

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      try {
        setLoading(true);
        setIsResetting(true);
        setError(null);

        const rawRows = await listBrands();

        const rows: BrandRow[] = rawRows.map(
          (brand) => {
            const registeredAt =
              safeDateLabelJa(
                brand.registeredAt,
                "",
              );

            const updatedAt =
              safeDateLabelJa(
                brand.updatedAt,
                registeredAt,
              );

            return {
              ...brand,
              registeredAt,
              updatedAt,
            };
          },
        );

        if (!cancelled) {
          setBaseRows(rows);
        }
      } catch (error: unknown) {
        if (!cancelled) {
          const normalizedError =
            error instanceof Error
              ? error
              : new Error(String(error));

          setError(normalizedError);
          setBaseRows([]);
          setManagerOptions([]);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
          setIsResetting(false);
        }
      }
    };

    void load();

    return () => {
      cancelled = true;
    };
  }, [reloadKey]);

  useEffect(() => {
    const seen = new Set<string>();
    const options: ManagerOption[] = [];

    for (const brand of baseRows) {
      const managerId =
        brand.managerId ?? "";

      if (
        !managerId ||
        seen.has(managerId)
      ) {
        continue;
      }

      seen.add(managerId);

      const memberName =
        brand.memberName ?? "";

      options.push({
        value: managerId,
        label:
          memberName !== ""
            ? memberName
            : managerId,
      });
    }

    setManagerOptions(options);
  }, [baseRows]);

  const statusOptions = useMemo(() => {
    const values = Array.from(
      new Set<StatusFilterValue>(
        baseRows.map((brand) =>
          brand.isActive
            ? "active"
            : "inactive",
        ),
      ),
    );

    return values.map((value) => ({
      value,
      label:
        value === "active"
          ? "アクティブ"
          : "停止",
    }));
  }, [baseRows]);

  const rows = useMemo(() => {
    let filteredRows = baseRows.filter(
      (brand) => {
        const statusValue:
          StatusFilterValue =
          brand.isActive
            ? "active"
            : "inactive";

        const statusMatches =
          statusFilter.length === 0 ||
          statusFilter.includes(
            statusValue,
          );

        const managerId =
          brand.managerId ?? "";

        const managerMatches =
          managerFilter.length === 0 ||
          (
            managerId !== "" &&
            managerFilter.includes(
              managerId,
            )
          );

        return (
          statusMatches &&
          managerMatches
        );
      },
    );

    if (activeKey && direction) {
      filteredRows = [
        ...filteredRows,
      ].sort(
        (
          firstBrand,
          secondBrand,
        ) => {
          if (
            activeKey ===
            "registeredAt"
          ) {
            const firstValue = toTs(
              firstBrand.registeredAt,
            );

            const secondValue = toTs(
              secondBrand.registeredAt,
            );

            return direction === "asc"
              ? firstValue - secondValue
              : secondValue - firstValue;
          }

          if (
            activeKey === "updatedAt"
          ) {
            const firstValue = toTs(
              firstBrand.updatedAt,
            );

            const secondValue = toTs(
              secondBrand.updatedAt,
            );

            return direction === "asc"
              ? firstValue - secondValue
              : secondValue - firstValue;
          }

          return 0;
        },
      );
    }

    return filteredRows;
  }, [
    baseRows,
    statusFilter,
    managerFilter,
    activeKey,
    direction,
  ]);

  const resetFilters = useCallback(() => {
    setStatusFilter([]);
    setManagerFilter([]);
    setActiveKey("registeredAt");
    setDirection("desc");

    setReloadKey(
      (current) => current + 1,
    );
  }, []);

  return {
    rows,
    statusOptions,
    managerOptions,

    loading,
    error,
    isResetting,

    statusFilter,
    managerFilter,
    activeKey,
    direction,

    setStatusFilter,
    setManagerFilter,
    setActiveKey,
    setDirection,

    statusBadgeClass,
    resetFilters,
  };
}