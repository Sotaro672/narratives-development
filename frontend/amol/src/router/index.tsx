// frontend/amol/src/router/index.tsx
import { useEffect, useState } from "react";
import { createBrowserRouter, Navigate } from "react-router-dom";
import { onAuthStateChanged, type User } from "firebase/auth";

import { auth } from "../lib/firebase";

import LandingPage from "../pages/LandingPage";
import SignInPage from "../pages/SignInPage";
import SignInSelectPage from "../pages/SignInSelectPage";
import SignUpPage from "../pages/SignUpPage";
import VerificationSentPage from "../pages/VerificationSentPage";
import PasswordResetPage from "../pages/PasswordResetPage";
import AvatarPage from "../pages/AvatarPage";
import EmailPage from "../pages/EmailPage";
import PasswordPage from "../pages/PasswordPage";
import PaymentMethodPage from "../pages/PaymentMethodPage";
import ShippingAddressPage from "../pages/ShippingAddressPage";
import AuthActionPage from "../pages/AuthActionPage";
import ListsPage from "../pages/ListsPage";
import CartPage from "../pages/CartPage";
import CatalogPage from "../pages/CatalogPage";
import BrandPage from "../pages/BrandPage";
import PaymentPage from "../pages/PaymentPage";
import OrderConfirmedPage from "../pages/OrderConfirmedPage";
import ScanPage from "../pages/ScanPage";
import ScanResultPage from "../pages/ScanResultPage";
import InquiryPage from "../pages/InquiryPage";
import WalletPage from "../pages/WalletPage";
import FollowPage from "../pages/FollowPage";
import ContentsPage from "../pages/ContentsPage";
import AvatarShareQrPage from "../pages/AvatarShareQrPage";
import AnnouncementPage from "../pages/AnnouncementPage";
import AnnouncementDetailPage from "../pages/AnnouncementDetailPage";
import TermsPage from "../pages/TermsPage";
import ContactPage from "../pages/ContactPage";
import VisionPage from "../pages/VisionPage";
import HowToUsePage from "../pages/HowToUsePage";
import PricePlan from "../pages/PricePlan";
import ProtectedRoute from "../components/auth/ProtectedRoute";

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

export const router = createBrowserRouter([
  { path: "/", element: <RootPage /> },
  { path: "/landing", element: <LandingPage /> },

  { path: "/signin", element: <SignInPage /> },
  { path: "/signin/select", element: <SignInSelectPage /> },
  { path: "/signup", element: <SignUpPage /> },
  { path: "/verification-sent", element: <VerificationSentPage /> },
  { path: "/password-reset", element: <PasswordResetPage /> },

  { path: "/how-to-use", element: <HowToUsePage /> },
  { path: "/pricing", element: <PricePlan /> },
  { path: "/faq", element: <VisionPage /> },
  { path: "/terms", element: <TermsPage /> },
  { path: "/privacy-policy", element: <TermsPage /> },
  { path: "/specified-commercial-transactions", element: <TermsPage /> },
  { path: "/contact", element: <ContactPage /> },

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
        <WalletPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/avatars/:avatarId/follow",
    element: (
      <ProtectedRoute>
        <FollowPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/avatars/:avatarId/share-qr",
    element: (
      <ProtectedRoute>
        <AvatarShareQrPage />
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
        <AnnouncementPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/announcements/:announcementId",
    element: (
      <ProtectedRoute>
        <AnnouncementDetailPage />
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
        <InquiryPage />
      </ProtectedRoute>
    ),
  },
  {
    path: "/wallet",
    element: (
      <ProtectedRoute>
        <WalletPage />
      </ProtectedRoute>
    ),
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
    path: "/settings/payment-method",
    element: (
      <ProtectedRoute>
        <PaymentMethodPage />
      </ProtectedRoute>
    ),
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
    path: "/auth/action",
    element: <AuthActionPage />,
  },
  {
    path: "/:productId",
    element: <ScanResultPage />,
  },
]);