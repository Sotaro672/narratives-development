// frontend/mall/src/components/layout/FooterNav.tsx

import { useEffect, useState } from "react";
import { getAuth, onAuthStateChanged } from "firebase/auth";
import { NavLink } from "react-router-dom";
import {
  Heart,
  MessageCircle,
  ScanLine,
  ShoppingBag,
  Store,
  UserRound,
} from "lucide-react";

import Button from "../ui/Button";
import { getMyAvatar } from "../../features/avatar/api/avatarApi";

import "./footer.css";

type FooterNavProps =
  | {
      variant?: "default";
      renderMode?: "bottom" | "sidebar";
      onNavigate?: () => void;
      centerActionLabel?: string;
      centerActionDisabled?: boolean;
      onCenterActionClick?: () => void | Promise<void>;
    }
  | {
      variant: "action";
      buttonLabel: string;
      disabled?: boolean;
      buttonType?: "button" | "submit";
      buttonForm?: string;
      onButtonClick?: () => void | Promise<void>;
    };

export default function FooterNav(props: FooterNavProps) {
  const [avatarIcon, setAvatarIcon] = useState("");

  useEffect(() => {
    if (props.variant === "action") return;

    const auth = getAuth();

    const unsubscribe = onAuthStateChanged(auth, async (user) => {
      if (!user) {
        setAvatarIcon("");
        return;
      }

      try {
        const avatar = await getMyAvatar();
        setAvatarIcon(avatar?.avatarIcon ?? "");
      } catch {
        setAvatarIcon("");
      }
    });

    return unsubscribe;
  }, [props.variant]);

  if (props.variant === "action") {
    const {
      buttonLabel,
      disabled = false,
      buttonType = "button",
      buttonForm,
      onButtonClick,
    } = props;

    return (
      <footer className="footer-nav--action">
        <Button
          type={buttonType}
          form={buttonForm}
          variant="primary"
          size="lg"
          fullWidth
          className="footer-nav__action-button"
          onClick={
            onButtonClick
              ? () => void onButtonClick()
              : undefined
          }
          disabled={disabled}
          aria-label={buttonLabel}
        >
          {buttonLabel}
        </Button>
      </footer>
    );
  }

  const renderMode = props.renderMode ?? "bottom";
  const onNavigate = props.onNavigate;
  const centerActionLabel = props.centerActionLabel?.trim() ?? "";
  const hasCenterAction =
    centerActionLabel !== "" &&
    typeof props.onCenterActionClick === "function";

  const footerClassName =
    renderMode === "sidebar"
      ? "footer-nav footer-nav--sidebar"
      : "footer-nav";

  return (
    <footer className={footerClassName}>
      <NavLink
        to="/lists"
        onClick={onNavigate}
        className={({ isActive }) =>
          `footer-nav__item${isActive ? " footer-nav__item--active" : ""}`
        }
      >
        <span className="footer-nav__icon" aria-hidden="true">
          <ShoppingBag className="footer-nav__svg-icon" strokeWidth={2.2} />
        </span>
        <span className="footer-nav__label">モール</span>
      </NavLink>

      <NavLink
        to="/market"
        onClick={onNavigate}
        className={({ isActive }) =>
          `footer-nav__item${isActive ? " footer-nav__item--active" : ""}`
        }
      >
        <span className="footer-nav__icon" aria-hidden="true">
          <Store className="footer-nav__svg-icon" strokeWidth={2.2} />
        </span>
        <span className="footer-nav__label">マーケット</span>
      </NavLink>

      {hasCenterAction ? (
        <button
          type="button"
          onClick={() => void props.onCenterActionClick?.()}
          disabled={props.centerActionDisabled}
          className={[
            "footer-nav__item",
            "footer-nav__item--button",
          ].join(" ")}
          aria-label={centerActionLabel}
        >
          <span className="footer-nav__icon" aria-hidden="true">
            <MessageCircle className="footer-nav__svg-icon" strokeWidth={2.2} />
          </span>
          <span className="footer-nav__label">{centerActionLabel}</span>
        </button>
      ) : (
        <NavLink
          to="/scan"
          onClick={onNavigate}
          className={({ isActive }) =>
            `footer-nav__item${isActive ? " footer-nav__item--active" : ""}`
          }
        >
          <span className="footer-nav__icon" aria-hidden="true">
            <ScanLine className="footer-nav__svg-icon" strokeWidth={2.2} />
          </span>
          <span className="footer-nav__label">スキャン</span>
        </NavLink>
      )}

      <NavLink
        to="/favorites"
        onClick={onNavigate}
        className={({ isActive }) =>
          `footer-nav__item${isActive ? " footer-nav__item--active" : ""}`
        }
      >
        <span className="footer-nav__icon" aria-hidden="true">
          <Heart className="footer-nav__svg-icon" strokeWidth={2.2} />
        </span>
        <span className="footer-nav__label">お気に入り</span>
      </NavLink>

      <NavLink
        to="/wallet"
        onClick={onNavigate}
        className={({ isActive }) =>
          `footer-nav__item${isActive ? " footer-nav__item--active" : ""}`
        }
      >
        <span className="footer-nav__icon" aria-hidden="true">
          {avatarIcon ? (
            <img src={avatarIcon} alt="" className="footer-nav__avatar-icon" />
          ) : (
            <UserRound className="footer-nav__svg-icon" strokeWidth={2.2} />
          )}
        </span>
        <span className="footer-nav__label">ウォレット</span>
      </NavLink>
    </footer>
  );
}