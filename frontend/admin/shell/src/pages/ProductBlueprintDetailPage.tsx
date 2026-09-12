// frontend/admin/shell/src/pages/ProductBlueprintDetailPage.tsx

import { useNavigate, useParams } from "react-router-dom";

import ProductBlueprintDetailAside from "../features/productBlueprint/presentation/components/ProductBlueprintDetailAside";
import ProductBlueprintModelList from "../features/productBlueprint/presentation/components/ProductBlueprintModelList";
import ProductBlueprintReviewTable from "../features/productBlueprint/presentation/components/ProductBlueprintReviewTable";
import ProductBlueprintSummary from "../features/productBlueprint/presentation/components/ProductBlueprintSummary";
import { useProductBlueprintDetail } from "../features/productBlueprint/presentation/hooks/useProductBlueprintDetail";
import Page, { DetailPageBody, PageHeader } from "../shared/ui/Page/Page";

import "./ProductBlueprintDetailPage.css";

export default function ProductBlueprintDetailPage() {
  const navigate = useNavigate();
  const { companyId = "", productBlueprintId = "" } = useParams<{
    companyId?: string;
    productBlueprintId?: string;
  }>();

  const { detail, loading, error, reload } =
    useProductBlueprintDetail(companyId, productBlueprintId);

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
        <ProductBlueprintSummary productBlueprint={productBlueprint} />
        <ProductBlueprintModelList modelRefs={productBlueprint.modelRefs} />

        <section className="ui-detail-section">
          <h2 className="ui-detail-section__title">レビュー</h2>
          <ProductBlueprintReviewTable
            companyId={companyId}
            productBlueprintId={productBlueprintId}
            perPage={20}
          />
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

      {company && productBlueprint ? (
        <DetailPageBody
          main={renderMain()}
          aside={
            <ProductBlueprintDetailAside
              company={company}
              productBlueprint={productBlueprint}
              onOpenBrand={() =>
                navigate(
                  `/contracts/${encodeURIComponent(companyId)}/brands/${encodeURIComponent(productBlueprint.brandId)}`,
                )
              }
              onOpenReport={(reportCaseId) =>
                navigate(`/reports/${encodeURIComponent(reportCaseId)}`)
              }
            />
          }
        />
      ) : (
        renderMain()
      )}
    </Page>
  );
}