// frontend/admin/shell/src/pages/ListDetailPage.tsx

import { useNavigate, useParams } from "react-router-dom";

import { useContractListDetail } from "../features/company/presentation/hooks/useContractListDetail";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
import { formatDateTime } from "../shared/util/dateFormat";

function formatListStatus(status: string): string {
  switch (status) {
    case "listing":
      return "出品中";
    case "suspended":
      return "停止中";
    default:
      return status || "-";
  }
}

export default function ListDetailPage() {
  const navigate = useNavigate();
  const { companyId = "", listId = "" } = useParams<{
    companyId?: string;
    listId?: string;
  }>();
  const { detail, loading, error, reload } = useContractListDetail(
    companyId,
    listId,
  );

  const company = detail?.company ?? null;
  const list = detail?.list ?? null;

  const renderMain = () => {
    if (loading && !detail) {
      return <p>出品詳細を取得しています...</p>;
    }

    if (error && !detail) {
      return (
        <div role="alert">
          <p>出品詳細を取得できませんでした。</p>
          <p>{error}</p>
          <button type="button" onClick={() => void reload()}>
            再読み込み
          </button>
        </div>
      );
    }

    if (!list) {
      return <p role="alert">出品情報を取得できませんでした。</p>;
    }

    return (
      <>
        <section className="ui-detail-section">
          <h2 className="ui-detail-section__title">出品情報</h2>
          <dl className="ui-detail-definition-list">
            <dt>出品名</dt>
            <dd>{list.title || "-"}</dd>

            <dt>出品ID</dt>
            <dd>{list.readableId || list.id || "-"}</dd>

            <dt>状態</dt>
            <dd>{formatListStatus(list.status)}</dd>

            <dt>商品</dt>
            <dd>{list.productName || "-"}</dd>

            <dt>トークン設計</dt>
            <dd>{list.tokenName || "-"}</dd>

            <dt>ブランド</dt>
            <dd>{list.brandName || "-"}</dd>

            <dt>担当者</dt>
            <dd>{list.assigneeName || "-"}</dd>

            <dt>説明</dt>
            <dd>{list.description || "-"}</dd>

            <dt>作成日時</dt>
            <dd>{formatDateTime(list.createdAt)}</dd>

            <dt>最終更新日時</dt>
            <dd>{formatDateTime(list.updatedAt)}</dd>
          </dl>
        </section>

        <section className="ui-detail-section">
          <h2 className="ui-detail-section__title">関連ID</h2>
          <dl className="ui-detail-definition-list">
            <dt>内部ID</dt>
            <dd>{list.id || "-"}</dd>

            <dt>在庫ID</dt>
            <dd>{list.inventoryId || "-"}</dd>

            <dt>商品設計ID</dt>
            <dd>{list.productBlueprintId || "-"}</dd>

            <dt>トークン設計ID</dt>
            <dd>{list.tokenBlueprintId || "-"}</dd>

            <dt>画像ID</dt>
            <dd>{list.imageId || "-"}</dd>
          </dl>
        </section>

        <section className="ui-detail-section">
          <h2 className="ui-detail-section__title">価格</h2>
          {list.prices.length > 0 ? (
            <dl className="ui-detail-definition-list">
              {list.prices.map((price) => (
                <div key={price.modelId}>
                  <dt>{price.modelId || "モデル"}</dt>
                  <dd>{price.price.toLocaleString("ja-JP")}円</dd>
                </div>
              ))}
            </dl>
          ) : (
            <p>価格情報はありません。</p>
          )}
        </section>
      </>
    );
  };

  return (
    <Page>
      <PageHeader
        title={list?.title || list?.readableId || "出品詳細"}
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
              <h2 className="ui-detail-section__title">企業情報</h2>
              <dl className="ui-detail-definition-list">
                <dt>企業名</dt>
                <dd>{company.name || "-"}</dd>

                <dt>代表者</dt>
                <dd>{company.representativeName || "-"}</dd>

                <dt>登録日時</dt>
                <dd>{formatDateTime(company.createdAt)}</dd>

                <dt>最終更新日</dt>
                <dd>{formatDateTime(company.updatedAt)}</dd>
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