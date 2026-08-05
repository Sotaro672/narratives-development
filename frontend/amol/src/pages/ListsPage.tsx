// frontend/amol/src/pages/ListsPage.tsx

import {
  useNavigate,
} from "react-router-dom";

import Layout from "../components/layout/Layout";

import ListGrid from "../features/list/presentation/components/ListGrid";
import ListPagination from "../features/list/presentation/components/ListPagination";

import {
  useListsPage,
} from "../features/list/presentation/hooks/useListsPage";

import "../features/list/presentation/styles/lists-page.css";

export default function ListsPage() {
  const navigate = useNavigate();

  const {
    items,
    page,
    totalPages,
    isLoading,
    canGoPrev,
    canGoNext,
    goPrev,
    goNext,
  } = useListsPage();

  function handleCartButtonClick() {
    navigate("/cart");
  }

  function handleOpenItem(
    listId: string,
  ) {
    const normalizedListId =
      listId.trim();

    if (!normalizedListId) {
      return;
    }

    navigate(
      `/lists/${encodeURIComponent(
        normalizedListId,
      )}`,
    );
  }

  return (
    <Layout
      title="AMOL"
      mode="mypage"
      showCartButton
      cartButtonLabel="カート"
      onCartButtonClick={
        handleCartButtonClick
      }
    >
      <section className="content-page-section rooms-page-section-root lists-page-section-root">
        {!isLoading &&
        items.length > 0 ? (
          <ListGrid
            items={items}
            onOpenItem={
              handleOpenItem
            }
          />
        ) : null}

        {!isLoading &&
        totalPages > 1 ? (
          <ListPagination
            page={page}
            totalPages={
              totalPages
            }
            canGoPrev={
              canGoPrev
            }
            canGoNext={
              canGoNext
            }
            onPrev={goPrev}
            onNext={goNext}
          />
        ) : null}
      </section>
    </Layout>
  );
}