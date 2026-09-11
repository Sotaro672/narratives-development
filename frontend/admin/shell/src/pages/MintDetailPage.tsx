// frontend/admin/shell/src/pages/MintDetailPage.tsx

import { useNavigate, useParams } from "react-router-dom";

import { useMintDetail } from "../features/mint/hooks/useMintDetail";
import type { MintDetailModel, MintStatus } from "../shared/type/mint";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
import Tab, { type TabTone } from "../shared/ui/Tab/Tab";
import TextLink from "../shared/ui/TextLink/TextLink";

import "./MintDetailPage.css";

const STATUS_LABELS: Record<MintStatus, string> = {
  CREATED: "作成済み",
  QUEUED: "待機中",
  MINTING: "ミント中",
  PARTIALLY_MINTED: "一部ミント済み",
  MINTED: "ミント完了",
  FAILED_RETRYABLE: "再試行可能",
  FAILED_FATAL: "ミント失敗",
};

function getStatusTone(status: MintStatus): TabTone {
  switch (status) {
    case "MINTED":
      return "success";
    case "QUEUED":
    case "MINTING":
    case "PARTIALLY_MINTED":
      return "warning";
    case "FAILED_RETRYABLE":
    case "FAILED_FATAL":
      return "danger";
    default:
      return "neutral";
  }
}

function rgbToCssColor(rgb: number): string {
  return `#${rgb.toString(16).padStart(6, "0").slice(-6)}`;
}

function renderModelMetadata(model: MintDetailModel) {
  const measurements = Object.entries(model.measurements ?? {});

  return (
    <dl className="mint-detail-page__model-definition-list">
      {model.size ? (
        <div className="mint-detail-page__definition-row">
          <dt>サイズ</dt>
          <dd>{model.size}</dd>
        </div>
      ) : null}

      {model.colorName ? (
        <div className="mint-detail-page__definition-row">
          <dt>カラー</dt>
          <dd className="mint-detail-page__color-value">
            <span>{model.colorName}</span>
            {model.rgb !== undefined ? (
              <span
                className="mint-detail-page__color-chip"
                style={{ backgroundColor: rgbToCssColor(model.rgb) }}
                aria-label={`${model.colorName}の色`}
                title={rgbToCssColor(model.rgb)}
              />
            ) : null}
          </dd>
        </div>
      ) : null}

      {model.volume !== undefined ? (
        <div className="mint-detail-page__definition-row">
          <dt>容量</dt>
          <dd>
            {model.volume.toLocaleString()}
            {model.volumeUnit || ""}
          </dd>
        </div>
      ) : null}

      {measurements.length > 0 ? (
        <div className="mint-detail-page__definition-row">
          <dt>採寸</dt>
          <dd className="mint-detail-page__measurements">
            {measurements.map(([key, value]) => (
              <span key={key}>
                {key}: {value.toLocaleString()}
              </span>
            ))}
          </dd>
        </div>
      ) : null}
    </dl>
  );
}

export default function MintDetailPage() {
  const navigate = useNavigate();
  const { mintId = "" } = useParams<{ mintId: string }>();
  const { detail, loading, error, reload } = useMintDetail(mintId);

  const renderMain = () => {
    if (loading && !detail) return <p>Mint詳細を取得しています...</p>;

    if (error && !detail) {
      return (
        <div role="alert">
          <p>Mint詳細を取得できませんでした。</p>
          <p>{error}</p>
          <button type="button" onClick={() => void reload()}>再読み込み</button>
        </div>
      );
    }

    if (!detail) return <p role="alert">Mint情報を取得できませんでした。</p>;

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
        <div className="mint-detail-page__models">
          {detail.models.map((model) => (
            <article className="mint-detail-page__model" key={model.modelId}>
              <div className="mint-detail-page__model-header">
                <h3 className="mint-detail-page__model-title">{model.modelNumber || model.modelId}</h3>

                <div className="mint-detail-page__product-count">
                  <span className="mint-detail-page__product-count-value">{model.productCount.toLocaleString()}</span>
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
        meta={
          detail?.status ? (
            <Tab tone={getStatusTone(detail.status)} aria-label={`Mintステータス ${STATUS_LABELS[detail.status]}`}>
              {STATUS_LABELS[detail.status]}
            </Tab>
          ) : undefined
        }
        leading={
          <button type="button" className="ui-page-header__back" aria-label="戻る" onClick={() => navigate("/gas")}>
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
              <dl className="ui-detail-definition-list ui-detail-definition-list--meta">
                <dt>企業名</dt>
                <dd>{detail.companyName || "-"}</dd>

                <dt>トークンブランド名</dt>
                <dd>{detail.tokenBrandName || "-"}</dd>

                <dt>トークン名</dt>
                <dd>
                  <TextLink
                    tone="inherit"
                    onClick={() =>
                      navigate(`/contracts/${encodeURIComponent(detail.companyId)}/token-blueprints/${encodeURIComponent(detail.tokenBlueprintId)}`)
                    }
                  >
                    {detail.tokenName || "-"}
                  </TextLink>
                </dd>

                <dt>商品ブランド名</dt>
                <dd>{detail.productBrandName || "-"}</dd>

                <dt>商品名</dt>
                <dd>
                  <TextLink
                    tone="inherit"
                    onClick={() =>
                      navigate(`/contracts/${encodeURIComponent(detail.companyId)}/product-blueprints/${encodeURIComponent(detail.productBlueprintId)}`)
                    }
                  >
                    {detail.productName || "-"}
                  </TextLink>
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