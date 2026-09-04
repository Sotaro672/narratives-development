//frontend\admin\shell\src\layout\Sidebar\Sidebar.tsx
import "./Sidebar.css";

export default function Sidebar() {
  return (
    <aside className="left-sidebar">
      <nav className="sidebar-nav">
        <button
          type="button"
          className="sidebar-item sidebar-item-active"
        >
          商品レビュー
        </button>

        <button
          type="button"
          className="sidebar-item"
        >
          トークンレビュー
        </button>
      </nav>
    </aside>
  );
}