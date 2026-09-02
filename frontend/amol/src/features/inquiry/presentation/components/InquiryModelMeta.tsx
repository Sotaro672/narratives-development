// frontend/amol/src/features/inquiry/presentation/components/InquiryModelMeta.tsx

import ChatMetaSection, { type ChatMetaItem } from "../../../shared/presentation/components/ChatMetaSection";
import type { InquiryDetailModelMeta } from "../../../shared/types/inquiryTypes";

type InquiryModelMetaProps = {
  modelMeta: InquiryDetailModelMeta;
};

export default function InquiryModelMeta({
  modelMeta,
}: InquiryModelMetaProps) {
  const metaItems = getModelMetaItems(modelMeta);

  return (
    <ChatMetaSection
      title="対象商品"
      items={metaItems}
    />
  );
}

function getModelMetaItems(
  modelMeta: InquiryDetailModelMeta,
): ChatMetaItem[] {
  const items: ChatMetaItem[] = [];

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