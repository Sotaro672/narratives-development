// frontend/amol/src/features/market/presentation/components/MarketSellerCard.tsx

import EntitySummaryCard from "../../../shared/presentation/components/EntitySummaryCard";

type MarketSellerCardProps = {
  avatarId: string;
  avatarName: string;
  avatarIcon: string;
  onOpen: () => void;
};

export default function MarketSellerCard({
  avatarId,
  avatarName,
  avatarIcon,
  onOpen,
}: MarketSellerCardProps) {
  if (!avatarId && !avatarName && !avatarIcon) {
    return null;
  }

  return (
    <EntitySummaryCard
      icon={avatarIcon}
      iconAlt={avatarName || "出品者アイコン"}
      iconFallback="◎"
      label="出品者"
      name={avatarName || avatarId || "アバター名未設定"}
      onClick={onOpen}
      disabled={!avatarId}
    />
  );
}