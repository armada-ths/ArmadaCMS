import { Login, LoginForm } from "react-admin";
import { Card, CardContent, Typography } from "@mui/material";
import { ArmadaBrandLockup } from "./Branding";

export const CustomLoginPage = () => (
  <Login
    sx={{
      minHeight: "100vh",
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      p: { xs: 2, sm: 3 },
      background: "transparent",
      "& .RaLogin-avatar": { display: "none" },
      "& .RaLogin-main": {
        background: "transparent",
        display: "contents",
      },
      "& .RaLogin-card": {
        background: "transparent",
        boxShadow: "none",
        border: "none",
        outline: "none",
        width: "min(100%, 28rem)",
        maxWidth: "none",
        marginTop: "0 !important",
      },
    }}
  >
    <Card
      sx={{
        width: "100%",
        borderRadius: 2,
        border: "1px solid",
        borderColor: "divider",
        backgroundColor: "background.paper",
        boxShadow: 2,
      }}
    >
      <CardContent sx={{ p: { xs: 3, sm: 4 } }}>
        <ArmadaBrandLockup />
        <Typography
          variant="body2"
          color="text.secondary"
          sx={{ mt: 1, mb: 0.5 }}
        >
          Sign in to manage the content on Armada&apos;s website.
        </Typography>

        <LoginForm />
      </CardContent>
    </Card>
  </Login>
);
