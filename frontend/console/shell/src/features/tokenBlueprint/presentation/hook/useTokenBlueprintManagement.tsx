// frontend/console/shell/src/features/tokenBlueprint/presentation/hook/useTokenBlueprintManagement.tsx

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useNavigate } from "react-router-dom";

import {
  useAuthContext,
} from "../../../../auth/application/AuthContext";

import type { TokenBlueprint } from "../../../../shared/types/tokenBlueprint";

import {
  buildOptionsFromTokenBlueprints,
  fetchTokenBlueprintsForCompany,
  filterAndSortTokenBlueprints,
  type SortDir,
  type SortKey,
} from "../../application/tokenBlueprintManagementService";

export type UseTokenBlueprintManagementResult = {
  rows: TokenBlueprint[];

  brandOptions: {
    value: string;
    label: string;
  }[];

  assigneeOptions: {
    value: string;
    label: string;
  }[];

  mintedOptions: {
    value: string;
    label: string;
  }[];

  brandFilter: string[];
  assigneeFilter: string[];
  mintedFilter: string[];

  sortKey: SortKey;
  sortDir: SortDir;

  isResetting: boolean;

  handleChangeBrandFilter: (
    values: string[],
  ) => void;

  handleChangeAssigneeFilter: (
    values: string[],
  ) => void;

  handleChangeMintedFilter: (
    values: string[],
  ) => void;

  handleChangeSort: (
    key: string | null,
    direction: SortDir,
  ) => void;

  handleReset: () => void;
  handleCreate: () => void;

  handleRowClick: (
    id: string,
  ) => void;
};

/**
 * ISO 8601文字列をyyyy/MM/dd HH:mm形式に整形する。
 *
 * 例:
 * 2026/01/24 13:05
 */
function formatDateYYYYMMDDHHmm(
  iso: string,
): string {
  if (!iso) {
    return "";
  }

  const date =
    new Date(iso);

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return iso;
  }

  const year =
    date.getFullYear();

  const month =
    String(
      date.getMonth() + 1,
    ).padStart(
      2,
      "0",
    );

  const day =
    String(
      date.getDate(),
    ).padStart(
      2,
      "0",
    );

  const hour =
    String(
      date.getHours(),
    ).padStart(
      2,
      "0",
    );

  const minute =
    String(
      date.getMinutes(),
    ).padStart(
      2,
      "0",
    );

  return `${year}/${month}/${day} ${hour}:${minute}`;
}

export function useTokenBlueprintManagement(): UseTokenBlueprintManagementResult {
  const navigate =
    useNavigate();

  const {
    currentMember,
  } = useAuthContext();

  const [
    rows,
    setRows,
  ] = useState<
    TokenBlueprint[]
  >([]);

  const [
    brandFilter,
    setBrandFilter,
  ] = useState<string[]>(
    [],
  );

  const [
    assigneeFilter,
    setAssigneeFilter,
  ] = useState<string[]>(
    [],
  );

  const [
    mintedFilter,
    setMintedFilter,
  ] = useState<string[]>(
    [],
  );

  const [
    sortKey,
    setSortKey,
  ] = useState<SortKey>(
    null,
  );

  const [
    sortDir,
    setSortDir,
  ] = useState<SortDir>(
    null,
  );

  const [
    isResetting,
    setIsResetting,
  ] = useState<boolean>(
    false,
  );

  /**
   * 認証中の会社に属するTokenBlueprint一覧を再取得する。
   *
   * companyIdはリクエスト引数として送信せず、
   * backendが認証コンテキストから会社境界を判定する。
   */
  const reload =
    useCallback(
      async (): Promise<void> => {
        if (
          !currentMember?.companyId
        ) {
          setRows(
            [],
          );

          return;
        }

        setIsResetting(
          true,
        );

        try {
          const result =
            await fetchTokenBlueprintsForCompany();

          setRows(
            result,
          );
        } catch {
          setRows(
            [],
          );
        } finally {
          setIsResetting(
            false,
          );
        }
      },
      [
        currentMember?.companyId,
      ],
    );

  useEffect(() => {
    void reload();
  }, [reload]);

  const {
    brandOptions,
    assigneeOptions,
  } = useMemo(() => {
    const baseOptions =
      buildOptionsFromTokenBlueprints(
        rows,
      );

    const brandNameById =
      new Map<
        string,
        string
      >();

    const assigneeNameById =
      new Map<
        string,
        string
      >();

    for (
      const row of rows
    ) {
      if (
        row.brandId &&
        row.brandName &&
        !brandNameById.has(
          row.brandId,
        )
      ) {
        brandNameById.set(
          row.brandId,
          row.brandName,
        );
      }

      if (
        row.assigneeId &&
        row.assigneeName &&
        !assigneeNameById.has(
          row.assigneeId,
        )
      ) {
        assigneeNameById.set(
          row.assigneeId,
          row.assigneeName,
        );
      }
    }

    const nextBrandOptions =
      baseOptions.brandOptions.map(
        (option) => {
          return {
            ...option,

            label:
              brandNameById.get(
                option.value,
              ) ||
              option.label ||
              option.value,
          };
        },
      );

    const nextAssigneeOptions =
      baseOptions.assigneeOptions.map(
        (option) => {
          return {
            ...option,

            label:
              assigneeNameById.get(
                option.value,
              ) ||
              option.label ||
              option.value,
          };
        },
      );

    return {
      brandOptions:
        nextBrandOptions,

      assigneeOptions:
        nextAssigneeOptions,
    };
  }, [rows]);

  const mintedOptions =
    useMemo(
      () => {
        return [
          {
            value: "true",
            label: "true",
          },
          {
            value: "false",
            label: "false",
          },
        ];
      },
      [],
    );

  const filteredRows =
    useMemo<
      TokenBlueprint[]
    >(() => {
      const baseRows =
        filterAndSortTokenBlueprints(
          rows,
          {
            brandFilter,
            assigneeFilter,
            sortKey,
            sortDir,
          },
        );

      if (
        mintedFilter.length === 0
      ) {
        return baseRows;
      }

      return baseRows.filter(
        (tokenBlueprint) => {
          const mintedValue =
            String(
              tokenBlueprint.minted,
            );

          return mintedFilter.includes(
            mintedValue,
          );
        },
      );
    }, [
      rows,
      brandFilter,
      assigneeFilter,
      mintedFilter,
      sortKey,
      sortDir,
    ]);

  const displayRows =
    useMemo<
      TokenBlueprint[]
    >(() => {
      return filteredRows.map(
        (tokenBlueprint) => {
          return {
            ...tokenBlueprint,

            createdAt:
              tokenBlueprint.createdAt
                ? formatDateYYYYMMDDHHmm(
                    tokenBlueprint.createdAt,
                  )
                : tokenBlueprint.createdAt,

            updatedAt:
              tokenBlueprint.updatedAt
                ? formatDateYYYYMMDDHHmm(
                    tokenBlueprint.updatedAt,
                  )
                : tokenBlueprint.updatedAt,
          };
        },
      );
    }, [filteredRows]);

  const handleRowClick =
    useCallback(
      (
        id: string,
      ) => {
        navigate(
          `/tokenBlueprint/${encodeURIComponent(id)}`,
        );
      },
      [navigate],
    );

  const handleCreate =
    useCallback(() => {
      navigate(
        "/tokenBlueprint/create",
      );
    }, [navigate]);

  const handleReset =
    useCallback(() => {
      setBrandFilter(
        [],
      );

      setAssigneeFilter(
        [],
      );

      setMintedFilter(
        [],
      );

      setSortKey(
        null,
      );

      setSortDir(
        null,
      );

      void reload();
    }, [reload]);

  const handleChangeBrandFilter =
    useCallback(
      (
        values: string[],
      ) => {
        setBrandFilter(
          values,
        );
      },
      [],
    );

  const handleChangeAssigneeFilter =
    useCallback(
      (
        values: string[],
      ) => {
        setAssigneeFilter(
          values,
        );
      },
      [],
    );

  const handleChangeMintedFilter =
    useCallback(
      (
        values: string[],
      ) => {
        setMintedFilter(
          values,
        );
      },
      [],
    );

  const handleChangeSort =
    useCallback(
      (
        key: string | null,
        direction: SortDir,
      ) => {
        setSortKey(
          key === "createdAt"
            ? "createdAt"
            : null,
        );

        setSortDir(
          direction,
        );
      },
      [],
    );

  return {
    rows:
      displayRows,

    brandOptions,
    assigneeOptions,
    mintedOptions,

    brandFilter,
    assigneeFilter,
    mintedFilter,

    sortKey,
    sortDir,

    isResetting,

    handleChangeBrandFilter,
    handleChangeAssigneeFilter,
    handleChangeMintedFilter,
    handleChangeSort,

    handleReset,
    handleCreate,
    handleRowClick,
  };
}