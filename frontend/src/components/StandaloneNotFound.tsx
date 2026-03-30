import { Button, Card, CardContent, Stack, Typography } from "@mui/material";
import { Link } from "react-router-dom";

export const StandaloneNotFound = () => (
  <main
    style={{
      minHeight: "100vh",
      display: "grid",
      placeItems: "center",
      padding: "2rem",
      backgroundColor: "#fafafa",
    }}
  >
    <Card sx={{ maxWidth: 520, width: "100%" }}>
      <CardContent>
        <Stack spacing={2}>
          <Typography variant="h4">404: Page not found</Typography>
          <Typography color="text.secondary">
            ArmadaCMS lives under <code>/admin</code>. This URL doesn&apos;t point to
            a valid page.
          </Typography>
          <Button component={Link} to="/admin" variant="contained">
            Open ArmadaCMS
          </Button>
        </Stack>
      </CardContent>
    </Card>
  </main>
);