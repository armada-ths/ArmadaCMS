import ReactDOM from "react-dom/client";
import { createBrowserRouter, RouterProvider } from "react-router-dom";
import { App } from "./App";
import { createAppRoutes } from "./routing";

const router = createBrowserRouter(createAppRoutes(<App />));

ReactDOM.createRoot(document.getElementById("root")!).render(
  <RouterProvider router={router} />,
);
