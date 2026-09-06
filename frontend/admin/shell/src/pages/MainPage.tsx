// frontend/admin/shell/src/pages/MainPage.tsx

import { ContactUnreadProvider } from "../features/contact/context/ContactUnreadContext";
import { ReportPendingProvider } from "../features/report/context/ReportPendingContext";
import Header from "../layout/Header/Header";
import Main from "../layout/Main/Main";
import Sidebar from "../layout/Sidebar/Sidebar";

type MainPageProps = {
  onLogout: () => void;
};

export default function MainPage({ onLogout }: MainPageProps) {
  return (
    <ContactUnreadProvider>
      <ReportPendingProvider>
        <Header onLogout={onLogout} />
        <Sidebar />
        <Main />
      </ReportPendingProvider>
    </ContactUnreadProvider>
  );
}