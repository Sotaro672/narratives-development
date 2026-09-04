//frontend\admin\shell\src\layout\Sidebar\Sidebar.tsx
import { NavLink } from "react-router-dom";

import "./Sidebar.css";

const menuItems = [
  {
    label: "問い合わせ",
    path: "/inquiries",
  },
  {
    label: "ガス",
    path: "/gas",
  },
  {
    label: "契約",
    path: "/contracts",
  },
  {
    label: "通報",
    path: "/reports",
  },
  {
    label: "請求",
    path: "/billing",
  },
];

export default function Sidebar() {
  return (
    <aside className="left-sidebar">
      <nav className="sidebar-nav">
        {menuItems.map((item) => (
          <NavLink
            key={item.path}
            to={item.path}
            className={({ isActive }) =>
              [
                "sidebar-item",
                isActive
                  ? "sidebar-item-active"
                  : "",
              ]
                .filter(Boolean)
                .join(" ")
            }
          >
            {item.label}
          </NavLink>
        ))}
      </nav>
    </aside>
  );
}