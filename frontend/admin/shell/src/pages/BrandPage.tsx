// frontend/admin/shell/src/pages/BrandPage.tsx

import { useNavigate, useParams } from "react-router-dom";

import { useBrandDetail } from "../features/brand/presentation/hooks/useBrandDetail";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
import { formatDateTime } from "../shared/util/dateFormat";

import "./BrandPage.css";

export default function BrandPage() {
  const navigate = useNavigate();
  const { companyId = "", brandId = "" } = useParams<{
    companyId?: string;
    brandId?: string;
  }>();

  const {
    company,
    brand,
    brandIconUrl,
    brandBackgroundImageUrl,
    loading,
    error,
    reload,
  } = useBrandDetail(companyId, brandId);

  const renderMain = () => {
    if (loading && !brand) {
      return <p>ブランド詳細を取得しています...</p>;
    }

    if (error && !brand) {
      return (
        <div role="alert">
          <p>ブランド詳細を取得できませんでした。</p>
          <p>{error}</p>
          <button type="button" onClick={() => void reload()}>
            再読み込み
          </button>
        </div>
      );
    }

    if (!brand) {
      return <p role="alert">ブランド情報を取得できませんでした。</p>;
    }

    return (
      <section className="ui-detail-section">
        <div className="brand-page__background-area">
          {brandBackgroundImageUrl ? (
            <img
              src={brandBackgroundImageUrl}
              alt={`${brand.name || "ブランド"}の背景画像`}
              className="brand-page__background"
            />
          ) : (
            <div className="brand-page__background-placeholder">
              背景画像未設定
            </div>
          )}
        </div>

        <div className="brand-page__summary">
          <div className="brand-page__icon-area">
            {brandIconUrl ? (
              <img
                src={brandIconUrl}
                alt={`${brand.name || "ブランド"}のアイコン`}
                className="brand-page__icon"
              />
            ) : (
              <div className="brand-page__icon-placeholder">
                アイコン未設定
              </div>
            )}
          </div>

          <div className="brand-page__summary-fields">
            <div className="brand-page__field">
              <div className="brand-page__field-label">ブランド名</div>
              <div className="brand-page__field-value">
                {brand.name || "-"}
              </div>
            </div>

            <div className="brand-page__field">
              <div className="brand-page__field-label">責任者</div>
              <div className="brand-page__field-value">
                {brand.managerName || "-"}
              </div>
            </div>
          </div>
        </div>
      </section>
    );
  };

  return (
    <Page>
      <PageHeader
        title={brand?.name || "ブランド詳細"}
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

      {company && brand ? (
        <DetailPageBody
          main={renderMain()}
          aside={
            <section className="ui-detail-section">
              <dl className="ui-detail-definition-list ui-detail-definition-list--meta">
                <dt>企業名</dt>
                <dd>{company.name || "-"}</dd>

                <dt>責任者</dt>
                <dd>{brand.managerName || "-"}</dd>

                <dt>状態</dt>
                <dd>{brand.isActive ? "有効" : "無効"}</dd>

                <dt>登録日時</dt>
                <dd>
                  {brand.createdAt
                    ? formatDateTime(brand.createdAt)
                    : "-"}
                </dd>

                <dt>最終更新日時</dt>
                <dd>
                  {brand.updatedAt
                    ? formatDateTime(brand.updatedAt)
                    : "-"}
                </dd>
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