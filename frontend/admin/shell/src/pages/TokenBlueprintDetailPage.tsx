// frontend/admin/shell/src/pages/TokenBlueprintDetailPage.tsx

import { useNavigate, useParams } from "react-router-dom";

import { useContractTokenBlueprintDetail } from "../features/company/presentation/hooks/useContractTokenBlueprintDetail";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
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
          <dl className="ui-detail-definition-list token-blueprint-detail-page__definition-list">
            <dt>シンボル</dt>
            <dd>{tokenBlueprint.symbol || "-"}</dd>

            <dt>ブランド</dt>
            <dd>{tokenBlueprint.brandName || "-"}</dd>

            <dt>担当者</dt>
            <dd>{tokenBlueprint.assigneeName || "-"}</dd>

            <dt>モデレーション状態</dt>
            <dd>{tokenBlueprint.moderationStatus || "-"}</dd>

            <dt>説明</dt>
            <dd>{tokenBlueprint.description || "-"}</dd>

            <dt>作成日時</dt>
            <dd>{formatDateTime(tokenBlueprint.createdAt)}</dd>

            <dt>最終更新日時</dt>
            <dd>{formatDateTime(tokenBlueprint.updatedAt)}</dd>
          </dl>
        </section>

        <section className="ui-detail-section">
          <dl className="ui-detail-definition-list token-blueprint-detail-page__definition-list">
            <dt>Metadata URI</dt>
            <dd>{tokenBlueprint.metadataUri || "-"}</dd>

            <dt>アイコンURL</dt>
            <dd>{tokenBlueprint.iconUrl || "-"}</dd>
          </dl>
        </section>

        <section className="ui-detail-section">
          {tokenBlueprint.contentFiles.length > 0 ? (
            tokenBlueprint.contentFiles.map((file) => (
              <dl
                key={file.id}
                className="ui-detail-definition-list token-blueprint-detail-page__definition-list"
              >
                <dt>ファイル名</dt>
                <dd>{file.name || "-"}</dd>

                <dt>タイプ</dt>
                <dd>{file.type || "-"}</dd>

                <dt>Content-Type</dt>
                <dd>{file.contentType || "-"}</dd>

                <dt>公開状態</dt>
                <dd>{file.isPublic ? "公開" : "非公開"}</dd>

                <dt>サイズ</dt>
                <dd>{file.size.toLocaleString("ja-JP")} bytes</dd>

                <dt>URL</dt>
                <dd>{file.url || "-"}</dd>

                <dt>作成日時</dt>
                <dd>{formatDateTime(file.createdAt)}</dd>

                <dt>最終更新日時</dt>
                <dd>{formatDateTime(file.updatedAt)}</dd>
              </dl>
            ))
          ) : (
            <p>コンテンツファイルはありません。</p>
          )}
        </section>
      </>
    );
  };

  return (
    <Page>
      <PageHeader
        title={tokenBlueprint?.name || "トークン設計詳細"}
        leading={
          <button
            type="button"
            className="ui-page-header__back"
            aria-label="戻る"
            onClick={() =>
              navigate(
                companyId
                  ? `/contracts/${encodeURIComponent(companyId)}`
                  : "/contracts",
              )
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

      {company ? (
        <DetailPageBody
          main={renderMain()}
          aside={
            <section className="ui-detail-section">
              <dl className="ui-detail-definition-list token-blueprint-detail-page__definition-list">
                <dt>企業名</dt>
                <dd>{company.name || "-"}</dd>
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