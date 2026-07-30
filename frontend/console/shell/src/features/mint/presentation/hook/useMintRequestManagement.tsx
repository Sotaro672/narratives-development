// frontend/console/shell/src/features/mintRequest/presentation/hook/useMintRequestManagement.tsx

import React, {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { useNavigate } from "react-router-dom";

import {
  FilterableTableHeader,
  SortableTableHeader,
} from "../../../../layout/List/List";

import { safeDateTimeLabelJa } from "../../../../shared/util/dateJa";

import {
  applyMintRequestManagementQuery,
  type MintRequestManagementSortDirection,
  type MintRequestManagementSortKey,
} from "../../application/selector/applyMintRequestManagementQuery";

import {
  buildMintRequestManagementFilterValues,
} from "../../application/selector/buildMintRequestManagementFilterValues";

import {
  loadMintRequestManagementRows,
  type ViewRow as ManagementRow,
} from "../../application/usecase/loadMintRequestManagementRows";

import {
  inspectionStatusLabel,
} from "../formatter/inspectionStatusLabel";

/**
 * 一覧画面でのみ使用する表示用の行型。
 *
 * Application層のManagementRowへ
 * 表示用ラベルと表示形式へ変換した日時を追加する。
 */
type ManagementPresentationRow =
  ManagementRow & {
    statusLabel: string;
  };

type ManagementInspectionStatus =
  ManagementRow["inspectionStatus"];

/**
 * Application層から受け取った行を、
 * 一覧画面用の表示形式へ変換する。
 *
 * ソートは変換前のmintedAtで完了しているため、
 * ここでは表示用の日本語日時へ変換するだけとする。
 */
function toPresentationRow(
  row: ManagementRow,
): ManagementPresentationRow {
  /**
   * mintedの場合は「ミント完了」、
   * mintingの場合は「ミント中」を優先し、
   * それ以外は検品ステータスを表示する。
   */
  const statusLabel =
    row.status === "minted"
      ? "ミント完了"
      : row.status === "minting"
        ? "ミント中"
        : inspectionStatusLabel(
            row.inspectionStatus,
          );

  return {
    ...row,

    /**
     * Mint実行日時の表示を
     * yyyy/mm/dd hh:mm:ss形式へ統一する。
     */
    mintedAt:
      row.mintedAt
        ? safeDateTimeLabelJa(
            row.mintedAt,
            "",
          )
        : null,

    statusLabel,
  };
}

function asManagementInspectionStatus(
  value: string,
): ManagementInspectionStatus | null {
  if (
    value === "inspecting" ||
    value === "completed"
  ) {
    return value;
  }

  /**
   * 一覧APIでは検品レコードが存在しない場合に
   * notYetが返る。
   *
   * 現行のInspectionStatus型にはnotYetが含まれていないため、
   * ManagementRowのinspectionStatusとして扱う。
   */
  if (value === "notYet") {
    return value as ManagementInspectionStatus;
  }

  return null;
}

function getErrorMessage(
  error: unknown,
): string {
  if (
    error instanceof Error &&
    error.message
  ) {
    return error.message;
  }

  return "Failed to fetch mint requests";
}

export const useMintRequestManagement =
  () => {
    const navigate = useNavigate();

    /**
     * Application層から取得した正規化済みの行。
     *
     * 表示用日時へ変換する前の値を保持し、
     * フィルターとソートにもこの値を使用する。
     */
    const [rawRows, setRawRows] =
      useState<ManagementRow[]>([]);

    const [loading, setLoading] =
      useState(false);

    const [
      isResetting,
      setIsResetting,
    ] = useState(false);

    const [error, setError] =
      useState<string | null>(null);

    /**
     * フィルター状態。
     */
    const [
      tokenFilter,
      setTokenFilter,
    ] = useState<string[]>([]);

    const [
      productionFilter,
      setProductionFilter,
    ] = useState<string[]>([]);

    const [
      requesterFilter,
      setRequesterFilter,
    ] = useState<string[]>([]);

    const [
      statusFilter,
      setStatusFilter,
    ] = useState<
      ManagementInspectionStatus[]
    >([]);

    /**
     * デフォルトはMint実行日時の降順。
     */
    const [sortKey, setSortKey] =
      useState<MintRequestManagementSortKey>(
        "mintedAt",
      );

    const [sortDirection, setSortDirection] =
      useState<MintRequestManagementSortDirection>(
        "desc",
      );

    /**
     * 一覧データを再取得する。
     *
     * 取得とDTO変換はApplication UseCaseへ委譲し、
     * Presentation層ではstateへの反映だけを行う。
     */
    const fetchRows =
      useCallback(async () => {
        setIsResetting(true);
        setLoading(true);
        setError(null);

        try {
          const rows =
            await loadMintRequestManagementRows();

          setRawRows(rows ?? []);
        } catch (
          fetchError: unknown
        ) {
          setRawRows([]);

          setError(
            getErrorMessage(
              fetchError,
            ),
          );
        } finally {
          setLoading(false);
          setIsResetting(false);
        }
      }, []);

    /**
     * 初回表示時の一覧取得。
     */
    useEffect(() => {
      let cancelled = false;

      const run = async () => {
        setIsResetting(true);
        setLoading(true);
        setError(null);

        try {
          const rows =
            await loadMintRequestManagementRows();

          if (cancelled) {
            return;
          }

          setRawRows(rows ?? []);
        } catch (
          fetchError: unknown
        ) {
          if (cancelled) {
            return;
          }

          setRawRows([]);

          setError(
            getErrorMessage(
              fetchError,
            ),
          );
        } finally {
          if (!cancelled) {
            setLoading(false);
            setIsResetting(false);
          }
        }
      };

      void run();

      return () => {
        cancelled = true;
      };
    }, []);

    /**
     * 一覧行からフィルター候補値を抽出する。
     *
     * 候補値の重複排除はApplication Selectorへ委譲する。
     */
    const filterValues =
      useMemo(
        () =>
          buildMintRequestManagementFilterValues(
            rawRows,
          ),
        [rawRows],
      );

    /**
     * UIコンポーネント用のvalue / label形式へ変換する。
     */
    const tokenOptions =
      useMemo(
        () =>
          filterValues.tokenNames.map(
            (value) => ({
              value,
              label: value,
            }),
          ),
        [filterValues.tokenNames],
      );

    const productionOptions =
      useMemo(
        () =>
          filterValues.productNames.map(
            (value) => ({
              value,
              label: value,
            }),
          ),
        [filterValues.productNames],
      );

    const requesterOptions =
      useMemo(
        () =>
          filterValues.requesterNames.map(
            (value) => ({
              value,
              label: value,
            }),
          ),
        [filterValues.requesterNames],
      );

    const statusOptions =
      useMemo(
        () =>
          filterValues.inspectionStatuses.map(
            (value) => ({
              value,
              label:
                inspectionStatusLabel(
                  value,
                ),
            }),
          ),
        [
          filterValues
            .inspectionStatuses,
        ],
      );

    /**
     * Application Selectorでフィルターとソートを適用した後、
     * Presentation用の日時・ラベルへ変換する。
     */
    const rows =
      useMemo<
        ManagementPresentationRow[]
      >(() => {
        const queriedRows =
          applyMintRequestManagementQuery(
            rawRows,
            {
              tokenNames:
                tokenFilter,

              productNames:
                productionFilter,

              requesterNames:
                requesterFilter,

              inspectionStatuses:
                statusFilter,

              sortKey,

              sortDirection,
            },
          );

        return queriedRows.map(
          toPresentationRow,
        );
      }, [
        rawRows,
        tokenFilter,
        productionFilter,
        requesterFilter,
        statusFilter,
        sortKey,
        sortDirection,
      ]);

    /**
     * 詳細画面へ遷移する。
     */
    const goDetail =
      useCallback(
        (productionId: string) => {
          navigate(
            `/mint/${encodeURIComponent(
              productionId,
            )}`,
          );
        },
        [navigate],
      );

    /**
     * テーブルヘッダー。
     *
     * UIコンポーネントとstate更新は
     * Presentation層に残す。
     */
    const headers:
      React.ReactNode[] = [
      <FilterableTableHeader
        key="tokenName"
        label="トークン設計"
        options={tokenOptions}
        selected={tokenFilter}
        onChange={setTokenFilter}
      />,

      <FilterableTableHeader
        key="productName"
        label="プロダクト名"
        options={productionOptions}
        selected={
          productionFilter
        }
        onChange={
          setProductionFilter
        }
      />,

      <SortableTableHeader
        key="mintQuantity"
        label="Mint数量"
        sortKey="mintQuantity"
        activeKey={sortKey}
        direction={
          sortDirection
        }
        onChange={(
          key,
          direction,
        ) => {
          setSortKey(
            key as MintRequestManagementSortKey,
          );

          setSortDirection(
            direction,
          );
        }}
      />,

      <SortableTableHeader
        key="productionQuantity"
        label="生産量"
        sortKey="productionQuantity"
        activeKey={sortKey}
        direction={
          sortDirection
        }
        onChange={(
          key,
          direction,
        ) => {
          setSortKey(
            key as MintRequestManagementSortKey,
          );

          setSortDirection(
            direction,
          );
        }}
      />,

      <FilterableTableHeader
        key="status"
        label="ステータス"
        options={statusOptions}
        selected={statusFilter}
        onChange={(
          nextValues: string[],
        ) => {
          const statuses =
            nextValues
              .map(
                asManagementInspectionStatus,
              )
              .filter(
                (
                  value,
                ): value is ManagementInspectionStatus =>
                  value !== null,
              );

          setStatusFilter(
            statuses,
          );
        }}
      />,

      <FilterableTableHeader
        key="requester"
        label="リクエスト者"
        options={
          requesterOptions
        }
        selected={
          requesterFilter
        }
        onChange={
          setRequesterFilter
        }
      />,

      <SortableTableHeader
        key="mintedAt"
        label="Mint実行日時"
        sortKey="mintedAt"
        activeKey={sortKey}
        direction={
          sortDirection
        }
        onChange={(
          key,
          direction,
        ) => {
          setSortKey(
            key as MintRequestManagementSortKey,
          );

          setSortDirection(
            direction,
          );
        }}
      />,
    ];

    /**
     * フィルターとソートを初期状態へ戻し、
     * Backendから一覧を再取得する。
     */
    const onReset =
      useCallback(async () => {
        setTokenFilter([]);
        setProductionFilter([]);
        setRequesterFilter([]);
        setStatusFilter([]);

        setSortKey(
          "mintedAt",
        );

        setSortDirection(
          "desc",
        );

        await fetchRows();
      }, [fetchRows]);

    const handleRowClick =
      useCallback(
        (productionId: string) => {
          goDetail(
            productionId,
          );
        },
        [goDetail],
      );

    const handleRowKeyDown =
      useCallback(
        (
          event:
            React.KeyboardEvent<HTMLTableRowElement>,
          productionId: string,
        ) => {
          if (
            event.key === "Enter" ||
            event.key === " "
          ) {
            event.preventDefault();

            goDetail(
              productionId,
            );
          }
        },
        [goDetail],
      );

    return {
      headers,
      rows,

      onReset,

      handleRowClick,
      handleRowKeyDown,

      loading,
      isResetting,
      error,
    };
  };