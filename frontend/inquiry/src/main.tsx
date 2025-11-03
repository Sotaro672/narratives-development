import React from "react";
import { createRoot } from "react-dom/client";
import InquiryManagementPage from "./pages/InquiryManagementPage";

const rootEl = document.getElementById("root")!;
createRoot(rootEl).render(<InquiryManagementPage />);