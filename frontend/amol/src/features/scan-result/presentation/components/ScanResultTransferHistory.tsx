//frontend\amol\src\features\scan-result\presentation\components\ScanResultTransferHistory.tsx
import { useNavigate } from "react-router-dom";

import MediaIcon from "../../../../components/ui/MediaIcon";
import SectionCard from "../../../../components/ui/SectionCard";
import TextState from "../../../../components/ui/TextState";
import { formatDateTime } from "../../../../components/utils/date";
import type { MallPreviewTransferInfo } from "../../types";
import {
  transferBrandId,
  transferDisplayName,
  transferIconUrl,
} from "../../utils/format";

type ScanResultTransferHistoryProps = {
  transfers: MallPreviewTransferInfo[];
};

function TransferPartyBlock(props: {
  title: string;
  name: string;
  iconUrl: string;
  brandId: string;
  onOpenBrand: (brandId: string) => void;
}) {
  const clickable = props.brandId.trim().length > 0;

  const content = (
    <>
      <MediaIcon
        src={props.iconUrl}
        fallback="img"
        size="sm"
        shape="circle"
        className="scan-result-party__icon"
      />

      <div className="scan-result-party__body">
        <div className="scan-result-party__title">{props.title}</div>
        <div className="scan-result-party__name">{props.name}</div>
      </div>
    </>
  );

  if (!clickable) {
    return <div className="scan-result-party">{content}</div>;
  }

  return (
    <button
      type="button"
      className="scan-result-party scan-result-party--button"
      onClick={() => props.onOpenBrand(props.brandId)}
    >
      {content}
    </button>
  );
}

export default function ScanResultTransferHistory(
  props: ScanResultTransferHistoryProps
) {
  const navigate = useNavigate();

  const transfers = [...props.transfers].sort((a, b) => {
    const at = a.transferredAt ? new Date(a.transferredAt).getTime() : 0;
    const bt = b.transferredAt ? new Date(b.transferredAt).getTime() : 0;
    return at - bt;
  });

  const handleOpenBrand = (brandId: string) => {
    const id = brandId.trim();

    if (!id) {
      return;
    }

    navigate(`/brands/${encodeURIComponent(id)}`);
  };

  return (
    <SectionCard>
      <h2>移譲履歴</h2>

      {transfers.length === 0 ? (
        <TextState>移譲履歴はありません</TextState>
      ) : (
        <div className="scan-result-transfer-list">
          {transfers.map((transfer, index) => {
            const fromName = transferDisplayName(transfer, "from");
            const toName = transferDisplayName(transfer, "to");
            const fromIcon = transferIconUrl(transfer, "from");
            const toIcon = transferIconUrl(transfer, "to");
            const fromBrandId = transferBrandId(transfer, "from");
            const toBrandId = transferBrandId(transfer, "to");

            return (
              <article
                className="scan-result-transfer"
                key={`${transfer.fromWalletAddress}-${transfer.toWalletAddress}-${
                  transfer.transferredAt || index
                }`}
              >
                <div className="scan-result-transfer__date">
                  日時: {formatDateTime(transfer.transferredAt)}
                </div>

                <TransferPartyBlock
                  title="移譲元"
                  name={fromName}
                  iconUrl={fromIcon}
                  brandId={fromBrandId}
                  onOpenBrand={handleOpenBrand}
                />

                <div className="scan-result-transfer__arrow">↓</div>

                <TransferPartyBlock
                  title="移譲先"
                  name={toName}
                  iconUrl={toIcon}
                  brandId={toBrandId}
                  onOpenBrand={handleOpenBrand}
                />
              </article>
            );
          })}
        </div>
      )}
    </SectionCard>
  );
}