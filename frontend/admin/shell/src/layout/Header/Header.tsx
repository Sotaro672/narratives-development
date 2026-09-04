//frontend\admin\shell\src\layout\Header\Header.tsx
import "./Header.css";

type HeaderProps = {
  onLogout: () => void;
};

export default function Header({
  onLogout,
}: HeaderProps) {
  return (
    <header className="app-header">
      <div className="header-title">
        Admin
      </div>

      <button
        type="button"
        className="header-logout"
        onClick={onLogout}
      >
        ログアウト
      </button>
    </header>
  );
}