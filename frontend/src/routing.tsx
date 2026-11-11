import type { ReactNode } from "react";
import { Box, Button, Stack, Typography } from "@mui/material";
import {
  isRouteErrorResponse,
  Link,
  Navigate,
  useRouteError,
  type RouteObject,
} from "react-router-dom";

function AppRouteError() {
  const error = useRouteError();
  const notFound = isRouteErrorResponse(error) && error.status === 404;

  return (
    <Stack
      component="main"
      spacing={2}
      sx={{ maxWidth: 560, mx: "auto", p: 3, mt: 6 }}
    >
      <Typography component="h1" variant="h4">
        {notFound ? "Page not found" : "Something went wrong"}
      </Typography>
      <Typography>
        {notFound
          ? "This page could not be found. Return to the dashboard to find the page you need."
          : "ArmadaCMS could not load this page. Return to the dashboard and try again."}
      </Typography>
      <Box>
        <Button component={Link} to="/admin" variant="contained" replace>
          Back to dashboard
        </Button>
      </Box>
    </Stack>
  );
}

export const createAppRoutes = (admin: ReactNode): RouteObject[] => [
  {
    path: "/",
    errorElement: <AppRouteError />,
    children: [
      { index: true, element: <Navigate to="/admin" replace /> },
      { path: "admin/*", element: admin },
    ],
  },
];
