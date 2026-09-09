// frontend/admin/shell/src/shared/util/modelMetaFormat.ts

export type ModelMeta = {
  kind: string;
  modelNumber: string;
  size?: string;
  color?: string;
  volumeValue?: number;
  volumeUnit?: string;
};

export function formatModelMeta(model: ModelMeta): string {
  const values: string[] = [];

  if (model.modelNumber) {
    values.push(model.modelNumber);
  }

  if (model.kind === "apparel") {
    if (model.size) {
      values.push(model.size);
    }
    if (model.color) {
      values.push(model.color);
    }
  }

  if (model.kind === "alcohol" && model.volumeValue != null) {
    values.push(`${model.volumeValue}${model.volumeUnit || ""}`);
  }

  return values.length > 0 ? values.join(" / ") : "モデル情報なし";
}