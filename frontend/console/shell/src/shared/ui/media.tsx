// frontend/console/shell/src/shared/ui/media.tsx

import * as React from "react";

import "./media.css";

function cn(...classes: Array<string | undefined | false | null>): string {
  return classes.filter(Boolean).join(" ");
}

export type MediaType =
  | "image"
  | "video"
  | "pdf"
  | "document";

export type MediaVariant =
  | "default"
  | "landscape"
  | "square"
  | "viewer"
  | "cover";

export type MediaFit =
  | "cover"
  | "contain";

export type MediaProps = {
  src?: string | null;
  type?: MediaType;
  alt?: string;
  name?: string;
  contentType?: string;

  variant?: MediaVariant;
  fit?: MediaFit;
  bordered?: boolean;

  className?: string;
  mediaClassName?: string;

  emptyIcon?: React.ReactNode;
  emptyText?: React.ReactNode;
  emptyDescription?: React.ReactNode;

  linkLabel?: string;

  loading?: "eager" | "lazy";

  onActivate?: () => void;
  disabled?: boolean;

  onMediaError?: () => void;

  imageProps?: Omit<
    React.ImgHTMLAttributes<HTMLImageElement>,
    "src" | "alt" | "className" | "loading"
  >;

  videoProps?: Omit<
    React.VideoHTMLAttributes<HTMLVideoElement>,
    "src" | "className" | "children"
  >;

  linkProps?: Omit<
    React.AnchorHTMLAttributes<HTMLAnchorElement>,
    "href" | "className" | "children"
  >;
};

function resolveFileLabel(
  type: MediaType,
  name: string,
  linkLabel?: string,
): string {
  if (linkLabel) {
    return linkLabel;
  }

  if (type === "pdf") {
    return name
      ? `PDFを開く: ${name}`
      : "PDFを開く";
  }

  return name
    ? `ファイルを開く: ${name}`
    : "ファイルを開く";
}

export const Media = React.forwardRef<
  HTMLDivElement,
  MediaProps
>(
  (
    {
      src,
      type = "image",
      alt,
      name,
      contentType,
      variant = "default",
      fit = "cover",
      bordered = true,
      className,
      mediaClassName,
      emptyIcon,
      emptyText = "メディアが登録されていません。",
      emptyDescription,
      linkLabel,
      loading = "lazy",
      onActivate,
      disabled = false,
      onMediaError,
      imageProps,
      videoProps,
      linkProps,
    },
    ref,
  ) => {
    const source = String(src ?? "").trim();
    const displayName = String(name ?? "").trim();
    const isInteractive = Boolean(onActivate);

    const handleClick = () => {
      if (disabled) {
        return;
      }

      onActivate?.();
    };

    const handleKeyDown = (
      event: React.KeyboardEvent<HTMLDivElement>,
    ) => {
      if (
        disabled ||
        !onActivate ||
        (event.key !== "Enter" && event.key !== " ")
      ) {
        return;
      }

      event.preventDefault();
      onActivate();
    };

    const renderContent = () => {
      if (!source) {
        return (
          <div className="ui-media__empty">
            {emptyIcon ? (
              <div className="ui-media__empty-icon">
                {emptyIcon}
              </div>
            ) : null}

            {emptyText ? (
              <div className="ui-media__empty-title">
                {emptyText}
              </div>
            ) : null}

            {emptyDescription ? (
              <div className="ui-media__empty-description">
                {emptyDescription}
              </div>
            ) : null}
          </div>
        );
      }

      if (type === "image") {
        return (
          <img
            {...imageProps}
            src={source}
            alt={alt ?? displayName ?? "画像"}
            loading={loading}
            className={cn(
              "ui-media__image",
              `ui-media__media--${fit}`,
              mediaClassName,
            )}
            onError={(event) => {
              imageProps?.onError?.(event);
              onMediaError?.();
            }}
          />
        );
      }

      if (type === "video") {
        return (
          <video
            {...videoProps}
            className={cn(
              "ui-media__video",
              `ui-media__media--${fit}`,
              mediaClassName,
            )}
            onError={(event) => {
              videoProps?.onError?.(event);
              onMediaError?.();
            }}
          >
            <source
              src={source}
              type={contentType}
            />

            お使いのブラウザは動画再生に対応していません。
          </video>
        );
      }

      return (
        <a
          {...linkProps}
          href={source}
          target={linkProps?.target ?? "_blank"}
          rel={linkProps?.rel ?? "noreferrer"}
          className={cn(
            "ui-media__file-link",
            mediaClassName,
          )}
        >
          {resolveFileLabel(
            type,
            displayName,
            linkLabel,
          )}
        </a>
      );
    };

    return (
      <div
        ref={ref}
        className={cn(
          "ui-media",
          `ui-media--${variant}`,
          !bordered && "ui-media--borderless",
          isInteractive && "ui-media--interactive",
          disabled && "ui-media--disabled",
          className,
        )}
        role={isInteractive ? "button" : undefined}
        tabIndex={
          isInteractive && !disabled
            ? 0
            : undefined
        }
        aria-disabled={
          isInteractive
            ? disabled
            : undefined
        }
        onClick={
          isInteractive
            ? handleClick
            : undefined
        }
        onKeyDown={
          isInteractive
            ? handleKeyDown
            : undefined
        }
      >
        {renderContent()}
      </div>
    );
  },
);

Media.displayName = "Media";

export default Media;