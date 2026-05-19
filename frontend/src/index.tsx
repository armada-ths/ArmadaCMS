import "@fontsource/bebas-neue/400.css";
import "@fontsource/lato/400.css";
import "@fontsource/lato/700.css";
import ReactDOM from "react-dom/client";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { App } from "./App";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <BrowserRouter>
    <Routes>
      <Route path="/" element={<Navigate to="/admin" replace />} />
      <Route path="/admin/*" element={<App />} />
    </Routes>
  </BrowserRouter>,
);
