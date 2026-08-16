// frontend/console/shell/src/features/brand/presentation/hook/useBrandManagement.ts

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";

import type { SortOrder } from "../../../../shared/types/common/common";
import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";
import type { BrandRow } from "../../application/brandService";
import { listBrands } from "../../application/brandService";

export type SortKey = "registeredAt" | "updatedAt" | null;
export type StatusFilterValue = "active" | "inactive";

type ManagerOption = {
  value: string;
  label: string;
};

const toTs = (value: string): number => {
  const normalized = safeDateTimeLabelJa(value, "");
  if (!normalized) return 0;

  const matched = normalized.match(
    /^(\d{4})\/(\d{2})\/(\d{2}) (\d{2}):(\d{2})$/,
  );

  if (!matched) return 0;

  const year = Number(matched[1]);
  const month = Number(matched[2]);
  const day = Number(matched[3]);
  const hour = Number(matched[4]);
  const minute = Number(matched[5]);

  if (
    !Number.isFinite(year) ||
    !Number.isFinite(month) ||
    !Number.isFinite(day) ||
    !Number.isFinite(hour) ||
    !Number.isFinite(minute)
  ) {
    return 0;
  }

  return new Date(
    year,
    month - 1,
    day,
    hour,
    minute,
  ).getTime();
};

export function useBrandManagement() {
  const [baseRows, setBaseRows] = useState<BrandRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [isResetting, setIsResetting] = useState(false);
  const [statusFilter, setStatusFilter] = useState<StatusFilterValue[]>([]);
  const [managerFilter, setManagerFilter] = useState<string[]>([]);
  const [activeKey, setActiveKey] = useState<SortKey>("registeredAt");
  const [direction, setDirection] = useState<SortOrder | null>("desc");
  const [reloadKey, setReloadKey] = useState(0);

  const statusBadgeClass = (isActive: boolean): string =>
    `brand-status-badge ${isActive ? "is-active" : "is-inactive"}`;

  useEffect(() => {
    let cancelled = false;

    const load = async () => {
      try {
        setLoading(true);
        setIsResetting(true);
        setError(null);

        const rows = await listBrands();

        if (!cancelled) {
          setBaseRows(rows);
        }
      } catch (error: unknown) {
        if (!cancelled) {
          setError(
            error instanceof Error
              ? error
              : new Error(String(error)),
          );
          setBaseRows([]);
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

  const managerOptions = useMemo<ManagerOption[]>(() => {
    const seen = new Set<string>();
    const options: ManagerOption[] = [];

    for (const brand of baseRows) {
      const managerId = brand.managerId;

      if (!managerId || seen.has(managerId)) continue;

      seen.add(managerId);

      options.push({
        value: managerId,
        label: brand.memberName || managerId,
      });
    }

    return options;
  }, [baseRows]);

  const statusOptions = useMemo(() => {
    const values = Array.from(
      new Set<StatusFilterValue>(
        baseRows.map((brand) =>
          brand.isActive ? "active" : "inactive",
        ),
      ),
    );

    return values.map((value) => ({
      value,
      label: value === "active" ? "アクティブ" : "停止",
    }));
  }, [baseRows]);

  const rows = useMemo(() => {
    let filteredRows = baseRows.filter((brand) => {
      const statusValue: StatusFilterValue =
        brand.isActive ? "active" : "inactive";

      const statusMatches =
        statusFilter.length === 0 ||
        statusFilter.includes(statusValue);

      const managerMatches =
        managerFilter.length === 0 ||
        (
          brand.managerId != null &&
          managerFilter.includes(brand.managerId)
        );

      return statusMatches && managerMatches;
    });

    if (activeKey && direction) {
      filteredRows = [...filteredRows].sort(
        (firstBrand, secondBrand) => {
          const firstValue = toTs(firstBrand[activeKey]);
          const secondValue = toTs(secondBrand[activeKey]);

          return direction === "asc"
            ? firstValue - secondValue
            : secondValue - firstValue;
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
    setReloadKey((current) => current + 1);
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