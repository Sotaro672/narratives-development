// frontend/amol/src/pages/TradeChatRedirectPage.tsx

import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import Layout from "../components/layout/Layout";
import { fetchTradeByOrderItem } from "../features/trade/infrastructure/tradeApi";

import "../styles/page-layout.css";

type TradeChatRedirectRouteParams = {
  orderId?: string;
  itemIndex?: string;
};

function getErrorMessage(
  caught: unknown,
  fallbackMessage: string,
): string {
  if (caught instanceof Error && caught.message) {
    return caught.message;
  }

  return fallbackMessage;
}

export default function TradeChatRedirectPage() {
  const navigate = useNavigate();
  const { orderId, itemIndex } =
    useParams<TradeChatRedirectRouteParams>();

  const [error, setError] = useState("");

  useEffect(() => {
    let cancelled = false;

    async function resolveTrade(): Promise<void> {
      if (!orderId) {
        setError("注文IDが見つかりません。");
        return;
      }

      if (itemIndex === undefined) {
        setError("商品番号が見つかりません。");
        return;
      }

      const parsedItemIndex = Number(itemIndex);

      if (
        !Number.isInteger(parsedItemIndex) ||
        parsedItemIndex < 0
      ) {
        setError("商品番号が正しくありません。");
        return;
      }

      setError("");

      try {
        const trade = await fetchTradeByOrderItem({
          orderId,
          orderItemIndex: parsedItemIndex,
          limit: 1,
        });

        if (cancelled) {
          return;
        }

        if (!trade.id) {
          setError("取引が見つかりません。");
          return;
        }

        navigate(
          `/chats/trades/${encodeURIComponent(trade.id)}`,
          {
            replace: true,
            state: {
              trade,
            },
          },
        );
      } catch (caught) {
        if (cancelled) {
          return;
        }

        setError(
          getErrorMessage(
            caught,
            "取引を開けませんでした。",
          ),
        );
      }
    }

    void resolveTrade();

    return () => {
      cancelled = true;
    };
  }, [
    itemIndex,
    navigate,
    orderId,
  ]);

  return (
    <Layout
      title="取引"
      mode="mypage"
      showFooter={false}
    >
      <section className="page-section content-page-section">
        {error ? (
          <div role="alert">
            {error}
          </div>
        ) : (
          <div>
            取引を読み込み中...
          </div>
        )}
      </section>
    </Layout>
  );
}