// frontend/admin/shell/src/pages/ResalePage.tsx

import { useNavigate, useParams } from "react-router-dom";

import { useAvatarResales } from "../features/avatar/presentation/hooks/useAvatarResales";
import MediaGallery, { type MediaGalleryItem } from "../shared/ui/MediaGallery/MediaGallery";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
import Tab, { type TabTone } from "../shared/ui/Tab/Tab";
import TextLink from "../shared/ui/TextLink/TextLink";
import { formatDateTime } from "../shared/util/dateFormat";

const STATUS_LABELS: Record<string, string> = {
  listing: "出品中",
  suspended: "停止中",
  sold: "売却済み",
};

function getStatusTone(status: string): TabTone {
  switch (status) {
    case "listing":
      return "success";
    case "suspended":
      return "danger";
    case "sold":
      return "neutral";
    default:
      return "neutral";
  }
}

export default function ResalePage() {
  const navigate = useNavigate();
  const { avatarId = "", resaleId = "" } = useParams<{
    avatarId?: string;
    resaleId?: string;
  }>();

  const { resales, loading, error, reload } = useAvatarResales(avatarId);
  const resale = resales.find((item) => item.id === resaleId) ?? null;

  const galleryItems: MediaGalleryItem[] = resale
    ? [...resale.images]
        .sort((a, b) => a.displayOrder - b.displayOrder)
        .filter((image) => Boolean(image.id) && Boolean(image.url))
        .map((image) => ({
          id: image.id,
          url: image.url,
        }))
    : [];

  const renderMain = () => {
    if (!resale) return null;

    return (
      <section className="ui-detail-section">
        <MediaGallery
          items={galleryItems}
          altFallback={resale.productName || "再販商品画像"}
          placeholderText="再販画像はありません。"
        />
      </section>
    );
  };

  const renderAside = () => {
    if (loading) return <p>Resaleを取得しています...</p>;

    if (error) {
      return (
        <div role="alert">
          <p>Resaleを取得できませんでした。</p>
          <p>{error}</p>
          <button type="button" onClick={() => void reload()}>再読み込み</button>
        </div>
      );
    }

    if (!resale) return <p role="alert">Resaleが見つかりませんでした。</p>;

    return (
      <section className="ui-detail-section">
        <dl className="ui-detail-definition-list ui-detail-definition-list--rows">
          <div>
            <dt>商品名</dt>
            <dd>
              <TextLink
                tone="inherit"
                onClick={() =>
                  navigate(`/contracts/${encodeURIComponent(resale.companyId)}/product-blueprints/${encodeURIComponent(resale.productBlueprintId)}`)
                }
              >
                {resale.productName || "-"}
              </TextLink>
            </dd>
          </div>

          <div>
            <dt>トークン名</dt>
            <dd>
              <TextLink
                tone="inherit"
                onClick={() =>
                  navigate(`/contracts/${encodeURIComponent(resale.companyId)}/token-blueprints/${encodeURIComponent(resale.tokenBlueprintId)}`)
                }
              >
                {resale.tokenName || "-"}
              </TextLink>
            </dd>
          </div>

          <div>
            <dt>価格</dt>
            <dd>{resale.price.toLocaleString("ja-JP")}円</dd>
          </div>

          <div>
            <dt>商品の状態</dt>
            <dd>{resale.condition || "-"}</dd>
          </div>

          <div>
            <dt>通報数</dt>
            <dd>
              {resale.reportCount > 0 && resale.reportCaseId ? (
                <TextLink
                  tone="accent"
                  onClick={() => navigate(`/reports/${encodeURIComponent(resale.reportCaseId)}`)}
                >
                  {resale.reportCount.toLocaleString("ja-JP")}
                </TextLink>
              ) : (
                resale.reportCount.toLocaleString("ja-JP")
              )}
            </dd>
          </div>

          <div>
            <dt>登録日時</dt>
            <dd>{formatDateTime(resale.createdAt)}</dd>
          </div>

          <div>
            <dt>最終更新日時</dt>
            <dd>{resale.updatedAt ? formatDateTime(resale.updatedAt) : "-"}</dd>
          </div>
        </dl>
      </section>
    );
  };

  return (
    <Page>
      <PageHeader
        title={resale?.productName || "Resale詳細"}
        meta={
          resale?.status ? (
            <Tab
              tone={getStatusTone(resale.status)}
              aria-label={`Resale状態 ${STATUS_LABELS[resale.status] ?? resale.status}`}
            >
              {STATUS_LABELS[resale.status] ?? resale.status}
            </Tab>
          ) : undefined
        }
        leading={
          <button
            type="button"
            className="ui-page-header__back"
            aria-label="戻る"
            onClick={() => navigate(`/avatars/${encodeURIComponent(avatarId)}`)}
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

      <DetailPageBody main={renderMain()} aside={renderAside()} />
    </Page>
  );
}