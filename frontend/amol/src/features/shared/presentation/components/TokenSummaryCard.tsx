// frontend\amol\src\features\shared\presentation\components\TokenSummaryCard.tsx

import EntitySummaryCard from "./EntitySummaryCard";

export type TokenSummaryCardProps = {
  brandName?: string | null;
  tokenName?: string | null;
  tokenIcon?: string | null;
  symbol?: string | null;
  description?: string | null;
};

export default function TokenSummaryCard({
  brandName,
  tokenName,
  tokenIcon,
  symbol,
  description,
}: TokenSummaryCardProps) {
  const safeBrandName = brandName?.trim() || "ブランド名未設定";
  const safeTokenName = tokenName?.trim() || "トークン名未設定";
  const safeTokenIcon = tokenIcon?.trim() || "";
  const safeSymbol = symbol?.trim() || "";
  const safeDescription = description?.trim() || "";

  if (!safeTokenIcon && !tokenName?.trim()) {
    return null;
  }

  return (
    <EntitySummaryCard
      icon={safeTokenIcon}
      iconAlt={tokenName?.trim() ? `${tokenName.trim()}のトークンアイコン` : "トークンアイコン"}
      iconFallback="◎"
      label={safeBrandName}
      name={safeTokenName}
      secondaryText={safeSymbol}
      description={safeDescription}
    />
  );
}