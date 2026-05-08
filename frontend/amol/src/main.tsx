// frontend/src/main.tsx
import React from "react";
import ReactDOM from "react-dom/client";
import { RouterProvider } from "react-router-dom";
import { SolanaProvider } from "@solana/react-hooks";
import { router } from "./router";
import { solanaClient } from "./lib/solana";
import "./styles/reset.css";
import "./styles/variables.css";
import "./styles/globals.css";
import "./styles/app.css";
import "./styles/page-layout.css";
import "./styles/page-split-layout.css";
import "./styles/form.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <SolanaProvider client={solanaClient}>
      <RouterProvider router={router} />
    </SolanaProvider>
  </React.StrictMode>
);