// frontend/amol/src/features/catalog/presentation/hooks/useCatalogPage.ts

import {
  useEffect,
  useMemo,
  useState,
  type TouchEvent,
} from "react";
import { useNavigate, useParams } from "react-router-dom";

import { addSelectedCatalogItemToCart } from "../../application/catalogCartUsecase";
import { loadCatalogPage } from "../../application/catalogPageLoader";
import { createCatalogPageViewModel } from "../../application/catalogPageViewModelFactory";
import { resolveCatalogSwipeDirection } from "../../application/catalogSwipeUsecase";
import { SWIPE_THRESHOLD_PX } from "../../constants";
import { getApiBaseUrl } from "../../infrastructure/apiBaseUrlProvider";
import type {
  CatalogProductBlueprintReviewPage,
  CatalogResponse,
} from "../../types";
import { useMobilePortrait } from "./useMobilePortrait";

export function useCatalogPage() {
  const navigate = useNavigate();
  const { listId } = useParams();

  const [catalog, setCatalog] = useState<CatalogResponse | null>(null);
  const [reviews, setReviews] =
    useState<CatalogProductBlueprintReviewPage | null>(null);

  const [isLoadingCatalog, setIsLoadingCatalog] = useState(true);
  const [isLoadingReviews, setIsLoadingReviews] = useState(false);
  const [isAddingToCart, setIsAddingToCart] = useState(false);

  const [errorMessage, setErrorMessage] = useState("");
  const [reviewErrorMessage, setReviewErrorMessage] = useState("");
  const [cartMessage, setCartMessage] = useState("");
  const [cartErrorMessage, setCartErrorMessage] = useState("");

  const [activeImageIndex, setActiveImageIndex] = useState(0);
  const [touchStartX, setTouchStartX] = useState<number | null>(null);
  const [touchStartY, setTouchStartY] = useState<number | null>(null);

  const [selectedColorKey, setSelectedColorKey] = useState("");
  const [selectedSize, setSelectedSize] = useState("");

  const apiBaseUrl = useMemo(() => getApiBaseUrl(), []);
  const isMobilePortrait = useMobilePortrait();

  const viewModel = useMemo(() => {
    return createCatalogPageViewModel({
      catalog,
      reviews,
      selectedColorKey,
      selectedSize,
      activeImageIndex,
      isAddingToCart,
    });
  }, [
    catalog,
    reviews,
    selectedColorKey,
    selectedSize,
    activeImageIndex,
    isAddingToCart,
  ]);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      if (!listId) {
        setCatalog(null);
        setReviews(null);
        setErrorMessage("listIdが見つかりません。");
        setReviewErrorMessage("");
        setIsLoadingCatalog(false);
        setIsLoadingReviews(false);
        return;
      }

      setIsLoadingCatalog(true);
      setIsLoadingReviews(true);

      setErrorMessage("");
      setReviewErrorMessage("");
      setCartMessage("");
      setCartErrorMessage("");

      setCatalog(null);
      setReviews(null);

      setActiveImageIndex(0);
      setSelectedColorKey("");
      setSelectedSize("");
      setTouchStartX(null);
      setTouchStartY(null);

      try {
        const result = await loadCatalogPage({
          apiBaseUrl,
          listId,
        });

        if (cancelled) {
          return;
        }

        setCatalog(result.catalog);
        setReviews(result.reviews);
        setReviewErrorMessage(result.reviewErrorMessage);
      } catch (error) {
        if (cancelled) {
          return;
        }

        setCatalog(null);
        setReviews(null);
        setReviewErrorMessage("");
        setErrorMessage(
          error instanceof Error
            ? error.message
            : "カタログ詳細の取得中にエラーが発生しました。",
        );
      } finally {
        if (!cancelled) {
          setIsLoadingCatalog(false);
          setIsLoadingReviews(false);
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [apiBaseUrl, listId]);

  useEffect(() => {
    if (viewModel.colorOptions.length === 1 && !selectedColorKey) {
      setSelectedColorKey(viewModel.colorOptions[0].key);
      return;
    }

    if (
      selectedColorKey &&
      !viewModel.colorOptions.some((option) => option.key === selectedColorKey)
    ) {
      setSelectedColorKey("");
    }
  }, [selectedColorKey, viewModel.colorOptions]);

  useEffect(() => {
    if (viewModel.sizeOptions.length === 1 && !selectedSize) {
      setSelectedSize(viewModel.sizeOptions[0]);
      return;
    }

    if (selectedSize && !viewModel.sizeOptions.includes(selectedSize)) {
      setSelectedSize("");
    }
  }, [selectedSize, viewModel.sizeOptions]);

  useEffect(() => {
    if (activeImageIndex > viewModel.catalogImages.length - 1) {
      setActiveImageIndex(0);
    }
  }, [activeImageIndex, viewModel.catalogImages.length]);

  function handlePrevImage() {
    if (viewModel.catalogImages.length === 0) {
      return;
    }

    setActiveImageIndex((current) =>
      current === 0 ? viewModel.catalogImages.length - 1 : current - 1,
    );
  }

  function handleNextImage() {
    if (viewModel.catalogImages.length === 0) {
      return;
    }

    setActiveImageIndex((current) =>
      current === viewModel.catalogImages.length - 1 ? 0 : current + 1,
    );
  }

  function handleImageTouchStart(event: TouchEvent<HTMLDivElement>) {
    if (!isMobilePortrait || viewModel.catalogImages.length <= 1) {
      return;
    }

    const touch = event.touches[0];

    if (!touch) {
      return;
    }

    setTouchStartX(touch.clientX);
    setTouchStartY(touch.clientY);
  }

  function handleImageTouchEnd(event: TouchEvent<HTMLDivElement>) {
    if (
      !isMobilePortrait ||
      viewModel.catalogImages.length <= 1 ||
      touchStartX === null ||
      touchStartY === null
    ) {
      setTouchStartX(null);
      setTouchStartY(null);
      return;
    }

    const touch = event.changedTouches[0];

    setTouchStartX(null);
    setTouchStartY(null);

    if (!touch) {
      return;
    }

    const direction = resolveCatalogSwipeDirection({
      startX: touchStartX,
      startY: touchStartY,
      endX: touch.clientX,
      endY: touch.clientY,
      thresholdPx: SWIPE_THRESHOLD_PX,
    });

    if (direction === "next") {
      handleNextImage();
      return;
    }

    if (direction === "prev") {
      handlePrevImage();
    }
  }

  function handleSelectColor(colorKey: string) {
    setSelectedColorKey(colorKey);
    setSelectedSize("");
    setCartMessage("");
    setCartErrorMessage("");
  }

  function handleSelectSize(size: string) {
    setSelectedSize(size);
    setCartMessage("");
    setCartErrorMessage("");
  }

  async function handleAddToCart() {
    setIsAddingToCart(true);
    setCartMessage("");
    setCartErrorMessage("");

    try {
      await addSelectedCatalogItemToCart({
        apiBaseUrl,
        catalog,
        selectedModel: viewModel.selectedModel,
        hasSelectedModelStock: viewModel.hasSelectedModelStock,
        isAlcoholCatalog: viewModel.isAlcoholCatalog,
      });

      navigate("/cart");
    } catch (error) {
      setCartErrorMessage(
        error instanceof Error
          ? error.message
          : "カートへの追加中にエラーが発生しました。",
      );
    } finally {
      setIsAddingToCart(false);
    }
  }

  function handleCartButtonClick() {
    navigate("/cart");
  }

  return {
    catalog,
    isLoadingCatalog,
    isLoadingReviews,
    isAddingToCart,
    errorMessage,
    reviewErrorMessage,
    cartMessage,
    cartErrorMessage,

    selectedColorKey,
    selectedSize,
    activeImageIndex,
    isMobilePortrait,

    ...viewModel,

    setActiveImageIndex,
    handlePrevImage,
    handleNextImage,
    handleImageTouchStart,
    handleImageTouchEnd,
    handleSelectColor,
    handleSelectSize,
    handleAddToCart,
    handleCartButtonClick,
  };
}