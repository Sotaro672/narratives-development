// frontend/admin/shell/src/pages/ProductBlueprintDetailPage.tsx

import { useNavigate, useParams } from "react-router-dom";

import { useContractProductBlueprintDetail } from "../features/company/presentation/hooks/useContractProductBlueprintDetail";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";
import { formatDateTime } from "../shared/util/dateFormat";
import { formatModelMeta } from "../shared/util/modelMetaFormat";

import "./ProductBlueprintDetailPage.css";

function formatPrinted(printed: boolean): string {
  return printed ? "印刷済み" : "未印刷";
}

function formatCategoryFields(
  categoryFields: Record<string, unknown>,
): string {
  const entries = Object.entries(categoryFields);
  if (entries.length === 0) {
    return "-";
  }

  return entries
    .map(([key, value]) => {
      if (value == null) {
        return `${key}: -`;
      }
      if (typeof value === "object") {
        return `${key}: ${JSON.stringify(value)}`;
      }
      return `${key}: ${String(value)}`;
    })
    .join("\n");
}

export default function ProductBlueprintDetailPage() {
  const navigate = useNavigate();
  const { companyId = "", productBlueprintId = "" } = useParams<{
    companyId?: string;
    productBlueprintId?: string;
  }>();
  const { detail, loading, error, reload } =
    useContractProductBlueprintDetail(companyId, productBlueprintId);

  const company = detail?.company ?? null;
  const productBlueprint = detail?.productBlueprint ?? null;

  const renderMain = () => {
    if (loading && !detail) {
      return <p>商品設計詳細を取得しています...</p>;
    }

    if (error && !detail) {
      return (
        <div role="alert">
          <p>商品設計詳細を取得できませんでした。</p>
          <p>{error}</p>
          <button type="button" onClick={() => void reload()}>
            再読み込み
          </button>
        </div>
      );
    }

    if (!productBlueprint) {
      return <p role="alert">商品設計情報を取得できませんでした。</p>;
    }

    return (
      <>
        <section className="ui-detail-section">
          <h2 className="ui-detail-section__title">カテゴリ</h2>
          <dl className="ui-detail-definition-list">
            <dt>カテゴリパス</dt>
            <dd>
              {productBlueprint.productBlueprintCategoryPath.length > 0
                ? productBlueprint.productBlueprintCategoryPath.join(" / ")
                : "-"}
            </dd>

            <dt>カテゴリ項目</dt>
            <dd style={{ whiteSpace: "pre-wrap" }}>
              {formatCategoryFields(productBlueprint.categoryFields)}
            </dd>
          </dl>
        </section>

        <section className="ui-detail-section">
          <h2 className="ui-detail-section__title">モデル</h2>
          {productBlueprint.modelRefs.length > 0 ? (
            <dl className="ui-detail-definition-list">
              {productBlueprint.modelRefs
                .slice()
                .sort((a, b) => a.displayOrder - b.displayOrder)
                .map((modelRef) => (
                  <div key={modelRef.modelId}>
                    <dd>{formatModelMeta(modelRef)}</dd>
                  </div>
                ))}
            </dl>
          ) : (
            <p>モデル情報はありません。</p>
          )}
        </section>
      </>
    );
  };

  return (
    <Page>
      <PageHeader
        title={productBlueprint?.productName || "商品設計詳細"}
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

      {company && productBlueprint ? (
        <DetailPageBody
          main={renderMain()}
          aside={
            <div className="product-blueprint-detail-page__aside">
              <section className="ui-detail-section">
                <dl className="ui-detail-definition-list product-blueprint-detail-page__definition-list">
                  <dt>企業名</dt>
                  <dd>{company.name || "-"}</dd>

                  <dt>ブランド</dt>
                  <dd>{productBlueprint.brandName || "-"}</dd>

                  <dt>担当者</dt>
                  <dd>{productBlueprint.assigneeName || "-"}</dd>

                  <dt>印刷状態</dt>
                  <dd>{formatPrinted(productBlueprint.printed)}</dd>

                  <dt>タグ種別</dt>
                  <dd>{productBlueprint.productIdTagType || "-"}</dd>

                  <dt>作成日時</dt>
                  <dd>{formatDateTime(productBlueprint.createdAt)}</dd>

                  <dt>最終更新日時</dt>
                  <dd>{formatDateTime(productBlueprint.updatedAt)}</dd>
                </dl>
              </section>
            </div>
          }
        />
      ) : (
        renderMain()
      )}
    </Page>
  );
}