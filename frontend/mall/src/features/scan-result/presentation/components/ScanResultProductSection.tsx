// frontend/mall/src/features/scan-result/presentation/components/ScanResultProductSection.tsx

import { useNavigate } from "react-router-dom";

import Badge from "../../../../components/ui/Badge";
import InfoList, { InfoRow } from "../../../../components/ui/InfoList";
import SectionHeader from "../../../../components/ui/SectionHeader";
import Tab from "../../../../components/ui/Tab";
import TextLink from "../../../../components/ui/textLink";
import TextState from "../../../../components/ui/TextState";

import type { ScanAlcoholInfo } from "../../application/scanAlcoholInfoFactory";
import type { ScanDisplayRowViewModel } from "../../application/scanPageViewModelFactory";

type ScanResultProductSectionProps = {
  title: string;
  owned: boolean | null;
  ownedError: string;
  ownerLabel: string;
  brandId: string;
  brandName: string;
  hasBrandInfo: boolean;
  productBlueprintRows: ScanDisplayRowViewModel[];
  qualityAssuranceTabs: string[];
  modelNumber: string;
  size: string;
  color: string;
  swatch: string;
  measurementEntries: ScanDisplayRowViewModel[];
  alcoholInfo?: ScanAlcoholInfo | null;
};

function hasAlcoholDisplayInfo(alcoholInfo?: ScanAlcoholInfo | null): boolean {
  if (!alcoholInfo?.isAlcohol) {
    return false;
  }

  return Boolean(
    alcoholInfo.volumeLabel ||
      alcoholInfo.vintage ||
      alcoholInfo.region ||
      alcoholInfo.material ||
      alcoholInfo.alcoholContent,
  );
}

export default function ScanResultProductSection(
  props: ScanResultProductSectionProps,
) {
  const navigate = useNavigate();

  const {
    title,
    owned,
    ownedError,
    ownerLabel,
    brandId,
    brandName,
    hasBrandInfo,
    productBlueprintRows,
    qualityAssuranceTabs,
    modelNumber,
    size,
    color,
    swatch,
    measurementEntries,
    alcoholInfo,
  } = props;

  const isAlcohol = alcoholInfo?.isAlcohol === true;
  const shouldShowAlcoholInfo = hasAlcoholDisplayInfo(alcoholInfo);
  const canOpenBrand = Boolean(brandId.trim());

  const handleOpenBrand = () => {
    const normalizedBrandId = brandId.trim();

    if (!normalizedBrandId) {
      return;
    }

    navigate(`/brands/${encodeURIComponent(normalizedBrandId)}`);
  };

  return (
    <section>
      <SectionHeader
        title={title}
        className="ui-section-header--title-sm"
        right={
          owned === true ? (
            <Badge variant="success" size="md">
              Owned
            </Badge>
          ) : null
        }
      />

      <TextState>所有者: {ownerLabel || "-"}</TextState>

      {owned === null && ownedError ? (
        <TextState>
          保有判定に失敗しました: {ownedError}
        </TextState>
      ) : null}

      {hasBrandInfo ||
      productBlueprintRows.length > 0 ||
      qualityAssuranceTabs.length > 0 ? (
        <InfoList>
          {hasBrandInfo ? (
            <InfoRow label="ブランド">
              <TextLink
                onClick={handleOpenBrand}
                disabled={!canOpenBrand}
                aria-label={`${brandName || "ブランド"}のブランドページへ移動`}
              >
                {brandName || "-"}
              </TextLink>
            </InfoRow>
          ) : null}

          {productBlueprintRows.map((row) => (
            <InfoRow label={row.label} key={row.label}>
              {row.value || "-"}
            </InfoRow>
          ))}

          {qualityAssuranceTabs.length > 0 ? (
            <InfoRow label="品質保証">
              <span className="scan-result-quality-tabs">
                {qualityAssuranceTabs.map((quality, index) => (
                  <Tab
                    key={`${quality}-${index}`}
                    className="scan-result-quality-tab"
                    disabled
                  >
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

        {isAlcohol ? (
          <>
            <InfoRow label="容量">{alcoholInfo?.volumeLabel || "-"}</InfoRow>
            <InfoRow label="ヴィンテージ">
              {alcoholInfo?.vintage || "-"}
            </InfoRow>
            <InfoRow label="地域・産地">{alcoholInfo?.region || "-"}</InfoRow>
            <InfoRow label="原材料">{alcoholInfo?.material || "-"}</InfoRow>
            <InfoRow label="アルコール度数">
              {alcoholInfo?.alcoholContent
                ? `${alcoholInfo.alcoholContent}%`
                : "-"}
            </InfoRow>
          </>
        ) : (
          <>
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
          </>
        )}
      </InfoList>

      {!isAlcohol && measurementEntries.length > 0 ? (
        <div className="scan-result-measurements">
          <SectionHeader
            title="採寸"
            titleAs="h2"
            className="ui-section-header--title-sm"
          />

          <InfoList>
            {measurementEntries.map((row, index) => (
              <InfoRow label={row.label} key={`${row.label}-${index}`}>
                {row.value || "-"}
              </InfoRow>
            ))}
          </InfoList>
        </div>
      ) : null}

      {isAlcohol && !shouldShowAlcoholInfo ? (
        <TextState>酒類情報を取得できませんでした。</TextState>
      ) : null}
    </section>
  );
}