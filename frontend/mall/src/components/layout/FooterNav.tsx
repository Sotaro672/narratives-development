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
      buttonWidth?: "full" | "content";
      disabled?: boolean;
      buttonType?: "button" | "submit";
      buttonForm?: string;
      onButtonClick?: () => void | Promise<void>;
    }
  | {
      variant: "tripleAction";
      leftButtonLabel: string;
      centerButtonLabel: string;
      rightButtonLabel: string;
      leftButtonDisabled?: boolean;
      centerButtonDisabled?: boolean;
      rightButtonDisabled?: boolean;
      onLeftButtonClick: () => void | Promise<void>;
      onCenterButtonClick: () => void | Promise<void>;
      onRightButtonClick: () => void | Promise<void>;
    };

export default function FooterNav(props: FooterNavProps) {
  const [avatarIcon, setAvatarIcon] = useState("");

  useEffect(() => {
    if (
      props.variant === "action" ||
      props.variant === "tripleAction"
    ) {
      return;
    }

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
      buttonWidth = "full",
      disabled = false,
      buttonType = "button",
      buttonForm,
      onButtonClick,
    } = props;

    const isContentWidth = buttonWidth === "content";

    return (
      <footer
        className={[
          "footer-nav--action",
          isContentWidth ? "footer-nav--action--content" : "",
        ]
          .filter(Boolean)
          .join(" ")}
      >
        <Button
          type={buttonType}
          form={buttonForm}
          variant="primary"
          size="lg"
          fullWidth={!isContentWidth}
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

  if (props.variant === "tripleAction") {
    const {
      leftButtonLabel,
      centerButtonLabel,
      rightButtonLabel,
      leftButtonDisabled = false,
      centerButtonDisabled = false,
      rightButtonDisabled = false,
      onLeftButtonClick,
      onCenterButtonClick,
      onRightButtonClick,
    } = props;

    return (
      <footer className="footer-nav--action footer-nav--triple-action">
        <Button
          type="button"
          variant="secondary"
          size="lg"
          className="footer-nav__triple-action-button"
          disabled={leftButtonDisabled}
          aria-label={leftButtonLabel}
          onClick={() => void onLeftButtonClick()}
        >
          {leftButtonLabel}
        </Button>

        <Button
          type="button"
          variant="primary"
          size="lg"
          className="footer-nav__triple-action-button"
          disabled={centerButtonDisabled}
          aria-label={centerButtonLabel}
          onClick={() => void onCenterButtonClick()}
        >
          {centerButtonLabel}
        </Button>

        <Button
          type="button"
          variant="ghost"
          size="lg"
          className="footer-nav__triple-action-button footer-nav__delete-button"
          disabled={rightButtonDisabled}
          aria-label={rightButtonLabel}
          onClick={() => void onRightButtonClick()}
        >
          {rightButtonLabel}
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
          <ShoppingBag
            className="footer-nav__svg-icon"
            strokeWidth={2.2}
          />
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
          <Store
            className="footer-nav__svg-icon"
            strokeWidth={2.2}
          />
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
            <MessageCircle
              className="footer-nav__svg-icon"
              strokeWidth={2.2}
            />
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
            <ScanLine
              className="footer-nav__svg-icon"
              strokeWidth={2.2}
            />
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
          <Heart
            className="footer-nav__svg-icon"
            strokeWidth={2.2}
          />
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
            <img
              src={avatarIcon}
              alt=""
              className="footer-nav__avatar-icon"
            />
          ) : (
            <UserRound
              className="footer-nav__svg-icon"
              strokeWidth={2.2}
            />
          )}
        </span>
        <span className="footer-nav__label">ウォレット</span>
      </NavLink>
    </footer>
  );
}