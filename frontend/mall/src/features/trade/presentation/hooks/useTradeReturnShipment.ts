// frontend/mall/src/features/trade/presentation/hooks/useTradeReturnShipment.ts

import { useCallback, useEffect, useState } from "react";

import type {
  TradeDetail,
  TradeReturnShipment,
} from "../../../shared/types/trade";
import { createTradeReturnShipment } from "../../infrastructure/tradeApi";
import { getErrorMessage } from "../util/tradeChatDetail";

type UseTradeReturnShipmentInput = {
  tradeId: string;
  trade: TradeDetail | null;
  reload: () => Promise<void>;
  blocked: boolean;
};

function canPrepareReturnShipment(
  tradeId: string,
  trade: TradeDetail | null,
  blocked: boolean,
): boolean {
  if (
    blocked ||
    tradeId.trim() === "" ||
    trade === null ||
    trade.viewerSide !== "buyer" ||
    trade.status !== "active" ||
    trade.isCancelled ||
    !trade.isDispatched ||
    trade.transferred ||
    trade.returnStatus !== "agreed"
  ) {
    return false;
  }

  const proposal = trade.returnProposal;

  if (
    !proposal ||
    proposal.id.trim() === "" ||
    proposal.agreement !== "agree" ||
    proposal.rejectedAt ||
    proposal.returnRequirement !== "required"
  ) {
    return false;
  }

  return (
    proposal.refundAmount !== undefined &&
    Number.isInteger(proposal.refundAmount) &&
    proposal.refundAmount > 0
  );
}

function isReadyReturnShipment(
  shipment: TradeReturnShipment | null,
): shipment is TradeReturnShipment & {
  status: "ready_for_dropoff";
  qrCodePayload: string;
} {
  return (
    shipment !== null &&
    shipment.status === "ready_for_dropoff" &&
    shipment.dropOffMethod === "pudo" &&
    typeof shipment.qrCodePayload === "string" &&
    shipment.qrCodePayload.trim() !== ""
  );
}

export function useTradeReturnShipment({
  tradeId,
  trade,
  reload,
  blocked,
}: UseTradeReturnShipmentInput) {
  const [open, setOpen] = useState(false);
  const [shipment, setShipment] =
    useState<TradeReturnShipment | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const available = canPrepareReturnShipment(
    tradeId,
    trade,
    blocked,
  );

  const ready = isReadyReturnShipment(shipment);
  const qrCodePayload = ready ? shipment.qrCodePayload : "";

  useEffect(() => {
    setOpen(false);
    setShipment(null);
    setError("");
    setLoading(false);
  }, [tradeId]);

  const prepare = useCallback(
    async (): Promise<TradeReturnShipment | null> => {
      if (
        loading ||
        !canPrepareReturnShipment(
          tradeId,
          trade,
          blocked,
        )
      ) {
        return null;
      }

      setLoading(true);
      setError("");

      try {
        // POSTは冪等。
        // 未作成ならReturnShipmentを作成してmock PUDO QRを準備し、
        // pendingなら準備処理を再試行し、
        // ready_for_dropoffなら保存済みReturnShipmentを返す。
        const preparedShipment =
          await createTradeReturnShipment({
            tradeId,
          });

        setShipment(preparedShipment);

        try {
          // ReturnShipment準備時にsystem messageが追加されるため、
          // 最新の取引情報とメッセージ一覧を再取得する。
          await reload();
        } catch (reloadError) {
          setError(
            getErrorMessage(
              reloadError,
              "返品用QRは取得できましたが、取引情報の再取得に失敗しました。",
            ),
          );
        }

        return preparedShipment;
      } catch (caught) {
        setError(
          getErrorMessage(
            caught,
            "返品用QRを準備できませんでした。",
          ),
        );
        return null;
      } finally {
        setLoading(false);
      }
    },
    [
      blocked,
      loading,
      reload,
      trade,
      tradeId,
    ],
  );

  const openModal = useCallback(
    async (): Promise<void> => {
      if (
        loading ||
        !canPrepareReturnShipment(
          tradeId,
          trade,
          blocked,
        )
      ) {
        return;
      }

      setOpen(true);
      setError("");

      await prepare();
    },
    [
      blocked,
      loading,
      prepare,
      trade,
      tradeId,
    ],
  );

  const retryPreparation = useCallback(
    async (): Promise<TradeReturnShipment | null> => {
      if (!open) {
        return null;
      }

      return prepare();
    },
    [
      open,
      prepare,
    ],
  );

  const closeModal = useCallback(() => {
    if (loading) {
      return;
    }

    setOpen(false);
    setError("");
  }, [loading]);

  return {
    open,
    shipment,
    qrCodePayload,
    error,
    loading,
    available,
    ready,
    openModal,
    closeModal,
    retryPreparation,
  };
}

export default useTradeReturnShipment;