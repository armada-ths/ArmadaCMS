import type { ReactNode } from "react";
import { Navigate, type RouteObject } from "react-router-dom";

export const createAppRoutes = (admin: ReactNode): RouteObject[] => [
  { path: "/", element: <Navigate to="/admin" replace /> },
  { path: "/admin/*", element: admin },
];
