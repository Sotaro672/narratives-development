// frontend/amol/src/features/scan-result/components/ScanResultTokenSection.tsx
import MediaIcon from "../../../../components/ui/MediaIcon";
import SectionCard from "../../../../components/ui/SectionCard";
import Tab from "../../../../components/ui/Tab";
import TextState from "../../../../components/ui/TextState";

type ScanResultTokenSectionProps = {
  tokenName: string;
  tokenIconUrl: string;
  tokenBrandName: string;
  tokenCompanyName: string;
  tokenDescription: string;
  mintAddress: string;
  canOpenTokenContents: boolean;
  onOpenTokenContents: (mintAddress: string) => void | Promise<void>;
};

export default function ScanResultTokenSection(
  props: ScanResultTokenSectionProps
) {
  const {
    tokenName,
    tokenIconUrl,
    tokenBrandName,
    tokenCompanyName,
    tokenDescription,
    mintAddress,
    canOpenTokenContents,
    onOpenTokenContents,
  } = props;

  const handleOpenTokenContents = () => {
    if (!canOpenTokenContents) {
      return;
    }

    void onOpenTokenContents(mintAddress);
  };

  return (
    <SectionCard>
      <div className="scan-result-token-header">
        <h2>トークン情報</h2>
      </div>

      <div className="scan-result-token-body">
        <MediaIcon
          src={tokenIconUrl}
          fallback="icon"
          size="lg"
          shape="rounded"
          className="scan-result-token-icon"
        />

        <div>
          {canOpenTokenContents ? (
            <Tab
              className="scan-result-token-name"
              onClick={handleOpenTokenContents}
            >
              {tokenName}
            </Tab>
          ) : (
            <h3>{tokenName || "-"}</h3>
          )}

          {tokenBrandName ? (
            <TextState>ブランド: {tokenBrandName}</TextState>
          ) : null}

          {tokenCompanyName ? (
            <TextState>会社: {tokenCompanyName}</TextState>
          ) : null}

          {tokenDescription ? (
            <p className="scan-result-description">{tokenDescription}</p>
          ) : null}
        </div>
      </div>
    </SectionCard>
  );
}