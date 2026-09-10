// frontend/admin/shell/src/pages/TokenBlueprintDetailPage.tsx

import { useMemo } from "react";
import { useNavigate, useParams } from "react-router-dom";

import TokenBlueprintReviewTable from "../features/company/presentation/components/TokenBlueprintReviewTable";
import { useContractTokenBlueprintDetail } from "../features/company/presentation/hooks/useContractTokenBlueprintDetail";
import MediaGallery, { type MediaGalleryItem } from "../shared/ui/MediaGallery/MediaGallery";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
import Tab from "../shared/ui/Tab/Tab";
import { formatDateTime } from "../shared/util/dateFormat";

import "./TokenBlueprintDetailPage.css";

export default function TokenBlueprintDetailPage() {
  const navigate = useNavigate();
  const { companyId = "", tokenBlueprintId = "" } = useParams<{
    companyId?: string;
    tokenBlueprintId?: string;
  }>();
  const { detail, loading, error, reload } = useContractTokenBlueprintDetail(
    companyId,
    tokenBlueprintId,
  );

  const company = detail?.company ?? null;
  const tokenBlueprint = detail?.tokenBlueprint ?? null;
  const reportCaseId = tokenBlueprint ? `tokenBlueprint_${tokenBlueprint.id}` : "";

  const galleryItems = useMemo<MediaGalleryItem[]>(() => {
    if (!tokenBlueprint) return [];

    return tokenBlueprint.contentFiles
      .filter((file) => file.type === "image" && Boolean(file.id) && Boolean(file.url))
      .map((file) => ({
        id: file.id,
        url: file.url,
        fileName: file.name || undefined,
      }));
  }, [tokenBlueprint]);

  const moderationTone =
    tokenBlueprint?.moderationStatus === "ACTIVE"
      ? "success"
      : tokenBlueprint?.moderationStatus === "HIDDEN_BY_MODERATION"
        ? "danger"
        : "neutral";

  const renderMain = () => {
    if (loading && !detail) {
      return <p>トークン設計詳細を取得しています...</p>;
    }

    if (error && !detail) {
      return (
        <div role="alert">
          <p>トークン設計詳細を取得できませんでした。</p>
          <p>{error}</p>
          <button type="button" onClick={() => void reload()}>
            再読み込み
          </button>
        </div>
      );
    }

    if (!tokenBlueprint) {
      return <p role="alert">トークン設計情報を取得できませんでした。</p>;
    }

    return (
      <>
        <section className="ui-detail-section">
          <div className="token-blueprint-detail-page__summary">
            <div className="token-blueprint-detail-page__icon-area">
              {tokenBlueprint.iconUrl ? (
                <img
                  src={tokenBlueprint.iconUrl}
                  alt={`${tokenBlueprint.name || "トークン"}のアイコン`}
                  className="token-blueprint-detail-page__icon"
                />
              ) : (
                <div className="token-blueprint-detail-page__icon-placeholder">
                  アイコン未設定
                </div>
              )}
            </div>

            <div className="token-blueprint-detail-page__summary-fields">
              <div className="token-blueprint-detail-page__field">
                <div className="token-blueprint-detail-page__field-label">シンボル</div>
                <div className="token-blueprint-detail-page__field-value">
                  {tokenBlueprint.symbol || "-"}
                </div>
              </div>

              <div className="token-blueprint-detail-page__field">
                <div className="token-blueprint-detail-page__field-label">説明</div>
                <div className="token-blueprint-detail-page__description">
                  {tokenBlueprint.description || "-"}
                </div>
              </div>
            </div>
          </div>
        </section>

        <section className="ui-detail-section">
          <MediaGallery
            items={galleryItems}
            altFallback={tokenBlueprint.name || "コンテンツ画像"}
            placeholderText="コンテンツ画像はありません。"
          />
        </section>

        <section className="ui-detail-section">
          <h2 className="ui-detail-section__title">レビュー</h2>
          <TokenBlueprintReviewTable
            companyId={companyId}
            tokenBlueprintId={tokenBlueprintId}
            perPage={20}
          />
        </section>
      </>
    );
  };

  return (
    <Page>
      <PageHeader
        title={tokenBlueprint?.name || "トークン設計詳細"}
        meta={
          tokenBlueprint?.moderationStatus ? (
            <Tab
              tone={moderationTone}
              aria-label={`モデレーション状態 ${tokenBlueprint.moderationStatus}`}
            >
              {tokenBlueprint.moderationStatus}
            </Tab>
          ) : undefined
        }
        leading={
          <button
            type="button"
            className="ui-page-header__back"
            aria-label="戻る"
            onClick={() => navigate(-1)}
          >
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <path d="M19 12H5" />
              <path d="M12 19l-7-7 7-7" />
            </svg>
          </button>
        }
      />

      {company && tokenBlueprint ? (
        <DetailPageBody
          main={renderMain()}
          aside={
            <section className="ui-detail-section">
              <dl className="ui-detail-definition-list ui-detail-definition-list--meta">
                <dt>企業名</dt>
                <dd>{company.name || "-"}</dd>

                <dt>ブランド</dt>
                <dd>{tokenBlueprint.brandName || "-"}</dd>

                <dt>担当者</dt>
                <dd>{tokenBlueprint.assigneeName || "-"}</dd>

                <dt>高評価</dt>
                <dd>{tokenBlueprint.likeCount.toLocaleString()}</dd>

                <dt>低評価</dt>
                <dd>{tokenBlueprint.dislikeCount.toLocaleString()}</dd>

                <dt>コメント</dt>
                <dd>{tokenBlueprint.commentCount.toLocaleString()}</dd>

                <dt>通報数</dt>
                <dd>
                  {tokenBlueprint.reportCount > 0 ? (
                    <button
                      type="button"
                      className="token-blueprint-detail-page__report-link"
                      onClick={() =>
                        navigate(`/reports/${encodeURIComponent(reportCaseId)}`)
                      }
                    >
                      {tokenBlueprint.reportCount.toLocaleString()}
                    </button>
                  ) : (
                    tokenBlueprint.reportCount.toLocaleString()
                  )}
                </dd>

                <dt>作成日時</dt>
                <dd>{formatDateTime(tokenBlueprint.createdAt)}</dd>

                <dt>最終更新日時</dt>
                <dd>{formatDateTime(tokenBlueprint.updatedAt)}</dd>
              </dl>
            </section>
          }
        />
      ) : (
        renderMain()
      )}
    </Page>
  );
}