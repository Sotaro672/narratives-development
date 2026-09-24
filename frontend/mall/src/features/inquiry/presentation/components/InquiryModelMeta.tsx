// frontend/mall/src/features/inquiry/presentation/components/InquiryModelMeta.tsx

import InfoList, {
  type InfoListRow,
} from "../../../../components/ui/InfoList";
import SectionHeader from "../../../../components/ui/SectionHeader";
import type { InquiryDetailModelMeta } from "../../../shared/types/inquiryTypes";

type InquiryModelMetaProps = {
  modelMeta: InquiryDetailModelMeta;
};

export default function InquiryModelMeta({
  modelMeta,
}: InquiryModelMetaProps) {
  const metaItems = getModelMetaItems(modelMeta);

  if (metaItems.length === 0) {
    return null;
  }

  return (
    <section>
      <SectionHeader
        title="対象商品"
        titleAs="h3"
        className="ui-section-header--title-sm"
      />

      <InfoList rows={metaItems} />
    </section>
  );
}

function getModelMetaItems(
  modelMeta: InquiryDetailModelMeta,
): InfoListRow[] {
  const items: InfoListRow[] = [];

  if (modelMeta.modelNumber) {
    items.push({
      label: "モデル番号",
      value: modelMeta.modelNumber,
    });
  }

  if (modelMeta.size) {
    items.push({
      label: "サイズ",
      value: modelMeta.size,
    });
  }

  if (modelMeta.color?.name) {
    items.push({
      label: "カラー",
      value: modelMeta.color.name,
    });
  }

  if (
    modelMeta.volumeValue !== undefined &&
    modelMeta.volumeValue !== null
  ) {
    items.push({
      label: "容量",
      value: `${modelMeta.volumeValue}${modelMeta.volumeUnit ?? ""}`,
    });
  }

  return items;
}