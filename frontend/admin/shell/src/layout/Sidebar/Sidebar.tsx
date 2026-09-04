// frontend/admin/shell/src/layout/Sidebar/Sidebar.tsx
import { NavLink } from "react-router-dom";

import { useContactUnreadCount } from "../../features/contact/hooks/useContactUnreadCount";

import "./Sidebar.css";

const menuItems = [
  { label: "問い合わせ", path: "/inquiries", showUnreadCount: true },
  { label: "ガス", path: "/gas" },
  { label: "契約", path: "/contracts" },
  { label: "通報", path: "/reports" },
  { label: "請求", path: "/billing" },
];

export default function Sidebar() {
  const { unreadCount } = useContactUnreadCount();

  return (
    <aside className="left-sidebar">
      <nav className="sidebar-nav">
        {menuItems.map((item) => (
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

            {item.showUnreadCount && unreadCount > 0 ? (
              <span
                className="sidebar-item__badge"
                aria-label={`未読 ${unreadCount}件`}
              >
                {unreadCount > 99 ? "99+" : unreadCount}
              </span>
            ) : null}
          </NavLink>
        ))}
      </nav>
    </aside>
  );
}