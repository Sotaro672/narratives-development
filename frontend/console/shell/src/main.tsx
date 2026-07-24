import "../../shell/src/shared/index.css";

import React from "react";
import ReactDOM from "react-dom/client";
import App from "./app/App";

const container = document.getElementById("root");
if (!container) {
  throw new Error(
    "Root element not found. Please ensure index.html includes <div id='root'></div>."
  );
}

const root = ReactDOM.createRoot(container);

root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);

if (import.meta.hot) {
  import.meta.hot.accept();
}