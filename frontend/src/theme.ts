import { alpha, createTheme, responsiveFontSizes } from "@mui/material/styles";

export const ARMADA_BRAND = {
  melon: "#48bc8e",
  grapefruit: "#e73953",
  licorice: "#2d2d2c",
  snow: "#ffffff",
  pineapple: "#f7b519",
} as const;

const sharedTypography = {
  fontFamily: '"Lato", "Inter", "Segoe UI", sans-serif',
  h1: {
    fontFamily: '"Bebas Neue", "Oswald", "Arial Narrow", sans-serif',
    letterSpacing: "0.08em",
  },
  h2: {
    fontFamily: '"Bebas Neue", "Oswald", "Arial Narrow", sans-serif',
    letterSpacing: "0.07em",
  },
  h3: {
    fontFamily: '"Bebas Neue", "Oswald", "Arial Narrow", sans-serif',
    letterSpacing: "0.06em",
  },
  h4: {
    fontFamily: '"Bebas Neue", "Oswald", "Arial Narrow", sans-serif',
    letterSpacing: "0.05em",
  },
  h5: {
    fontFamily: '"Bebas Neue", "Oswald", "Arial Narrow", sans-serif',
    letterSpacing: "0.05em",
  },
  h6: {
    fontFamily: '"Bebas Neue", "Oswald", "Arial Narrow", sans-serif',
    letterSpacing: "0.04em",
  },
  button: {
    fontWeight: 700,
    letterSpacing: "0.04em",
    textTransform: "none" as const,
  },
} as const;

const buildTheme = (mode: "light" | "dark") =>
  responsiveFontSizes(
    createTheme({
      palette: {
        mode,
        primary: {
          main: ARMADA_BRAND.melon,
          light: "#71cdaa",
          dark: "#338664",
          contrastText: ARMADA_BRAND.licorice,
        },
        secondary: {
          main: ARMADA_BRAND.grapefruit,
          light: "#ee6075",
          dark: "#b5283d",
          contrastText: ARMADA_BRAND.snow,
        },
        warning: { main: ARMADA_BRAND.pineapple },
        success: { main: ARMADA_BRAND.melon },
      },
      shape: { borderRadius: 4 },
      typography: sharedTypography,
      components: {
        MuiCssBaseline: {
          styleOverrides: {
            "::selection": {
              backgroundColor: alpha(ARMADA_BRAND.melon, 0.3),
            },
          },
        },
        // Prevent MUI from adding a gradient overlay in dark mode.
        MuiPaper: {
          styleOverrides: { root: { backgroundImage: "none" } },
        },
        MuiAppBar: {
          styleOverrides: {
            root: {
              backgroundColor: ARMADA_BRAND.licorice,
              color: ARMADA_BRAND.snow,
            },
          },
        },
        MuiButton: {
          defaultProps: { disableElevation: true },
          styleOverrides: {
            root: { textTransform: "none" as const },
          },
        },
      },
    }),
  );

export const armadaTheme = buildTheme("light");

export const armadaDarkTheme = buildTheme("dark");
