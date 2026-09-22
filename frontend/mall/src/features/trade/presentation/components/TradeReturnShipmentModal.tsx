// frontend/mall/src/features/trade/presentation/components/TradeReturnShipmentModal.tsx

import { QRCodeSVG } from "qrcode.react";

import Alert from "../../../../components/ui/Alert";
import Button from "../../../../components/ui/Button";
import Modal, {
  ModalBody,
  ModalDescription,
  ModalFooter,
  ModalHeader,
  ModalTitle,
} from "../../../../components/ui/Modal";
import type { TradeReturnShipment } from "../../../shared/types/trade";

export type TradeReturnShipmentModalProps = {
  open: boolean;
  shipment: TradeReturnShipment | null;
  qrCodePayload: string;
  error?: string | null;
  loading: boolean;
  onCancel: () => void;
  onRetryPreparation?: () => void;
};

function isReadyShipment(
  shipment: TradeReturnShipment | null,
  qrCodePayload: string,
): shipment is TradeReturnShipment & {
  status: "ready_for_dropoff";
  dropOffMethod: "pudo";
} {
  return (
    shipment !== null &&
    shipment.status === "ready_for_dropoff" &&
    shipment.dropOffMethod === "pudo" &&
    qrCodePayload.trim() !== ""
  );
}

function formatReadyAt(value: string | undefined): string | null {
  if (!value) {
    return null;
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return null;
  }

  return new Intl.DateTimeFormat("ja-JP", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export default function TradeReturnShipmentModal({
  open,
  shipment,
  qrCodePayload,
  error,
  loading,
  onCancel,
  onRetryPreparation,
}: TradeReturnShipmentModalProps) {
  const ready = isReadyShipment(
    shipment,
    qrCodePayload,
  );
  const readyAt = formatReadyAt(shipment?.readyAt);
  const handleClose = loading ? undefined : onCancel;

  const handleRetryPreparation = (): void => {
    if (loading || !onRetryPreparation) {
      return;
    }

    onRetryPreparation();
  };

  return (
    <Modal
      open={open}
      onClose={handleClose}
      size="md"
      mobilePosition="bottom"
      closeOnBackdrop={!loading}
      closeOnEscape={!loading}
      ariaLabelledBy="trade-return-shipment-modal-title"
      ariaDescribedBy="trade-return-shipment-modal-description"
      ariaBusy={loading}
    >
      <ModalHeader onClose={handleClose}>
        <ModalTitle id="trade-return-shipment-modal-title">
          返品用QR
        </ModalTitle>
      </ModalHeader>

      <ModalBody>
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            gap: 24,
          }}
        >
          <ModalDescription id="trade-return-shipment-modal-description">
            PUDOを利用した匿名返品用のQRを確認できます。
          </ModalDescription>

          <Alert variant="warning">
            現在表示しているQRは開発用のモックデータです。実際のPUDO端末では利用できません。
          </Alert>

          {loading && !ready ? (
            <Alert variant="info">
              返品用QRを準備しています。
            </Alert>
          ) : null}

          {ready ? (
            <>
              <section
                aria-label="返品用QRコード"
                style={{
                  display: "flex",
                  flexDirection: "column",
                  alignItems: "center",
                  gap: 16,
                }}
              >
                <div
                  style={{
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    width: "100%",
                    maxWidth: 280,
                    aspectRatio: "1",
                    padding: 20,
                    boxSizing: "border-box",
                    border: "1px solid #e5e7eb",
                    borderRadius: 16,
                    background: "#ffffff",
                  }}
                >
                  <QRCodeSVG
                    value={qrCodePayload}
                    size={240}
                    level="M"
                    style={{
                      display: "block",
                      width: "100%",
                      height: "100%",
                    }}
                  />
                </div>

                <div
                  style={{
                    display: "flex",
                    width: "100%",
                    flexDirection: "column",
                    gap: 8,
                  }}
                >
                  <div
                    style={{
                      display: "flex",
                      justifyContent: "space-between",
                      gap: 16,
                    }}
                  >
                    <span>返送方法</span>
                    <strong>PUDO</strong>
                  </div>

                  {readyAt ? (
                    <div
                      style={{
                        display: "flex",
                        justifyContent: "space-between",
                        gap: 16,
                      }}
                    >
                      <span>QR準備日時</span>
                      <strong>{readyAt}</strong>
                    </div>
                  ) : null}
                </div>
              </section>

              <Alert variant="info">
                QRの表示は返品商品の発送完了を意味しません。現在のAMOLは運送業者から配送状況を取得しません。
              </Alert>
            </>
          ) : null}

          {!loading && shipment?.status === "pending" ? (
            <Alert variant="info">
              返品用QRの準備が完了していません。必要に応じて準備を再試行してください。
            </Alert>
          ) : null}

          {error ? (
            <Alert variant="error">
              {error}
            </Alert>
          ) : null}
        </div>
      </ModalBody>

      <ModalFooter>
        <div
          style={{
            display: "flex",
            width: "100%",
            flexDirection: "column",
            gap: 8,
          }}
        >
          {!ready && onRetryPreparation ? (
            <Button
              type="button"
              variant="primary"
              size="md"
              fullWidth
              disabled={loading}
              onClick={handleRetryPreparation}
            >
              {loading
                ? "準備中..."
                : "QR準備を再試行する"}
            </Button>
          ) : null}

          <Button
            type="button"
            variant={ready ? "primary" : "secondary"}
            size="md"
            fullWidth
            disabled={loading}
            onClick={onCancel}
          >
            閉じる
          </Button>
        </div>
      </ModalFooter>
    </Modal>
  );
}