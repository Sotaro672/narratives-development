// frontend/admin/shell/src/pages/ResalePage.tsx

import { useNavigate, useParams } from "react-router-dom";

import { useAvatarResales } from "../features/avatar/presentation/hooks/useAvatarResales";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
import { formatDateTime } from "../shared/util/dateFormat";

import "./ResalePage.css";

const STATUS_LABELS: Record<string, string> = {
  listing: "出品中",
  suspended: "停止中",
  sold: "売却済み",
};

export default function ResalePage() {
  const navigate = useNavigate();
  const { avatarId = "", resaleId = "" } = useParams<{
    avatarId?: string;
    resaleId?: string;
  }>();

  const { resales, loading, error, reload } = useAvatarResales(avatarId);
  const resale = resales.find((item) => item.id === resaleId) ?? null;

  const renderAside = () => {
    if (loading) {
      return <p>Resaleを取得しています...</p>;
    }

    if (error) {
      return (
        <div role="alert">
          <p>Resaleを取得できませんでした。</p>
          <p>{error}</p>
          <button type="button" onClick={() => void reload()}>
            再読み込み
          </button>
        </div>
      );
    }

    if (!resale) {
      return <p role="alert">Resaleが見つかりませんでした。</p>;
    }

    return (
      <section className="ui-detail-section resale-page">
        <dl className="ui-detail-definition-list">
          <div>
            <dt>商品名</dt>
            <dd>{resale.productName || "-"}</dd>
          </div>
          <div>
            <dt>トークン名</dt>
            <dd>{resale.tokenName || "-"}</dd>
          </div>
          <div>
            <dt>ステータス</dt>
            <dd>{STATUS_LABELS[resale.status] ?? resale.status}</dd>
          </div>
          <div>
            <dt>価格</dt>
            <dd>{resale.price.toLocaleString("ja-JP")}円</dd>
          </div>
          <div>
            <dt>通報数</dt>
            <dd>{resale.reportCount}</dd>
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
        leading={
          <button
            type="button"
            className="ui-page-header__back"
            aria-label="戻る"
            onClick={() =>
              navigate(`/avatars/${encodeURIComponent(avatarId)}`)
            }
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

      <DetailPageBody main={null} aside={renderAside()} />
    </Page>
  );
}