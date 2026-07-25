// frontend/console/tokenBlueprintReview/src/presentation/component/review_aggregate_card.tsx

import "../../style/tokenBlueprintReview.css";

type ReviewAggregateCardProps = {
  likeCount: number;
  dislikeCount: number;
  reviewCount: number;
};

export default function ReviewAggregateCard({
  likeCount,
  dislikeCount,
  reviewCount,
}: ReviewAggregateCardProps) {
  return (
    <div className="bg-white border border-slate-200 rounded-xl p-4 shadow-sm tbrd-aggregate-card">
      <div className="flex items-center gap-3">
        <Metric label="高評価" value={likeCount} />
        <Metric label="低評価" value={dislikeCount} />
        <Metric label="レビュー" value={reviewCount} />
      </div>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  const v = Number.isFinite(value) ? value : 0;
  return (
    <div className="flex items-center gap-2 tbrd-aggregate-metric">
      <span className="text-sm text-slate-600">{label}</span>
      <span className="text-sm font-semibold text-slate-900">{v}</span>
    </div>
  );
}