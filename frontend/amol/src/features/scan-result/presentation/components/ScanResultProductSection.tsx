// frontend/amol/src/features/scan-result/components/ScanResultProductSection.tsx
import { useNavigate } from "react-router-dom";

import InfoList, { InfoRow } from "../../../../components/ui/InfoList";
import SectionCard from "../../../../components/ui/SectionCard";
import SectionHeader from "../../../../components/ui/SectionHeader";
import Tab from "../../../../components/ui/Tab";
import TextState from "../../../../components/ui/TextState";
import type { ScanAlcoholInfo } from "../../application/scanAlcoholInfoFactory";
import type { MallOwnerInfo } from "../../types";
import { ownerLabel, withCm } from "../../utils/format";
import type { InfoRow as ProductInfoRow } from "../../utils/productBlueprint";

type ScanResultProductSectionProps = {
  title: string;
  owned: boolean | null;
  ownedError: string;
  owner: MallOwnerInfo | null;
  brandId: string;
  brandName: string;
  hasBrandInfo: boolean;
  productBlueprintRows: ProductInfoRow[];
  qualityAssuranceTabs: string[];
  modelNumber: string;
  size: string;
  color: string;
  swatch: string;
  measurementEntries: [string, unknown][];

  /**
   * alcohol 用の表示情報。
   * productBlueprintPatch.categoryFields を正として application 層で生成する。
   */
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
    owner,
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
  const canOpenBrand = Boolean(brandId);

  const handleOpenBrand = () => {
    if (!brandId) {
      return;
    }

    navigate(`/brands/${encodeURIComponent(brandId)}`);
  };

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
              <button
                type="button"
                className="scan-result-brand-value scan-result-brand-value--button"
                onClick={handleOpenBrand}
                disabled={!canOpenBrand}
              >
                <span>{brandName || "-"}</span>
              </button>
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
                  <Tab
                    key={quality}
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

            <InfoRow label="地域・産地">
              {alcoholInfo?.region || "-"}
            </InfoRow>

            <InfoRow label="原材料">
              {alcoholInfo?.material || "-"}
            </InfoRow>

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

      {isAlcohol && !shouldShowAlcoholInfo ? (
        <TextState>酒類情報を取得できませんでした。</TextState>
      ) : null}
    </SectionCard>
  );
}