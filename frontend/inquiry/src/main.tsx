// frontend/inquiry/src/main.tsx
import { createRoot } from "react-dom/client";
import InquiryManagement from "./pages/InquiryManagement";

const rootEl = document.getElementById("root")!;
createRoot(rootEl).render(<InquiryManagement />);