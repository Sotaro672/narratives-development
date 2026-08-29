// frontend/amol/src/features/contents/hooks/useContentsPage.ts

import { useEffect, useMemo, useState } from "react";
import { getAuth } from "firebase/auth";
import { useNavigate, useSearchParams } from "react-router-dom";

import type { MediaGalleryItem } from "../../../components/ui/MediaGallery";
import { useMobilePortrait } from "../../../components/hooks/useMobilePortrait";
import { fetchPayoutAccount } from "../../payout/api/payoutApi";
import { hasMyResaleListingByProductId } from "../../resale/api/resaleApi";
import type {
  ContentsMetadata,
  ContentsSearchParams,
} from "../../shared/types/contents";
import { useTokenCommentCard } from "../../token-commnet/hooks/useTokenCommentCard";
import { fetchContentsMetadata } from "../api/contentsApi";

function buildContentsSearchParams(
  searchParams: URLSearchParams,
): ContentsSearchParams {
  return {
    assetId: searchParams.get("assetId") || "",
    productId: searchParams.get("productId") || "",
    brandId: searchParams.get("brandId") || "",
    brandName: searchParams.get("brandName") || "",
    productName: searchParams.get("productName") || "",
    productBlueprintId: searchParams.get("productBlueprintId") || "",
    tokenBlueprintId: searchParams.get("tokenBlueprintId") || "",
    metadataUri: searchParams.get("metadataUri") || "",
  };
}

export function useContentsPage() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const isMobilePortrait = useMobilePortrait();

  const contents = useMemo(
    () => buildContentsSearchParams(searchParams),
    [searchParams],
  );

  const commentCard = useTokenCommentCard({
    tokenBlueprintId: contents.tokenBlueprintId,
  });

  const [metadata, setMetadata] = useState<ContentsMetadata | null>(null);
  const [activeFileIndex, setActiveFileIndex] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [resaleChecking, setResaleChecking] = useState(false);
  const [isResaleListed, setIsResaleListed] = useState(false);

  const handleProductNameClick = () => {
    if (!contents.productId) return;

    navigate(
      `/scan/result?productId=${encodeURIComponent(contents.productId)}`,
    );
  };

  const handleBrandNameClick = () => {
    if (!contents.brandId) return;

    navigate(
      `/brands/${encodeURIComponent(contents.brandId)}`,
    );
  };

  useEffect(() => {
    if (!contents.metadataUri) {
      setMetadata(null);
      setActiveFileIndex(0);
      setError("");
      setLoading(false);
      return;
    }

    let isMounted = true;

    const load = async () => {
      setLoading(true);
      setError("");

      try {
        const result = await fetchContentsMetadata(contents.metadataUri);

        if (!isMounted) return;

        setMetadata(result);
        setActiveFileIndex(0);
      } catch (err) {
        if (!isMounted) return;

        setMetadata(null);
        setActiveFileIndex(0);
        setError(
          err instanceof Error
            ? err.message
            : "トークンコンテンツの取得に失敗しました。",
        );
      } finally {
        if (isMounted) setLoading(false);
      }
    };

    void load();

    return () => {
      isMounted = false;
    };
  }, [contents.metadataUri]);

  useEffect(() => {
    if (!contents.productId) {
      setIsResaleListed(false);
      setResaleChecking(false);
      return;
    }

    let isMounted = true;

    const load = async () => {
      setResaleChecking(true);
      setIsResaleListed(false);

      try {
        const listed = await hasMyResaleListingByProductId(
          contents.productId,
        );

        if (!isMounted) return;

        setIsResaleListed(listed);
      } catch (err) {
        if (!isMounted) return;

        console.error(
          "failed to check existing resale listing:",
          err,
        );

        setIsResaleListed(false);
      } finally {
        if (isMounted) {
          setResaleChecking(false);
        }
      }
    };

    void load();

    return () => {
      isMounted = false;
    };
  }, [contents.productId]);

  const tokenName = metadata?.name ?? "";
  const tokenIconUrl = metadata?.image ?? "";
  const tokenDescription = metadata?.description ?? "";
  const pageTitle = tokenName || "トークン詳細";

  const mediaItems = useMemo<MediaGalleryItem[]>(() => {
    if (!metadata) return [];

    return metadata.files
      .filter((file) => file.uri !== metadata.image)
      .map((file, index) => ({
        id: `${index}-${file.uri}`,
        url: file.uri,
        type: file.type,
      }));
  }, [metadata]);

  useEffect(() => {
    if (activeFileIndex >= mediaItems.length) {
      setActiveFileIndex(0);
    }
  }, [activeFileIndex, mediaItems.length]);

  const hasMediaItems = mediaItems.length > 0;
  const resaleButtonDisabled = resaleChecking || isResaleListed;
  const resaleButtonLabel = resaleChecking
    ? "確認中"
    : isResaleListed
      ? "出品済"
      : "出品";

  const handlePrevFile = () => {
    if (!hasMediaItems) return;

    setActiveFileIndex((current) =>
      current === 0 ? mediaItems.length - 1 : current - 1,
    );
  };

  const handleNextFile = () => {
    if (!hasMediaItems) return;

    setActiveFileIndex((current) =>
      current === mediaItems.length - 1 ? 0 : current + 1,
    );
  };

  const handleOpenResalePage = async () => {
    if (
      resaleButtonDisabled ||
      !contents.productId ||
      !contents.tokenBlueprintId
    ) {
      return;
    }

    const auth = getAuth();
    const currentUser = auth.currentUser;

    if (!currentUser) {
      navigate("/signin", { replace: true });
      return;
    }

    const resaleCreateState = {
      assetId: contents.assetId,
      productId: contents.productId,
      brandId: contents.brandId,
      brandName: contents.brandName,
      productName: contents.productName,
      productBlueprintId: contents.productBlueprintId,
      tokenBlueprintId: contents.tokenBlueprintId,
      tokenName,
      tokenIconUrl,
      tokenDescription,
    };

    try {
      setError("");

      const idToken = await currentUser.getIdToken(true);
      const payoutAccount = await fetchPayoutAccount({
        idToken,
      });

      if (!payoutAccount) {
        navigate("/settings/payout-account", {
          state: {
            returnAfterRegistration: {
              pathname: "/resale",
              state: resaleCreateState,
            },
          },
        });
        return;
      }

      navigate("/resale", {
        state: resaleCreateState,
      });
    } catch (err) {
      console.error(
        "failed to check payout account before resale:",
        err,
      );

      setError(
        err instanceof Error
          ? err.message
          : "売上受取口座の確認に失敗しました。",
      );
    }
  };

  return {
    contents,
    commentCard,
    metadata,
    mediaItems,
    activeFileIndex,
    setActiveFileIndex,
    loading,
    error,
    tokenName,
    tokenIconUrl,
    tokenDescription,
    pageTitle,
    hasMediaItems,
    isMobilePortrait,
    resaleChecking,
    isResaleListed,
    resaleButtonDisabled,
    resaleButtonLabel,
    handleProductNameClick,
    handleBrandNameClick,
    handlePrevFile,
    handleNextFile,
    handleOpenResalePage,
  };
}