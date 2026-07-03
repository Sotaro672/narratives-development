// frontend/amol/src/components/layout/FooterNav.tsx
import { useEffect, useState } from "react";
import { getAuth, onAuthStateChanged } from "firebase/auth";
import { NavLink, useLocation } from "react-router-dom";
import {
  Heart,
  MessageCircle,
  ScanLine,
  ShoppingBag,
  Store,
  UserRound,
} from "lucide-react";
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
      onButtonClick: () => void | Promise<void>;
    }
  | {
      variant: "commentAction";
      value: string;
      placeholder?: string;
      buttonLabel: string;
      disabled?: boolean;
      posting?: boolean;
      onChange: (value: string) => void;
      onSubmit: () => void | Promise<void>;
    }
  | {
      variant: "reviewAction";
      value: string;
      rating: number;
      placeholder?: string;
      buttonLabel: string;
      disabled?: boolean;
      posting?: boolean;
      onChange: (value: string) => void;
      onRatingChange: (rating: number) => void;
      onSubmit: () => void | Promise<void>;
    };

type AvatarResponse = {
  avatarId?: string;
  avatarName?: string;
  avatarIcon?: string | null;
};

function isAvatarResponse(value: unknown): value is AvatarResponse {
  if (!value || typeof value !== "object") return false;

  return "avatarIcon" in value || "avatarName" in value || "avatarId" in value;
}

export default function FooterNav(props: FooterNavProps) {
  const location = useLocation();
  const [avatarIcon, setAvatarIcon] = useState("");
  const [reviewRatingOpen, setReviewRatingOpen] = useState(false);

  useEffect(() => {
    if (
      props.variant === "action" ||
      props.variant === "commentAction" ||
      props.variant === "reviewAction"
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
        const backendUrl = import.meta.env.VITE_API_BASE_URL;

        if (!backendUrl) {
          setAvatarIcon("");
          return;
        }

        const idToken = await user.getIdToken(true);

        const response = await fetch(
          `${String(backendUrl).replace(/\/$/, "")}/mall/me/avatars`,
          {
            method: "GET",
            headers: {
              Accept: "application/json",
              Authorization: `Bearer ${idToken}`,
            },
            credentials: "include",
          },
        );

        if (!response.ok) {
          setAvatarIcon("");
          return;
        }

        const contentType = response.headers.get("content-type") || "";

        if (!contentType.includes("application/json")) {
          setAvatarIcon("");
          return;
        }

        const responseBody: unknown = await response.json();

        if (isAvatarResponse(responseBody) && responseBody.avatarIcon) {
          setAvatarIcon(responseBody.avatarIcon);
        } else {
          setAvatarIcon("");
        }
      } catch (error) {
        console.error(error);
        setAvatarIcon("");
      }
    });

    return unsubscribe;
  }, [props.variant]);

  useEffect(() => {
    setReviewRatingOpen(false);
  }, [location.pathname]);

  if (props.variant === "action") {
    const { buttonLabel, disabled = false, onButtonClick } = props;
    const isResalePageAction = location.pathname === "/resale";

    const footerClassName = [
      "footer-nav--action",
      isResalePageAction ? "footer-nav--resale-action" : "",
    ]
      .filter(Boolean)
      .join(" ");

    return (
      <footer className={footerClassName}>
        <button
          type="button"
          className="footer-nav__action-button"
          onClick={onButtonClick}
          disabled={disabled}
          aria-label={buttonLabel}
        >
          {buttonLabel}
        </button>
      </footer>
    );
  }

  if (props.variant === "commentAction") {
    const {
      value,
      placeholder = "コメントを書く…",
      buttonLabel,
      disabled = false,
      posting = false,
      onChange,
      onSubmit,
    } = props;

    const canSubmit = !disabled && value.trim().length > 0;

    return (
      <footer className="footer-nav--comment-action">
        <textarea
          className="footer-nav__comment-input"
          value={value}
          rows={1}
          placeholder={placeholder}
          disabled={posting}
          aria-label={placeholder}
          onChange={(event) => onChange(event.target.value)}
        />

        <button
          type="button"
          className="footer-nav__comment-button"
          disabled={!canSubmit}
          aria-label={buttonLabel}
          onClick={() => void onSubmit()}
        >
          {posting ? "投稿中" : buttonLabel}
        </button>
      </footer>
    );
  }

  if (props.variant === "reviewAction") {
    const {
      value,
      rating,
      placeholder = "口コミを入力",
      buttonLabel,
      disabled = false,
      posting = false,
      onChange,
      onRatingChange,
      onSubmit,
    } = props;

    const canSubmit = !disabled && !posting && value.trim().length > 0;
    const ratingOptions = [5, 4, 3, 2, 1];

    const handleRatingChange = (nextRating: number) => {
      onRatingChange(nextRating);
      setReviewRatingOpen(false);
    };

    return (
      <footer className="footer-nav--review-action">
        <div className="footer-nav__review-rating-wrap">
          <button
            type="button"
            className="footer-nav__review-rating-button"
            disabled={posting}
            aria-label="評価"
            aria-haspopup="listbox"
            aria-expanded={reviewRatingOpen}
            onClick={() => setReviewRatingOpen((open) => !open)}
          >
            <span className="footer-nav__review-rating-stars">★{rating}</span>
            <span
              className="footer-nav__review-rating-caret"
              aria-hidden="true"
            >
              ▾
            </span>
          </button>

          {reviewRatingOpen ? (
            <div className="footer-nav__review-rating-popover" role="listbox">
              {ratingOptions.map((nextRating) => (
                <button
                  key={nextRating}
                  type="button"
                  className={[
                    "footer-nav__review-rating-option",
                    rating === nextRating
                      ? "footer-nav__review-rating-option--selected"
                      : "",
                  ]
                    .filter(Boolean)
                    .join(" ")}
                  role="option"
                  aria-selected={rating === nextRating}
                  onClick={() => handleRatingChange(nextRating)}
                >
                  ★{nextRating}
                </button>
              ))}
            </div>
          ) : null}
        </div>

        <textarea
          className="footer-nav__comment-input footer-nav__review-input"
          value={value}
          rows={1}
          placeholder={placeholder}
          disabled={posting}
          aria-label={placeholder}
          onChange={(event) => onChange(event.target.value)}
        />

        <button
          type="button"
          className="footer-nav__comment-button"
          disabled={!canSubmit}
          aria-label={buttonLabel}
          onClick={() => void onSubmit()}
        >
          {posting ? "投稿中" : buttonLabel}
        </button>
      </footer>
    );
  }

  const renderMode = props.renderMode ?? "bottom";
  const onNavigate = props.onNavigate;
  const centerActionLabel = props.centerActionLabel?.trim() ?? "";
  const hasCenterAction =
    centerActionLabel !== "" && typeof props.onCenterActionClick === "function";

  const footerClassName =
    renderMode === "sidebar" ? "footer-nav footer-nav--sidebar" : "footer-nav";

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
          className="footer-nav__item footer-nav__item--button"
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