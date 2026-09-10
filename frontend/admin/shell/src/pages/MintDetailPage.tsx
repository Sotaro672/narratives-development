// frontend/admin/shell/src/pages/MintDetailPage.tsx

import { useNavigate, useParams } from "react-router-dom";

import { useMintDetail } from "../features/mint/hooks/useMintDetail";
import type { MintDetailModel } from "../shared/type/mint";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";

import "./MintDetailPage.css";

function formatRgb(rgb: number | undefined): string {
  if (rgb === undefined) return "-";
  return `#${rgb.toString(16).padStart(6, "0").toUpperCase()}`;
}

function renderModelMetadata(model: MintDetailModel) {
  const measurements = Object.entries(model.measurements ?? {});

  return (
    <dl className="ui-detail-definition-list mint-detail-page__model-definition-list">
      <dt>Model ID</dt>
      <dd>{model.modelId || "-"}</dd>

      <dt>種別</dt>
      <dd>{model.kind || "-"}</dd>

      <dt>モデル番号</dt>
      <dd>{model.modelNumber || "-"}</dd>

      {model.size ? (
        <>
          <dt>サイズ</dt>
          <dd>{model.size}</dd>
        </>
      ) : null}

      {model.colorName ? (
        <>
          <dt>カラー</dt>
          <dd>{model.colorName}</dd>
        </>
      ) : null}

      {model.rgb !== undefined ? (
        <>
          <dt>RGB</dt>
          <dd>{formatRgb(model.rgb)}</dd>
        </>
      ) : null}

      {model.volume !== undefined ? (
        <>
          <dt>容量</dt>
          <dd>
            {model.volume.toLocaleString()}
            {model.volumeUnit || ""}
          </dd>
        </>
      ) : null}

      {measurements.map(([key, value]) => (
        <div className="mint-detail-page__measurement" key={key}>
          <dt>{key}</dt>
          <dd>{value.toLocaleString()}</dd>
        </div>
      ))}
    </dl>
  );
}

export default function MintDetailPage() {
  const navigate = useNavigate();
  const { mintId = "" } = useParams<{ mintId: string }>();
  const { detail, loading, error, reload } = useMintDetail(mintId);

  const renderMain = () => {
    if (loading && !detail) {
      return <p>Mint詳細を取得しています...</p>;
    }

    if (error && !detail) {
      return (
        <div role="alert">
          <p>Mint詳細を取得できませんでした。</p>
          <p>{error}</p>
          <button type="button" onClick={() => void reload()}>
            再読み込み
          </button>
        </div>
      );
    }

    if (!detail) {
      return <p role="alert">Mint情報を取得できませんでした。</p>;
    }

    if (detail.models.length === 0) {
      return (
        <section className="ui-detail-section">
          <h2 className="ui-detail-section__title">モデル</h2>
          <p>Mint対象のモデルはありません。</p>
        </section>
      );
    }

    return (
      <section className="ui-detail-section">
        <h2 className="ui-detail-section__title">モデル</h2>
        <div className="mint-detail-page__models">
          {detail.models.map((model) => (
            <article className="mint-detail-page__model" key={model.modelId}>
              <div className="mint-detail-page__model-header">
                <div>
                  <h3 className="mint-detail-page__model-title">
                    {model.modelNumber || model.modelId}
                  </h3>
                  {model.kind ? (
                    <p className="mint-detail-page__model-kind">{model.kind}</p>
                  ) : null}
                </div>
                <div className="mint-detail-page__product-count">
                  <span className="mint-detail-page__product-count-value">
                    {model.productCount.toLocaleString()}
                  </span>
                  <span className="mint-detail-page__product-count-unit">点</span>
                </div>
              </div>

              {renderModelMetadata(model)}
            </article>
          ))}
        </div>
      </section>
    );
  };

  return (
    <Page>
      <PageHeader
        title={detail?.tokenName || "Mint詳細"}
        meta={mintId || undefined}
        leading={
          <button
            type="button"
            className="ui-page-header__back"
            aria-label="戻る"
            onClick={() => navigate("/gas")}
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

      {detail ? (
        <DetailPageBody
          main={renderMain()}
          aside={
            <section className="ui-detail-section">
              <dl className="ui-detail-definition-list">
                <dt>企業名</dt>
                <dd>{detail.companyName || "-"}</dd>

                <dt>トークンブランド名</dt>
                <dd>{detail.tokenBrandName || "-"}</dd>

                <dt>トークン名</dt>
                <dd>{detail.tokenName || "-"}</dd>

                <dt>商品ブランド名</dt>
                <dd>{detail.productBrandName || "-"}</dd>

                <dt>商品名</dt>
                <dd>{detail.productName || "-"}</dd>
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