/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  darkMode: "class",
  theme: {
    extend: {
      colors: {
        // Primary — fresh agricultural green
        primary: "#1B7A3D",
        "primary-hover": "#156833",
        "primary-dark": "#0D4F25",
        "primary-fixed": "#D4F0DD",
        "primary-fixed-dim": "#B8E6C8",
        "primary-container": "#236B3A",
        "on-primary": "#ffffff",
        "on-primary-fixed": "#0D4F25",
        "on-primary-fixed-variant": "#1B5E2D",
        "on-primary-container": "#0D4F25",
        "inverse-primary": "#B8E6C8",

        // Secondary — warm earth tone
        secondary: "#705a49",
        "secondary-fixed": "#fbddc7",
        "secondary-fixed-dim": "#dec1ac",
        "secondary-container": "#f8dac5",
        "on-secondary": "#ffffff",
        "on-secondary-fixed": "#28180b",
        "on-secondary-fixed-variant": "#574333",
        "on-secondary-container": "#755e4d",

        // Tertiary — tech blue
        tertiary: "#003c5f",
        "tertiary-fixed": "#cde5ff",
        "tertiary-fixed-dim": "#94ccff",
        "tertiary-container": "#005482",
        "on-tertiary": "#ffffff",
        "on-tertiary-fixed": "#001d32",
        "on-tertiary-fixed-variant": "#004b74",
        "on-tertiary-container": "#8bc8ff",

        // Accent — golden yellow for highlights on dark surfaces
        accent: "#F4B942",
        "accent-hover": "#E0A630",
        "accent-light": "#FDE8B4",
        "accent-dark": "#B8861A",

        // Semantic
        success: "#16A34A",
        "success-light": "#DCFCE7",
        "success-dark": "#15803D",
        warning: "#D97706",
        "warning-light": "#FEF3C7",
        "warning-dark": "#B45309",
        error: "#DC2626",
        "error-light": "#FEE2E2",
        "error-dark": "#991B1B",

        // Surfaces
        background: "#F8FAF7",
        "on-background": "#1A1F1B",
        surface: "#FFFFFF",
        "on-surface": "#1A1F1B",
        "surface-variant": "#EDF2EE",
        "on-surface-variant": "#3D4A41",
        "surface-bright": "#FFFFFF",
        "surface-dim": "#D9DDD9",
        "surface-container-lowest": "#FFFFFF",
        "surface-container-low": "#F3F6F2",
        "surface-container": "#EDF2EE",
        "surface-container-high": "#E2E8E3",
        "surface-container-highest": "#D6DDD8",
        "surface-tint": "#1B7A3D",
        "inverse-surface": "#2E3530",
        "inverse-on-surface": "#F0F4F0",

        // Outlines
        outline: "#72796E",
        "outline-variant": "#C7D1C9",
      },
      fontFamily: {
        sans: ["Inter", "Work Sans", "Noto Sans Bengali", "sans-serif"],
        display: ["Plus Jakarta Sans", "Manrope", "Noto Sans Bengali", "sans-serif"],
        headline: ["Manrope", "Noto Sans Bengali", "sans-serif"],
        body: ["Work Sans", "Noto Sans Bengali", "sans-serif"],
      },
      spacing: {
        "margin-mobile": "16px",
        "margin-desktop": "64px",
        gutter: "24px",
        xs: "4px",
        sm: "12px",
        base: "8px",
        md: "24px",
        lg: "48px",
        xl: "80px",
      },
      boxShadow: {
        soft: "0 4px 20px -2px rgba(0, 0, 0, 0.05)",
        hover: "0 12px 30px -4px rgba(0, 0, 0, 0.08)",
        "level-1": "0 4px 20px rgba(0, 0, 0, 0.04)",
        "level-2": "0 8px 24px rgba(0, 0, 0, 0.08)",
      },
      borderRadius: {
        DEFAULT: "0.25rem",
        lg: "0.5rem",
        xl: "0.75rem",
      },
      maxWidth: {
        page: "1440px",
      },
      transitionTimingFunction: {
        "out-soft": "cubic-bezier(0.22, 1, 0.36, 1)",
      },
    },
  },
  plugins: [],
};
