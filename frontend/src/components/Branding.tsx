import { Box, Stack, Typography } from "@mui/material";

type BrandingProps = {
  compact?: boolean;
};

export const ArmadaBrandLockup = ({ compact = false }: BrandingProps) => (
  <Stack direction="row" spacing={compact ? 1 : 1.5} alignItems="center">
    <Box
      component="img"
      src="/icons/armada-icon.svg"
      alt="Armada logo"
      sx={{
        width: compact ? 36 : 48,
        height: compact ? 36 : 48,
        borderRadius: "50%",
        flexShrink: 0,
      }}
    />

    <Box sx={{ minWidth: 0 }}>
      <Typography
        sx={{
          display: "block",
          fontFamily: '"Bebas Neue", "Oswald", "Arial Narrow", sans-serif',
          fontSize: compact ? "0.6rem" : "0.72rem",
          letterSpacing: "0.22em",
          lineHeight: 1,
          color: "inherit",
          opacity: 0.65,
          mb: 0.3,
        }}
      >
        THS ARMADA
      </Typography>
      <Typography
        sx={{
          fontFamily: '"Bebas Neue", "Oswald", "Arial Narrow", sans-serif',
          fontSize: compact ? "1rem" : "1.3rem",
          lineHeight: 1,
          letterSpacing: "0.06em",
        }}
      >
        CMS
      </Typography>
    </Box>
  </Stack>
);
