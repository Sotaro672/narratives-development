// frontend/amol/src/features/scan-result/components/ScanResultProductSection.tsx
import InfoList, { InfoRow } from "../../../components/ui/InfoList";
import MediaIcon from "../../../components/ui/MediaIcon";
import SectionCard from "../../../components/ui/SectionCard";
import SectionHeader from "../../../components/ui/SectionHeader";
import Tab from "../../../components/ui/Tab";
import TextState from "../../../components/ui/TextState";
import type { MallOwnerInfo } from "../types";
import { ownerLabel, withCm } from "../utils/format";
import type { InfoRow as ProductInfoRow } from "../utils/productBlueprint";

type ScanResultProductSectionProps = {
  title: string;
  owned: boolean | null;
  ownedError: string;
  owner: MallOwnerInfo | null;
  brandName: string;
  brandIcon: string;
  hasBrandInfo: boolean;
  productBlueprintRows: ProductInfoRow[];
  qualityAssuranceTabs: string[];
  modelNumber: string;
  size: string;
  color: string;
  swatch: string;
  measurementEntries: [string, unknown][];
};

export default function ScanResultProductSection(
  props: ScanResultProductSectionProps
) {
  const {
    title,
    owned,
    ownedError,
    owner,
    brandName,
    brandIcon,
    hasBrandInfo,
    productBlueprintRows,
    qualityAssuranceTabs,
    modelNumber,
    size,
    color,
    swatch,
    measurementEntries,
  } = props;

  return (
    <SectionCard>
      <SectionHeader
        eyebrow="商品情報"
        title={title}
        right={
          owned === true ? (
            <span className="scan-result-owned-badge">Owned</span>
          ) : null
        }
      />

      <TextState>所有者: {ownerLabel(owner)}</TextState>

      {owned === null && ownedError ? (
        <TextState>保有判定に失敗しました: {ownedError}</TextState>
      ) : null}

      {hasBrandInfo ||
      productBlueprintRows.length > 0 ||
      qualityAssuranceTabs.length > 0 ? (
        <InfoList>
          {hasBrandInfo ? (
            <InfoRow label="ブランド">
              <span className="scan-result-brand-value">
                <MediaIcon
                  src={brandIcon}
                  fallback=""
                  size="xs"
                  shape="circle"
                  className="scan-result-brand-icon"
                />
                <span>{brandName || "-"}</span>
              </span>
            </InfoRow>
          ) : null}

          {productBlueprintRows.map((row) => (
            <InfoRow label={row.label} key={row.label}>
              {row.value}
            </InfoRow>
          ))}

          {qualityAssuranceTabs.length > 0 ? (
            <InfoRow label="品質保証">
              <span className="scan-result-quality-tabs">
                {qualityAssuranceTabs.map((quality) => (
                  <Tab key={quality} className="scan-result-quality-tab" disabled>
                    {quality}
                  </Tab>
                ))}
              </span>
            </InfoRow>
          ) : null}
        </InfoList>
      ) : null}

      <InfoList>
        <InfoRow label="型番">{modelNumber || "-"}</InfoRow>

        <InfoRow label="サイズ">{size || "-"}</InfoRow>

        <InfoRow label="色名">
          <span className="scan-result-color-value">
            {color || "-"}
            <span
              className="scan-result-swatch"
              style={{ backgroundColor: swatch }}
            />
          </span>
        </InfoRow>
      </InfoList>

      {measurementEntries.length > 0 ? (
        <div className="scan-result-measurements">
          <h2>採寸</h2>

          <InfoList>
            {measurementEntries.map(([key, value]) => (
              <InfoRow label={key} key={key}>
                {withCm(value)}
              </InfoRow>
            ))}
          </InfoList>
        </div>
      ) : null}
    </SectionCard>
  );
}