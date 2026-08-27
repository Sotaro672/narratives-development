// frontend/amol/src/features/inquiry/presentation/components/InquiryModelMeta.tsx

import type { InquiryDetailModelMeta } from "../../../shared/types/inquiryTypes";

type InquiryModelMetaProps = {
  modelMeta: InquiryDetailModelMeta;
};

type ModelMetaItem = {
  label: string;
  value: string;
};

export default function InquiryModelMeta({
  modelMeta,
}: InquiryModelMetaProps) {
  const metaItems = getModelMetaItems(modelMeta);

  if (metaItems.length === 0) {
    return null;
  }

  return (
    <section className="chat-detail-page__product-meta">
      <h3 className="chat-detail-page__product-meta-title">
        対象商品
      </h3>

      <dl className="chat-detail-page__product-meta-list">
        {metaItems.map((meta) => (
          <div
            key={meta.label}
            className="chat-detail-page__product-meta-row"
          >
            <dt>{meta.label}</dt>
            <dd>{meta.value}</dd>
          </div>
        ))}
      </dl>
    </section>
  );
}

function getModelMetaItems(
  modelMeta: InquiryDetailModelMeta,
): ModelMetaItem[] {
  const items: ModelMetaItem[] = [];

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