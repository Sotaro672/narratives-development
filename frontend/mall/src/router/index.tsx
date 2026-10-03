// frontend/mall/src/router/index.tsx

import { useEffect, useState } from "react";
import { createBrowserRouter, Navigate, Outlet } from "react-router-dom";
import { onAuthStateChanged, type User } from "firebase/auth";

import { auth } from "../lib/firebase";

import LandingPage from "../pages/LandingPage";
import SignInPage from "../pages/SignInPage";
import SignInSelectPage from "../pages/SignInSelectPage";
import SignUpPage from "../pages/SignUpPage";
import VerificationSentPage from "../pages/VerificationSentPage";
import PasswordResetPage from "../pages/PasswordResetPage";
import AvatarPage from "../pages/AvatarPage";
import AvatarReviewPage from "../pages/AvatarReviewPage";
import AvatarReviewCreatePage from "../pages/AvatarReviewCreatePage";
import EmailPage from "../pages/EmailPage";
import PasswordPage from "../pages/PasswordPage";
import IdentityVerificationPage from "../pages/IdentityVerificationPage";
import PaymentMethodPage from "../pages/PaymentMethodPage";
import PayoutAccountPage from "../pages/PayoutAccountPage";
import PayoutBankSelectPage from "../pages/PayoutBankSelectPage";
import PayoutBranchSelectPage from "../pages/PayoutBranchSelectPage";
import PayoutBankAccountPage from "../pages/PayoutBankAccountPage";
import PayoutAccountConfirmPage from "../pages/PayoutAccountConfirmPage";
import PayoutAccountCompletePage from "../pages/PayoutAccountCompletePage";
import ShippingAddressPage from "../pages/ShippingAddressPage";
import AuthActionPage from "../pages/AuthActionPage";
import ListsPage from "../pages/ListsPage";
import MarketPage from "../pages/MarketPage";
import MarketDetailPage from "../pages/MarketDetailPage";
import LikesPage from "../pages/LikesPage";
import CartPage from "../pages/CartPage";
import CatalogPage from "../pages/CatalogPage";
import BrandPage from "../pages/BrandPage";
import PaymentPage from "../pages/PaymentPage";
import OrderConfirmedPage from "../pages/OrderConfirmedPage";
import OrderDetail from "../pages/OrderDetail";
import ReturnRequestPage from "../pages/ReturnRequestPage";
import ScanPage from "../pages/ScanPage";
import ScanResultPage from "../pages/ScanResultPage";
import ProductDescriptionInquiryPage from "../pages/ProductDescriptionInquiryPage";
import IssueReportPage from "../pages/IssueReportPage";
import ChatWorkspacePage from "../pages/ChatWorkspacePage";
import ChatDetailPage from "../pages/ChatDetailPage";
import TradeChatRedirectPage from "../pages/TradeChatRedirectPage";
import TradeReturnConsultationPage from "../pages/TradeReturnConsultationPage";
import TradeReturnDisputePage from "../pages/TradeReturnDisputePage";
import DispatchPage from "../pages/DispatchPage";
import WalletStackPage from "../pages/WalletStackPage";
import PublicWalletPage from "../pages/PublicWalletPage";
import ContentsPage from "../pages/ContentsPage";
import AnnouncementWorkspacePage from "../pages/AnnouncementWorkspacePage";
import AnnouncementDetailPage from "../pages/AnnouncementDetailPage";
import TermsPage from "../pages/TermsPage";
import HowToUseWorkspacePage from "../pages/HowToUseWorkspacePage";
import HowToUseDetailPage from "../pages/HowToUseDetailPage";
import ResaleCreatePage from "../pages/ResaleCreatePage";
import ResaleDetailPage from "../pages/ResaleDetailPage";
import ProtectedRoute from "../components/auth/ProtectedRoute";
import { PayoutAccountRegistrationProvider } from "../features/payout/context/PayoutAccountRegistrationProvider";

function RootPage() {
  const [user, setUser] = useState<User | null | undefined>(undefined);

  useEffect(() => {
    const unsubscribe = onAuthStateChanged(auth, (nextUser) => {
      setUser(nextUser);
    });

    return unsubscribe;
  }, []);

  if (user === undefined) {
    return null;
  }

  if (user) {
    return <Navigate to="/lists" replace />;
  }

  return <LandingPage />;
}

function PayoutAccountRouteGroup() {
  return (
    <ProtectedRoute>
      <PayoutAccountRegistrationProvider>
        <Outlet />
      </PayoutAccountRegistrationProvider>
    </ProtectedRoute>
  );
}

export const router = createBrowserRouter([
  {
    path: "/",
    element: <RootPage />,
  },
  {
    path: "/landing",
    element: <LandingPage />,
  },
  {
    path: "/signin",
    element: <SignInPage />,
  },
  {
    path: "/signin/select",
    element: <SignInSelectPage />,
  },
  {
    path: "/signup",
    element: <SignUpPage />,
  },
  {
    path: "/verification-sent",
    element: <VerificationSentPage />,
  },
  {
    path: "/password-reset",
    element: <PasswordResetPage />,
  },
  {
    path: "/how-to-use",
    element: <HowToUseWorkspacePage />,
    children: [
      {
        path: ":category/:slug",
        element: <HowToUseDetailPage />,
      },
    ],
  },
  {
    path: "/resale",
    element: <ResaleCreatePage />,
  },
  {
    path: "/terms",
    element: <TermsPage />,
  },
  {
    path: "/privacy-policy",
    element: <TermsPage />,
  },
  {
    path: "/specified-commercial-transactions",
    element: <TermsPage />,
  },
  {
    path: "/avatar",
    element: (
      <ProtectedRoute>
        <AvatarPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/avatars/:avatarId",
    element: (
      <ProtectedRoute>
        <PublicWalletPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/avatars/:avatarId/reviews",
    element: (
      <ProtectedRoute>
        <AvatarReviewPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/avatar-reviews/order-items/:orderId/:itemIndex/new",
    element: (
      <ProtectedRoute>
        <AvatarReviewCreatePage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/lists",
    element: (
      <ProtectedRoute>
        <ListsPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/lists/:listId",
    element: <CatalogPage />,
  },
  {
    path: "/market",
    element: (
      <ProtectedRoute>
        <MarketPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/market/:resaleId",
    element: (
      <ProtectedRoute>
        <MarketDetailPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/resales/:resaleId",
    element: (
      <ProtectedRoute>
        <ResaleDetailPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/favorites",
    element: (
      <ProtectedRoute>
        <LikesPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/favorites/:listId",
    element: (
      <ProtectedRoute>
        <CatalogPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/brands/:brandId",
    element: <BrandPage />,
  },
  {
    path: "/payments/:listId",
    element: (
      <ProtectedRoute>
        <PaymentPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/order-confirmed",
    element: (
      <ProtectedRoute>
        <OrderConfirmedPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/orders/:orderId",
    element: (
      <ProtectedRoute>
        <OrderDetail />
      </ProtectedRoute>
    ),
  },
  {
    path: "/orders/:orderId/items/:itemIndex/return",
    element: (
      <ProtectedRoute>
        <ReturnRequestPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/cart",
    element: (
      <ProtectedRoute>
        <CartPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/announcements",
    element: (
      <ProtectedRoute>
        <AnnouncementWorkspacePage />
      </ProtectedRoute>
    ),
    children: [
      {
        path: "news/:newsId",
        element: <AnnouncementDetailPage />,
      },
      {
        path: "report-decisions/:notificationId",
        element: <AnnouncementDetailPage />,
      },
      {
        path: ":announcementId",
        element: <AnnouncementDetailPage />,
      },
    ],
  },
  {
    path: "/chats",
    element: (
      <ProtectedRoute>
        <ChatWorkspacePage />
      </ProtectedRoute>
    ),
    children: [
      {
        path: "resales/:resaleId",
        element: <ChatDetailPage />,
      },
      {
        path: "trades/order-items/:orderId/:itemIndex",
        element: <TradeChatRedirectPage />,
      },
      {
        path: "trades/:tradeId",
        element: <ChatDetailPage />,
      },
      {
        path: ":inquiryId",
        element: <ChatDetailPage />,
      },
    ],
  },
  {
    path: "/trades/:tradeId/return-consultation",
    element: (
      <ProtectedRoute>
        <TradeReturnConsultationPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/trades/:tradeId/return-dispute",
    element: (
      <ProtectedRoute>
        <TradeReturnDisputePage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/dispatch/trades/:tradeId",
    element: (
      <ProtectedRoute>
        <DispatchPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/scan",
    element: (
      <ProtectedRoute>
        <ScanPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/scan/result",
    element: <ScanResultPage />,
  },
  {
    path: "/scan/result/:productId",
    element: <ScanResultPage />,
  },
  {
    path: "/inquiries/new",
    element: (
      <ProtectedRoute>
        <ProductDescriptionInquiryPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/wallet",
    element: (
      <ProtectedRoute>
        <WalletStackPage />
      </ProtectedRoute>
    ),
    children: [
      {
        path: "contents",
        element: <ContentsPage />,
      },
      {
        path: "scan-result",
        element: <ScanResultPage />,
      },
      {
        path: "scan-result/:productId",
        element: <ScanResultPage />,
      },
    ],
  },
  {
    path: "/contents",
    element: (
      <ProtectedRoute>
        <ContentsPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/settings/email",
    element: (
      <ProtectedRoute>
        <EmailPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/settings/password",
    element: (
      <ProtectedRoute>
        <PasswordPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/settings/identity-verification",
    element: (
      <ProtectedRoute>
        <IdentityVerificationPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/settings/payment-method",
    element: (
      <ProtectedRoute>
        <PaymentMethodPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/settings/payout-account",
    element: <PayoutAccountRouteGroup />,
    children: [
      {
        index: true,
        element: <PayoutAccountPage />,
      },
      {
        path: "bank",
        element: <PayoutBankSelectPage />,
      },
      {
        path: "branch",
        element: <PayoutBranchSelectPage />,
      },
      {
        path: "account",
        element: <PayoutBankAccountPage />,
      },
      {
        path: "confirm",
        element: <PayoutAccountConfirmPage />,
      },
      {
        path: "complete",
        element: <PayoutAccountCompletePage />,
      },
    ],
  },
  {
    path: "/settings/shipping-address",
    element: (
      <ProtectedRoute>
        <ShippingAddressPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/settings/issue-report",
    element: (
      <ProtectedRoute>
        <IssueReportPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/auth/action",
    element: <AuthActionPage />,
  },
  {
    path: "/:productId",
    element: <ScanResultPage />,
  },
]);