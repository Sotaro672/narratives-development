// frontend/admin/shell/src/features/trade/presentation/components/TradeMessageTable.tsx

import { useMemo } from "react";
import { useNavigate } from "react-router-dom";

import type {
  ReportCase,
  ReportItem,
} from "../../../../shared/type/report";
import type {
  TradeMessage,
  TradeMessageSenderSide,
} from "../../../../shared/type/trade";
import Table, { type TableColumn } from "../../../../shared/ui/Table/Table";
import TextLink from "../../../../shared/ui/TextLink/TextLink";
import { formatDateTime } from "../../../../shared/util/dateFormat";
import { useTradeMessages } from "../hooks/useTradeMessages";

type TradeMessageTableProps = {
  tradeId: string;
  sellerAvatarName?: string;
  returnReportCase?: ReportCase | null;
  returnReport?: ReportItem | null;
};

type TradeMessageTableRow =
  | {
      type: "message";
      id: string;
      createdAt: string;
      message: TradeMessage;
    }
  | {
      type: "returnReport";
      id: string;
      createdAt: string;
      reportCase: ReportCase;
      report: ReportItem;
    };

type TradeParticipantNames = {
  buyer?: string;
  seller?: string;
};

const SENDER_SIDE_LABELS: Record<TradeMessageSenderSide, string> = {
  buyer: "購入者",
  seller: "出品者",
  system: "システム",
};

function getSystemMessageActorSide(
  message: TradeMessage,
): "buyer" | "seller" | null {
  if (
    message.senderSide !== "system" ||
    message.senderType !== "system"
  ) {
    return null;
  }

  const messageId = message.id.trim();

  if (messageId === "dispatch") {
    return "seller";
  }

  if (messageId === "return-consultation") {
    return "buyer";
  }

  if (
    messageId.startsWith("return-proposal-accepted-") ||
    messageId.startsWith("return-proposal-rejected-") ||
    messageId.startsWith("return-shipment-ready-")
  ) {
    return "buyer";
  }

  if (
    messageId.startsWith("return-proposal-") ||
    messageId.startsWith("return-completed-")
  ) {
    return "seller";
  }

  return null;
}

function getDisplaySenderSide(
  message: TradeMessage,
): TradeMessageSenderSide {
  if (message.senderSide !== "system") {
    return message.senderSide;
  }

  return getSystemMessageActorSide(message) ?? "system";
}

function getDisplaySenderName(
  message: TradeMessage,
  displaySenderSide: TradeMessageSenderSide,
  participantNames: TradeParticipantNames,
): string {
  if (displaySenderSide === "system") {
    return message.senderName || "AMOL";
  }

  if (
    message.senderSide === displaySenderSide &&
    message.senderType === "avatar"
  ) {
    const senderName = message.senderName.trim();

    if (senderName) {
      return senderName;
    }
  }

  const participantName = participantNames[displaySenderSide]?.trim();

  if (participantName) {
    return participantName;
  }

  return SENDER_SIDE_LABELS[displaySenderSide];
}

function getCreatedAtTime(value: string): number {
  const time = new Date(value).getTime();
  return Number.isNaN(time) ? 0 : time;
}

export default function TradeMessageTable({
  tradeId,
  sellerAvatarName,
  returnReportCase,
  returnReport,
}: TradeMessageTableProps) {
  const navigate = useNavigate();
  const { messages, loading, error, reload } = useTradeMessages(tradeId);

  const participantNames = useMemo<TradeParticipantNames>(() => {
    const result: TradeParticipantNames = {
      seller: sellerAvatarName?.trim() || undefined,
    };

    for (const message of messages?.items ?? []) {
      if (
        message.senderType !== "avatar" ||
        (message.senderSide !== "buyer" &&
          message.senderSide !== "seller")
      ) {
        continue;
      }

      const senderName = message.senderName.trim();

      if (!senderName) {
        continue;
      }

      if (message.senderSide === "buyer" && !result.buyer) {
        result.buyer = senderName;
      }

      if (message.senderSide === "seller" && !result.seller) {
        result.seller = senderName;
      }
    }

    const reporterName = returnReport?.reporterName.trim();

    if (!result.buyer && reporterName) {
      result.buyer = reporterName;
    }

    return result;
  }, [messages, returnReport, sellerAvatarName]);

  const rows = useMemo<TradeMessageTableRow[]>(() => {
    const result: TradeMessageTableRow[] = (messages?.items ?? []).map(
      (message) => ({
        type: "message",
        id: `message:${message.id}`,
        createdAt: message.createdAt,
        message,
      }),
    );

    if (returnReportCase && returnReport) {
      result.push({
        type: "returnReport",
        id: `return-report:${returnReport.id}`,
        createdAt: returnReport.createdAt,
        reportCase: returnReportCase,
        report: returnReport,
      });
    }

    result.sort((left, right) => {
      const leftTime = getCreatedAtTime(left.createdAt);
      const rightTime = getCreatedAtTime(right.createdAt);

      if (leftTime !== rightTime) {
        return leftTime - rightTime;
      }

      return left.id.localeCompare(right.id);
    });

    return result;
  }, [messages, returnReport, returnReportCase]);

  const columns = useMemo<TableColumn<TradeMessageTableRow>[]>(
    () => [
      {
        key: "senderName",
        header: "送信者",
        render: (row) => {
          if (row.type === "returnReport") {
            const name =
              row.report.reporterName.trim() ||
              participantNames.buyer ||
              "購入者";

            return `${name}（購入者）`;
          }

          const displaySenderSide = getDisplaySenderSide(row.message);
          const name = getDisplaySenderName(
            row.message,
            displaySenderSide,
            participantNames,
          );
          const sideLabel = SENDER_SIDE_LABELS[displaySenderSide];

          return `${name}（${sideLabel}）`;
        },
        sortValue: (row) => {
          if (row.type === "returnReport") {
            return (
              row.report.reporterName.trim() ||
              participantNames.buyer ||
              "購入者"
            );
          }

          const displaySenderSide = getDisplaySenderSide(row.message);

          return getDisplaySenderName(
            row.message,
            displaySenderSide,
            participantNames,
          );
        },
        nowrap: true,
      },
      {
        key: "content",
        header: "メッセージ",
        render: (row) => {
          if (row.type === "returnReport") {
            return (
              <>
                <TextLink
                  tone="accent"
                  onClick={() =>
                    navigate(
                      `/reports/${encodeURIComponent(row.reportCase.id)}`,
                    )
                  }
                >
                  運営へ返品問題を報告しました。
                </TextLink>
                {row.report.detail ? ` ${row.report.detail}` : null}
              </>
            );
          }

          return row.message.content || "-";
        },
        minWidth: "280px",
      },
      {
        key: "images",
        header: "画像",
        render: (row) =>
          row.type === "returnReport"
            ? "-"
            : row.message.images.length,
        sortValue: (row) =>
          row.type === "returnReport"
            ? 0
            : row.message.images.length,
        nowrap: true,
      },
      {
        key: "reportCount",
        header: "通報",
        render: (row) => {
          if (row.type === "returnReport") {
            return "-";
          }

          const message = row.message;

          return message.reportCount > 0 && message.reportCaseId ? (
            <TextLink
              tone="accent"
              onClick={() =>
                navigate(
                  `/reports/${encodeURIComponent(message.reportCaseId!)}`,
                )
              }
            >
              {message.reportCount.toLocaleString("ja-JP")}
            </TextLink>
          ) : (
            message.reportCount.toLocaleString("ja-JP")
          );
        },
        sortValue: (row) =>
          row.type === "returnReport"
            ? 0
            : row.message.reportCount,
        nowrap: true,
      },
      {
        key: "createdAt",
        header: "送信日時",
        render: (row) => formatDateTime(row.createdAt),
        sortValue: (row) => getCreatedAtTime(row.createdAt),
        nowrap: true,
      },
    ],
    [navigate, participantNames],
  );

  if (loading && !messages) {
    return <p>取引メッセージを取得しています...</p>;
  }

  if (error && !messages) {
    return (
      <div role="alert">
        <p>取引メッセージを取得できませんでした。</p>
        <p>{error}</p>
        <button type="button" onClick={() => void reload()}>
          再読み込み
        </button>
      </div>
    );
  }

  return (
    <Table
      columns={columns}
      rows={rows}
      getRowKey={(row) => row.id}
      emptyMessage="メッセージはありません。"
      filteredEmptyMessage="条件に一致するメッセージはありません。"
    />
  );
}