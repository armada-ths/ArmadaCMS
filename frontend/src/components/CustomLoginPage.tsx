import { Login, LoginForm } from "react-admin";
import { Box, Typography } from "@mui/material";

export const CustomLoginPage = () => (
  <Login>
    <Box
      sx={{
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        pt: 1,
        px: 3,
      }}
    >
      <Typography variant="h5" gutterBottom>
        Welcome to ArmadaCMS
      </Typography>
      <Typography
        variant="body2"
        color="text.secondary"
        textAlign="center"
        gutterBottom
      >
        Sign in with your credentials to manage the Armada platform.
      </Typography>
      <LoginForm />
    </Box>
  </Login>
);
