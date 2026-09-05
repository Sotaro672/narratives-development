// frontend/admin/shell/src/features/contact/presentation/model/contactSourcePresentation.ts

export const CONTACT_SOURCE_LABELS = {
  "web-amol": "外部からの問い合わせ",
  console: "契約企業からの問い合わせ",
  mall: "購入者からの問い合わせ",
} as const;

export type KnownContactSource = keyof typeof CONTACT_SOURCE_LABELS;

export function getContactSourceLabel(source: string): string {
  const normalizedSource = source.trim();

  if (!normalizedSource) {
    return "問い合わせ";
  }

  return CONTACT_SOURCE_LABELS[
    normalizedSource as KnownContactSource
  ] ?? normalizedSource;
}