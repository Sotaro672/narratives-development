// frontend/console/productBlueprintReview/src/presentation/pages/productBlueprintReviewDetail.tsx

import { useMemo, useState } from "react";
import { useParams, useNavigate, useLocation } from "react-router-dom";

import PageStyle from "../../../../shell/src/layout/PageStyle/PageStyle";
import AdminCard from "../../../../admin/src/presentation/components/AdminCard";
import LogCard from "../../../../log/presentation/LogCard";

import Pagination from "../../../../shell/src/shared/ui/pagination";
import RefreshButton from "../../../../shell/src/shared/ui/refresh";
import { Button } from "../../../../shell/src/shared/ui/button";

import { ratingToStars, statusLabelJa } from "../../../../shell/src/shared/format/review";

import { useProductBlueprintReviewDetail } from "../hook/useProductBlueprintReviewDetail";

import "../../style/productBlueprintReview.css";

type DetailNavState = {
  ProductBlueprintID?: string;
  ProductName?: string;
  AssigneeName?: string;
};

type SortKey = "Rating" | "ReviewedAt" | null;
type SortDir = "asc" | "desc";

export default function ProductBlueprintReviewDetail() {
  const Params = useParams();
  const Navigate = useNavigate();
  const Location = useLocation();

  const ProductBlueprintReviewId = String(Params.productBlueprintReviewId ?? "");
  const State = (Location.state ?? {}) as DetailNavState;

  const HeaderProductName = String(State.ProductName ?? "");
  const HeaderAssigneeName = String(State.AssigneeName ?? "");

  const {
    Status,
    Page,
    Items,
    TotalPages,
    IsLoading,
    ErrorMessage,
    OnBack: HookOnBack,
    OnReload,
    SetStatus,
    SetPage,
  } = useProductBlueprintReviewDetail();

  const Title =
    HeaderProductName ||
    (ProductBlueprintReviewId ? `Review: ${ProductBlueprintReviewId}` : "Review Detail");

  const OnBack = () => {
    if (HookOnBack) HookOnBack();
    else Navigate("..");
  };

  const Noop = () => {};

  // ✅ Sort state (client-side: current Items only)
  const [SortBy, setSortBy] = useState<SortKey>(null);
  const [SortDir, setSortDir] = useState<SortDir>("desc");

  const toggleSort = (key: Exclude<SortKey, null>) => {
    if (SortBy !== key) {
      setSortBy(key);
      setSortDir("desc");
      return;
    }
    setSortDir((d) => (d === "desc" ? "asc" : "desc"));
  };

  const sortLabel = (key: Exclude<SortKey, null>) => {
    if (SortBy !== key) return "↕";
    return SortDir === "desc" ? "↓" : "↑";
  };

  const SortedItems = useMemo(() => {
    const arr = [...(Items ?? [])];
    if (!SortBy) return arr;

    const dir = SortDir === "asc" ? 1 : -1;

    if (SortBy === "Rating") {
      arr.sort((a, b) => {
        const av = Number(a?.Rating ?? 0);
        const bv = Number(b?.Rating ?? 0);
        return (av - bv) * dir;
      });
      return arr;
    }

    // ReviewedAt
    arr.sort((a, b) => {
      const as = String(a?.ReviewedAt ?? "");
      const bs = String(b?.ReviewedAt ?? "");
      const at = Date.parse(as);
      const bt = Date.parse(bs);

      const aValid = Number.isFinite(at);
      const bValid = Number.isFinite(bt);

      if (aValid && bValid) return (at - bt) * dir;
      if (aValid && !bValid) return -1 * dir;
      if (!aValid && bValid) return 1 * dir;

      return as.localeCompare(bs) * dir;
    });

    return arr;
  }, [Items, SortBy, SortDir]);

  return (
    <PageStyle
      layout="grid-2"
      title={Title}
      onBack={OnBack}
      onSave={undefined}
      onEdit={undefined}
      onDelete={undefined}
      onCancel={undefined}
    >
      {/* --- 左ペイン：reviews（カード表示） --- */}
      <div>
        <div className="pbrd-toolbar">
          <div className="pbrd-toolbar-left" />

          <div className="pbrd-toolbar-right">
            {/* Status フィルター（日本語ラベル） */}
            <select
              value={Status}
              onChange={(E) => SetStatus(E.target.value as any)}
              className="border rounded px-2 py-1"
            >
              <option value="PUBLISHED">{statusLabelJa("PUBLISHED")}</option>
              <option value="HIDDEN">{statusLabelJa("HIDDEN")}</option>
              <option value="REMOVED">{statusLabelJa("REMOVED")}</option>
            </select>

            {/* sort buttons */}
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => toggleSort("Rating")}
              aria-label="Rating でソート"
              title="Rating でソート"
            >
              評価 {sortLabel("Rating")}
            </Button>

            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => toggleSort("ReviewedAt")}
              aria-label="ReviewedAt でソート"
              title="ReviewedAt でソート"
            >
              投稿日時 {sortLabel("ReviewedAt")}
            </Button>

            <RefreshButton
              onClick={OnReload}
              loading={IsLoading}
              title="リフレッシュ"
              ariaLabel="リフレッシュ"
            />
          </div>
        </div>

        {ErrorMessage ? <div className="text-sm text-red-600 mb-3">{ErrorMessage}</div> : null}

        <div className="pbrd-reviewcard-wrapper">
          {IsLoading ? (
            <div className="pbrd-empty">読み込み中...</div>
          ) : SortedItems.length === 0 ? (
            <div className="pbrd-empty">No reviews</div>
          ) : (
            <div className="pbrd-grid">
              {SortedItems.map((R, idx) => {
                const ReviewID = String(R.ID ?? `rv_${idx}`);
                const Body = String(R.Body ?? "");
                const TitleText = String(R.Title ?? "");

                // ✅ AvatarID 表示をやめて、AvatarName / AvatarIcon を表示
                const AvatarName = String(R.AvatarName ?? "");
                const AvatarIcon = String(R.AvatarIcon ?? "");
                const authorPrimary = AvatarName || "-";

                const RatingNum = Number(R.Rating ?? 0);
                const RatingStars = ratingToStars(RatingNum);
                const ReviewedAt = String(R.ReviewedAt ?? "");

                // ✅ 日本語ラベル
                const statusJa = statusLabelJa(R.Status);

                return (
                  <div
                    key={ReviewID}
                    className="bg-white border border-slate-200 rounded-xl shadow-sm pbrd-review-item-card"
                  >
                    <div className="pbrd-author-row">
                      {AvatarIcon ? (
                        <img
                          src={AvatarIcon}
                          alt="author icon"
                          className="pbrd-author-icon"
                        />
                      ) : null}

                      <span className="pbrd-author-primary">{authorPrimary}</span>

                      {/* status: 日本語ラベル */}
                      <span className="pbrd-pill">{statusJa}</span>

                      {/* rating: 星表現 */}
                      <span className="pbrd-pill">{RatingStars}</span>
                    </div>

                    <div className="pbrd-title">
                      {TitleText || <span className="pbrd-body-empty">（タイトルなし）</span>}
                    </div>

                    <div className="pbrd-body">
                      {Body || <span className="pbrd-body-empty">（本文なし）</span>}
                    </div>

                    <div className="pbrd-datetime">投稿日時: {ReviewedAt || "-"}</div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        <Pagination currentPage={Page} totalPages={TotalPages} onPageChange={SetPage} />
      </div>

      {/* --- 右ペイン：管理情報 + ログ --- */}
      <div>
        <AdminCard
          title="管理情報"
          assigneeName={HeaderAssigneeName}
          createdByName=""
          createdAt=""
          updatedByName=""
          updatedAt=""
          mode="view"
          onClickAssignee={Noop}
        />

        <div className="section-gap">
          <LogCard />
        </div>
      </div>
    </PageStyle>
  );
}