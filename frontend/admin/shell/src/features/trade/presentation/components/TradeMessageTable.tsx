// frontend/admin/shell/src/features/trade/presentation/components/TradeMessageTable.tsx

import { useMemo } from "react";
import { useNavigate } from "react-router-dom";

import type { TradeMessage } from "../../../../shared/type/trade";
import Table, { type TableColumn } from "../../../../shared/ui/Table/Table";
import TextLink from "../../../../shared/ui/TextLink/TextLink";
import { formatDateTime } from "../../../../shared/util/dateFormat";
import { useTradeMessages } from "../hooks/useTradeMessages";

type TradeMessageTableProps = {
  tradeId: string;
};

const SENDER_SIDE_LABELS: Record<TradeMessage["senderSide"], string> = {
  buyer: "購入者",
  seller: "出品者",
  system: "システム",
};

export default function TradeMessageTable({
  tradeId,
}: TradeMessageTableProps) {
  const navigate = useNavigate();
  const { messages, loading, error, reload } = useTradeMessages(tradeId);

  const columns = useMemo<TableColumn<TradeMessage>[]>(
    () => [
      {
        key: "senderName",
        header: "送信者",
        render: (message) => {
          const name = message.senderName || message.senderId || "-";
          const sideLabel = SENDER_SIDE_LABELS[message.senderSide];
          return `${name}（${sideLabel}）`;
        },
        sortValue: (message) =>
          message.senderName || message.senderId || "",
        nowrap: true,
      },
      {
        key: "content",
        header: "メッセージ",
        render: (message) => message.content || "-",
        minWidth: "280px",
      },
      {
        key: "images",
        header: "画像",
        render: (message) => message.images.length,
        sortValue: (message) => message.images.length,
        nowrap: true,
      },
      {
        key: "reportCount",
        header: "通報",
        render: (message) =>
          message.reportCount > 0 && message.reportCaseId ? (
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
          ),
        sortValue: (message) => message.reportCount,
        nowrap: true,
      },
      {
        key: "createdAt",
        header: "送信日時",
        render: (message) => formatDateTime(message.createdAt),
        sortValue: (message) => new Date(message.createdAt).getTime(),
        nowrap: true,
      },
    ],
    [navigate],
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
      rows={messages?.items ?? []}
      getRowKey={(message) => message.id}
      emptyMessage="メッセージはありません。"
      filteredEmptyMessage="条件に一致するメッセージはありません。"
    />
  );
}