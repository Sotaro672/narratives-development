//frontend\amol\src\features\catalog\hooks\useCatalogPage.ts
import {
  useEffect,
  useMemo,
  useState,
  type TouchEvent,
} from "react";
import { useNavigate, useParams } from "react-router-dom";

import {
  addCatalogItemToCart,
  fetchCatalogDetail,
  fetchCatalogReviews,
  fetchCurrentAvatarId,
  getApiBaseUrl,
} from "../api/catalogApi";
import { SWIPE_THRESHOLD_PX } from "../constants";
import type {
  CatalogListImage,
  CatalogProductBlueprintReviewPage,
  CatalogResponse,
  MeasurementTableRow,
  ModelColorOption,
} from "../types";
import { getAvailableStock, getModelColorKey } from "../utils/model";
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

  useEffect(() => {
    let cancelled = false;

    async function loadCatalog() {
      if (!listId) {
        setCatalog(null);
        setReviews(null);
        setErrorMessage("listIdが見つかりません。");
        setIsLoadingCatalog(false);
        return;
      }

      setIsLoadingCatalog(true);
      setIsLoadingReviews(false);
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
        const catalogDetail = await fetchCatalogDetail({
          apiBaseUrl,
          listId,
        });

        if (cancelled) {
          return;
        }

        setCatalog(catalogDetail);
        setIsLoadingReviews(true);

        try {
          const reviewPage = await fetchCatalogReviews(
            apiBaseUrl,
            catalogDetail.productBlueprint.id,
          );

          if (!cancelled) {
            setReviews(reviewPage);
          }
        } catch (reviewError) {
          if (!cancelled) {
            setReviews(null);
            setReviewErrorMessage(
              reviewError instanceof Error
                ? reviewError.message
                : "レビュー一覧の取得中にエラーが発生しました。",
            );
          }
        } finally {
          if (!cancelled) {
            setIsLoadingReviews(false);
          }
        }
      } catch (error) {
        if (cancelled) {
          return;
        }

        setCatalog(null);
        setReviews(null);
        setErrorMessage(
          error instanceof Error
            ? error.message
            : "カタログ詳細の取得中にエラーが発生しました。",
        );
      } finally {
        if (!cancelled) {
          setIsLoadingCatalog(false);
        }
      }
    }

    loadCatalog();

    return () => {
      cancelled = true;
    };
  }, [apiBaseUrl, listId]);

  const catalogImages = useMemo<CatalogListImage[]>(() => {
    const uniqueImages = new Map<string, CatalogListImage>();

    for (const image of catalog?.listImages ?? []) {
      if (!image.url) {
        continue;
      }

      uniqueImages.set(image.id, image);
    }

    return Array.from(uniqueImages.values()).sort((a, b) => {
      if (a.displayOrder !== b.displayOrder) {
        return a.displayOrder - b.displayOrder;
      }

      return a.id.localeCompare(b.id);
    });
  }, [catalog?.listImages]);

  const measurementRows = useMemo<MeasurementTableRow[]>(() => {
    const rows = new Map<string, MeasurementTableRow>();

    for (const model of catalog?.modelVariations ?? []) {
      const size = model.size?.trim() || "-";

      if (rows.has(size)) {
        continue;
      }

      rows.set(size, {
        id: model.id,
        size,
        measurements: model.measurements ?? {},
      });
    }

    return Array.from(rows.values());
  }, [catalog?.modelVariations]);

  const measurementKeys = useMemo(() => {
    const keys = new Set<string>();

    for (const row of measurementRows) {
      for (const key of Object.keys(row.measurements ?? {})) {
        keys.add(key);
      }
    }

    return Array.from(keys);
  }, [measurementRows]);

  const colorOptions = useMemo<ModelColorOption[]>(() => {
    const options = new Map<string, ModelColorOption>();

    for (const model of catalog?.modelVariations ?? []) {
      const key = getModelColorKey(model);

      if (options.has(key)) {
        continue;
      }

      options.set(key, {
        key,
        colorName: model.colorName?.trim() || "-",
        colorRGB: Number.isFinite(model.colorRGB) ? model.colorRGB : 0,
      });
    }

    return Array.from(options.values());
  }, [catalog?.modelVariations]);

  const sizeOptions = useMemo(() => {
    const sizes = new Set<string>();

    for (const model of catalog?.modelVariations ?? []) {
      if (selectedColorKey && getModelColorKey(model) !== selectedColorKey) {
        continue;
      }

      sizes.add(model.size?.trim() || "-");
    }

    return Array.from(sizes);
  }, [catalog?.modelVariations, selectedColorKey]);

  const matchedModels = useMemo(() => {
    if (!catalog || !selectedColorKey || !selectedSize) {
      return [];
    }

    return catalog.modelVariations.filter((model) => {
      const modelSize = model.size?.trim() || "-";

      return (
        getModelColorKey(model) === selectedColorKey &&
        modelSize === selectedSize
      );
    });
  }, [catalog, selectedColorKey, selectedSize]);

  const selectedModel = matchedModels.length === 1 ? matchedModels[0] : null;

  useEffect(() => {
    if (colorOptions.length === 1 && !selectedColorKey) {
      setSelectedColorKey(colorOptions[0].key);
      return;
    }

    if (
      selectedColorKey &&
      !colorOptions.some((option) => option.key === selectedColorKey)
    ) {
      setSelectedColorKey("");
    }
  }, [colorOptions, selectedColorKey]);

  useEffect(() => {
    if (sizeOptions.length === 1 && !selectedSize) {
      setSelectedSize(sizeOptions[0]);
      return;
    }

    if (selectedSize && !sizeOptions.includes(selectedSize)) {
      setSelectedSize("");
    }
  }, [selectedSize, sizeOptions]);

  useEffect(() => {
    if (activeImageIndex > catalogImages.length - 1) {
      setActiveImageIndex(0);
    }
  }, [activeImageIndex, catalogImages.length]);

  const activeImage = catalogImages[activeImageIndex];
  const hasMultipleImages = catalogImages.length > 1;
  const firstPrice = catalog?.list.prices?.[0];
  const reviewSummary = catalog?.productReviewSummary;
  const selectedModelPrice = selectedModel
    ? catalog?.list.prices.find((price) => price.modelId === selectedModel.id)
    : undefined;
  const selectedModelStock = selectedModel
    ? getAvailableStock(catalog?.inventory, selectedModel.id)
    : undefined;
  const hasSelectedModelStock =
    typeof selectedModelStock === "number" && selectedModelStock > 0;
  const shouldShowMeasurementTable =
    measurementRows.length > 0 && measurementKeys.length > 0;
  const reviewItems = reviews?.items ?? [];
  const canAddToCart =
    Boolean(catalog && selectedModel) && hasSelectedModelStock && !isAddingToCart;

  function handlePrevImage() {
    if (catalogImages.length === 0) {
      return;
    }

    setActiveImageIndex((current) =>
      current === 0 ? catalogImages.length - 1 : current - 1,
    );
  }

  function handleNextImage() {
    if (catalogImages.length === 0) {
      return;
    }

    setActiveImageIndex((current) =>
      current === catalogImages.length - 1 ? 0 : current + 1,
    );
  }

  function handleImageTouchStart(event: TouchEvent<HTMLDivElement>) {
    if (!isMobilePortrait || catalogImages.length <= 1) {
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
      catalogImages.length <= 1 ||
      touchStartX === null ||
      touchStartY === null
    ) {
      setTouchStartX(null);
      setTouchStartY(null);
      return;
    }

    const touch = event.changedTouches[0];

    if (!touch) {
      setTouchStartX(null);
      setTouchStartY(null);
      return;
    }

    const diffX = touch.clientX - touchStartX;
    const diffY = touch.clientY - touchStartY;

    setTouchStartX(null);
    setTouchStartY(null);

    if (Math.abs(diffX) < SWIPE_THRESHOLD_PX) {
      return;
    }

    if (Math.abs(diffY) > Math.abs(diffX)) {
      return;
    }

    if (diffX < 0) {
      handleNextImage();
      return;
    }

    handlePrevImage();
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
    if (!catalog || !selectedModel) {
      setCartMessage("");
      setCartErrorMessage("カラーとサイズを選択してください。");
      return;
    }

    if (!hasSelectedModelStock) {
      setCartMessage("");
      setCartErrorMessage("選択した商品の在庫がありません。");
      return;
    }

    setIsAddingToCart(true);
    setCartMessage("");
    setCartErrorMessage("");

    try {
      const avatarId = await fetchCurrentAvatarId(apiBaseUrl);

      await addCatalogItemToCart({
        apiBaseUrl,
        avatarId,
        catalog,
        selectedModel,
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
    activeImage,
    activeImageIndex,
    catalogImages,
    hasMultipleImages,
    firstPrice,
    reviewSummary,
    reviewItems,
    measurementRows,
    measurementKeys,
    shouldShowMeasurementTable,
    colorOptions,
    sizeOptions,
    selectedColorKey,
    selectedSize,
    selectedModel,
    selectedModelPrice,
    selectedModelStock,
    hasSelectedModelStock,
    canAddToCart,
    isMobilePortrait,
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