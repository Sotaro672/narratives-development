// frontend/amol/src/features/shared/presentation/components/AvatarSummaryCard.tsx

import EntitySummaryCard from "./EntitySummaryCard";

export type AvatarSummaryCardProps = {
  avatarId?: string | null;
  avatarName?: string | null;
  avatarIcon?: string | null;
  onClick?: () => void;
};

export default function AvatarSummaryCard({
  avatarId,
  avatarName,
  avatarIcon,
  onClick,
}: AvatarSummaryCardProps) {
  const safeAvatarId = avatarId?.trim() || "";
  const safeAvatarName = avatarName?.trim() || "";
  const safeAvatarIcon = avatarIcon?.trim() || "";

  if (!safeAvatarId && !safeAvatarName && !safeAvatarIcon) {
    return null;
  }

  return (
    <EntitySummaryCard
      icon={safeAvatarIcon}
      iconAlt={safeAvatarName || "出品者アイコン"}
      iconFallback="◎"
      label="出品者"
      name={safeAvatarName || safeAvatarId || "アバター名未設定"}
      onClick={onClick}
      disabled={Boolean(onClick) && !safeAvatarId}
    />
  );
}