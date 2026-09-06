// frontend/admin/shell/src/layout/Sidebar/Sidebar.tsx
import { NavLink } from "react-router-dom";

import { useContactUnread } from "../../features/contact/context/ContactUnreadContext";
import { useReportPending } from "../../features/report/context/ReportPendingContext";

import "./Sidebar.css";

const menuItems = [
  { label: "問い合わせ", path: "/inquiries", countType: "contact" as const },
  { label: "ガス", path: "/gas" },
  { label: "契約", path: "/contracts" },
  { label: "通報", path: "/reports", countType: "report" as const },
  { label: "請求", path: "/billing" },
];

export default function Sidebar() {
  const { unreadCount } = useContactUnread();
  const { pendingCount } = useReportPending();

  return (
    <aside className="left-sidebar">
      <nav className="sidebar-nav">
        {menuItems.map((item) => {
          const count =
            item.countType === "contact"
              ? unreadCount
              : item.countType === "report"
                ? pendingCount
                : 0;

          const countLabel =
            item.countType === "report"
              ? `未対応 ${count}件`
              : `未読 ${count}件`;

          return (
            <NavLink
              key={item.path}
              to={item.path}
              className={({ isActive }) =>
                ["sidebar-item", isActive ? "sidebar-item-active" : ""]
                  .filter(Boolean)
                  .join(" ")
              }
            >
              <span className="sidebar-item__label">{item.label}</span>

              {count > 0 ? (
                <span
                  className="sidebar-item__badge"
                  aria-label={countLabel}
                >
                  {count > 99 ? "99+" : count}
                </span>
              ) : null}
            </NavLink>
          );
        })}
      </nav>
    </aside>
  );
}