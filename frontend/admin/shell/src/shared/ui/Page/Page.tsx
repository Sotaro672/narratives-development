// frontend/admin/shell/src/shared/ui/Page/Page.tsx
import type { ReactNode } from "react";

import "./Page.css";

type PageProps = {
  children: ReactNode;
};

export default function Page({ children }: PageProps) {
  return <div className="ui-page">{children}</div>;
}