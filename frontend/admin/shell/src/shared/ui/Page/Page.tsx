// frontend/admin/shell/src/shared/ui/Page/Page.tsx
import type { ReactNode } from "react";

import "./Page.css";

type PageProps = {
  children: ReactNode;
};

type PageHeaderProps = {
  title: ReactNode;
  meta?: ReactNode;
  leading?: ReactNode;
  actions?: ReactNode;
};

type DetailPageBodyProps = {
  main: ReactNode;
  aside: ReactNode;
};

export default function Page({ children }: PageProps) {
  return <div className="ui-page">{children}</div>;
}

export function PageHeader({
  title,
  meta,
  leading,
  actions,
}: PageHeaderProps) {
  return (
    <header className="ui-page-header">
      <div className="ui-page-header__left">
        {leading && <div className="ui-page-header__leading">{leading}</div>}

        <div className="ui-page-header__heading">
          <h1 className="ui-page-header__title">{title}</h1>
          {meta && <span className="ui-page-header__meta">{meta}</span>}
        </div>
      </div>

      {actions && <div className="ui-page-header__actions">{actions}</div>}
    </header>
  );
}

export function DetailPageBody({ main, aside }: DetailPageBodyProps) {
  return (
    <div className="ui-detail-page">
      <main className="ui-detail-page__main">{main}</main>
      <aside className="ui-detail-page__aside">{aside}</aside>
    </div>
  );
}