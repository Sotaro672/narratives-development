// frontend/amol/src/pages/BrandPage.tsx

import { useEffect, useState } from "react";
import {
  useNavigate,
  useParams,
} from "react-router-dom";

import Layout from "../components/layout/Layout";

import { getMyAvatar } from "../features/avatar/api/avatarApi";
import BrandContent from "../features/brand/presentation/components/BrandContent";
import BrandPageError from "../features/brand/presentation/components/BrandPageError";
import BrandPageLoading from "../features/brand/presentation/components/BrandPageLoading";
import { useBrandPage } from "../features/brand/presentation/hooks/useBrandPage";
import ReportModal from "../features/report/components/ReportModal";
import { useReport } from "../features/report/hooks/useReport";
import { useAuthState } from "../features/shared/hooks/useAuthState";

import "../styles/brand_page.css";

type BrandPageRouteParams = {
  brandId?: string;
};

export default function BrandPage() {
  const {
    brandId: routeBrandId,
  } = useParams<BrandPageRouteParams>();

  const navigate = useNavigate();
  const { authResolved, isLoggedIn } = useAuthState();
  const [currentAvatarId, setCurrentAvatarId] = useState("");

  const report = useReport();

  const brandId =
    routeBrandId?.trim() ?? "";

  const {
    brand,
    listItems,
    loading,
    error,
    reload,
  } = useBrandPage(brandId);

  useEffect(() => {
    let cancelled = false;

    async function loadCurrentAvatar() {
      if (!authResolved || !isLoggedIn) {
        setCurrentAvatarId("");
        return;
      }

      try {
        const avatar = await getMyAvatar();

        if (!cancelled) {
          setCurrentAvatarId(
            avatar?.avatarId?.trim() ?? "",
          );
        }
      } catch {
        if (!cancelled) {
          setCurrentAvatarId("");
        }
      }
    }

    void loadCurrentAvatar();

    return () => {
      cancelled = true;
    };
  }, [authResolved, isLoggedIn]);

  const title =
    brand?.brandName?.trim() ||
    "ブランド";

  const canReportBrand =
    authResolved &&
    isLoggedIn &&
    Boolean(currentAvatarId) &&
    Boolean(brandId) &&
    !report.submitting;

  const handleBack = () => {
    navigate(-1);
  };

  const handleOpenBrandReport = () => {
    if (!canReportBrand) {
      return;
    }

    report.openBrandReport({
      brandId,
    });
  };

  return (
    <>
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
            canReport={canReportBrand}
            onReport={handleOpenBrandReport}
          />
        ) : null}
      </Layout>

      <ReportModal
        open={report.isOpen}
        targetType={report.target?.type}
        reason={report.reason}
        detail={report.detail}
        submitting={report.submitting}
        error={report.error}
        result={report.result}
        canSubmit={report.canSubmit}
        onReasonChange={report.setReason}
        onDetailChange={report.setDetail}
        onSubmit={report.submit}
        onClose={report.close}
      />
    </>
  );
}