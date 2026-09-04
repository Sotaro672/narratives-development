//frontend\admin\shell\src\pages\MainPage.tsx
import Header from "../layout/Header/Header";
import Sidebar from "../layout/Sidebar/Sidebar";
import Main from "../layout/Main/Main";

type MainPageProps = {
  onLogout: () => void;
};

export default function MainPage({
  onLogout,
}: MainPageProps) {
  return (
    <>
      <Header
        onLogout={onLogout}
      />

      <Sidebar />

      <Main />
    </>
  );
}