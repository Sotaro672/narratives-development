// frontend/amol/src/components/layout/header/HeaderDesktopNavigation.tsx

import { useEffect, useState } from "react";
import {
  getAuth,
  onAuthStateChanged,
  type User,
} from "firebase/auth";
import {
  Link,
  NavLink,
} from "react-router-dom";
import {
  Heart,
  ScanLine,
  ShoppingBag,
  Store,
  UserRound,
} from "lucide-react";

import { getMyAvatar } from "../../../features/avatar/api/avatarApi";
import { getApiBaseUrl } from "../../../lib/apiBaseUrl";
import { publicHeaderNavigationItems } from "./headerNavigationItems";

export default function HeaderDesktopNavigation() {
  const [
    authResolved,
    setAuthResolved,
  ] = useState(false);

  const [
    currentUser,
    setCurrentUser,
  ] = useState<User | null>(null);

  const [
    avatarIcon,
    setAvatarIcon,
  ] = useState("");

  useEffect(() => {
    const auth = getAuth();

    const unsubscribe =
      onAuthStateChanged(
        auth,
        (user) => {
          setCurrentUser(user);
          setAuthResolved(true);
        },
      );

    return unsubscribe;
  }, []);

  useEffect(() => {
    let cancelled = false;

    async function loadAvatarIcon() {
      if (!currentUser) {
        setAvatarIcon("");
        return;
      }

      try {
        const apiBaseUrl =
          getApiBaseUrl();

        if (!apiBaseUrl) {
          setAvatarIcon("");
          return;
        }

        const idToken =
          await currentUser.getIdToken(
            true,
          );

        const avatar =
          await getMyAvatar({
            backendUrl: apiBaseUrl,
            idToken,
          });

        if (cancelled) {
          return;
        }

        setAvatarIcon(
          avatar?.avatarIcon ?? "",
        );
      } catch {
        if (!cancelled) {
          setAvatarIcon("");
        }
      }
    }

    void loadAvatarIcon();

    return () => {
      cancelled = true;
    };
  }, [currentUser]);

  if (!authResolved) {
    return null;
  }

  if (!currentUser) {
    return (
      <nav
        className="header__desktop-nav"
        aria-label="ページナビゲーション"
      >
        {publicHeaderNavigationItems.map(
          (item) => (
            <Link
              key={item.to}
              to={item.to}
              className="header__desktop-nav-link"
            >
              {item.label}
            </Link>
          ),
        )}
      </nav>
    );
  }

  return (
    <nav
      className={[
        "header__desktop-nav",
        "header__desktop-nav--authenticated",
      ].join(" ")}
      aria-label="メインナビゲーション"
    >
      <NavLink
        to="/lists"
        className={({ isActive }) =>
          [
            "header__desktop-nav-link",
            "header__desktop-nav-link--with-icon",
            isActive
              ? "header__desktop-nav-link--active"
              : "",
          ]
            .filter(Boolean)
            .join(" ")
        }
      >
        <ShoppingBag
          className="header__desktop-nav-svg-icon"
          strokeWidth={2.2}
          aria-hidden="true"
        />

        <span>モール</span>
      </NavLink>

      <NavLink
        to="/market"
        className={({ isActive }) =>
          [
            "header__desktop-nav-link",
            "header__desktop-nav-link--with-icon",
            isActive
              ? "header__desktop-nav-link--active"
              : "",
          ]
            .filter(Boolean)
            .join(" ")
        }
      >
        <Store
          className="header__desktop-nav-svg-icon"
          strokeWidth={2.2}
          aria-hidden="true"
        />

        <span>マーケット</span>
      </NavLink>

      <NavLink
        to="/scan"
        className={({ isActive }) =>
          [
            "header__desktop-nav-link",
            "header__desktop-nav-link--with-icon",
            isActive
              ? "header__desktop-nav-link--active"
              : "",
          ]
            .filter(Boolean)
            .join(" ")
        }
      >
        <ScanLine
          className="header__desktop-nav-svg-icon"
          strokeWidth={2.2}
          aria-hidden="true"
        />

        <span>スキャン</span>
      </NavLink>

      <NavLink
        to="/favorites"
        className={({ isActive }) =>
          [
            "header__desktop-nav-link",
            "header__desktop-nav-link--with-icon",
            isActive
              ? "header__desktop-nav-link--active"
              : "",
          ]
            .filter(Boolean)
            .join(" ")
        }
      >
        <Heart
          className="header__desktop-nav-svg-icon"
          strokeWidth={2.2}
          aria-hidden="true"
        />

        <span>お気に入り</span>
      </NavLink>

      <NavLink
        to="/wallet"
        className={({ isActive }) =>
          [
            "header__desktop-nav-link",
            "header__desktop-nav-link--with-icon",
            isActive
              ? "header__desktop-nav-link--active"
              : "",
          ]
            .filter(Boolean)
            .join(" ")
        }
      >
        <span
          className="header__desktop-nav-avatar-wrap"
          aria-hidden="true"
        >
          {avatarIcon ? (
            <img
              src={avatarIcon}
              alt=""
              className="header__desktop-nav-avatar"
            />
          ) : (
            <UserRound
              className="header__desktop-nav-svg-icon"
              strokeWidth={2.2}
            />
          )}
        </span>

        <span>ウォレット</span>
      </NavLink>
    </nav>
  );
}