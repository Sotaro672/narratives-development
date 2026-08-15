// frontend/amol/src/main.tsx

import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClientProvider } from "@tanstack/react-query";
import { SolanaProvider } from "@solana/react-hooks";
import { RouterProvider } from "react-router-dom";

import { queryClient } from "./lib/queryClient";
import { solanaClient } from "./lib/solana";
import { router } from "./router";

import "./styles/reset.css";
import "./styles/variables.css";
import "./styles/globals.css";
import "./styles/page-layout.css";
import "./styles/page-split-layout.css";
import "./styles/form.css";

const rootElement = document.getElementById("root");

if (!rootElement) {
  throw new Error("root element was not found");
}

ReactDOM.createRoot(rootElement).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <SolanaProvider client={solanaClient}>
        <RouterProvider router={router} />
      </SolanaProvider>
    </QueryClientProvider>
  </React.StrictMode>,
);