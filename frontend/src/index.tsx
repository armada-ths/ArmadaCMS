import ReactDOM from "react-dom/client";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { App } from "./App";
import { StandaloneNotFound } from "./components/StandaloneNotFound";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <BrowserRouter>
    <Routes>
      <Route path="/" element={<Navigate to="/admin" replace />} />
      <Route path="/admin/*" element={<App />} />
      <Route path="*" element={<StandaloneNotFound />} />
    </Routes>
  </BrowserRouter>,
);
