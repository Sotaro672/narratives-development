// frontend/amol/src/pages/BrandPage.tsx

import {
  useNavigate,
  useParams,
} from "react-router-dom";

import Layout from "../components/layout/Layout";

import BrandContent from "../features/brand/presentation/components/BrandContent";
import BrandPageError from "../features/brand/presentation/components/BrandPageError";
import BrandPageLoading from "../features/brand/presentation/components/BrandPageLoading";
import { useBrandPage } from "../features/brand/presentation/hooks/useBrandPage";

import "../styles/brand_page.css";

type BrandPageRouteParams = {
  brandId?: string;
};

export default function BrandPage() {
  const {
    brandId: routeBrandId,
  } = useParams<BrandPageRouteParams>();

  const navigate = useNavigate();

  const brandId =
    routeBrandId?.trim() ?? "";

  const {
    brand,
    listItems,
    loading,
    error,
    reload,
  } = useBrandPage(brandId);

  const title =
    brand?.brandName?.trim() ||
    "ブランド";

  const handleBack = () => {
    navigate(-1);
  };

  return (
    <Layout
      title={title}
      titleClickable={false}
      mode="landing"
      showHeader
      showBackButton
      backTo="/lists"
      onBackButtonClick={handleBack}
      showFooter={false}
      hideHamburgerMenu={false}
      hideSettingsButton
      mainClassName="brand-page-main"
    >
      {loading ? (
        <BrandPageLoading />
      ) : null}

      {!loading && (error || !brand) ? (
        <BrandPageError
          error={
            error ||
            "brand data is empty"
          }
          onBack={handleBack}
          onRetry={() => {
            void reload();
          }}
        />
      ) : null}

      {!loading && brand ? (
        <BrandContent
          brand={brand}
          listItems={listItems}
        />
      ) : null}
    </Layout>
  );
}