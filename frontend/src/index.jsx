import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import App from "./App";
import "./index.css";

const root = ReactDOM.createRoot(document.getElementById("root"));
root.render(
  <React.StrictMode>
    <BrowserRouter>
      <div className="dark min-h-screen bg-[#0b0202] text-slate-100">
        <App />
      </div>
    </BrowserRouter>
  </React.StrictMode>,
);
