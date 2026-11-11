/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: "class",
  content: ["./src/**/*.{js,jsx}"],
  theme: {
    extend: {
      colors: {
        obsidian: {
          950: "#0b0202",
          900: "#120505",
          800: "#1d0909",
        },
        ruby: {
          200: "#fecaca",
          300: "#fca5a5",
          400: "#f87171",
          500: "#dc2626",
          600: "#b91c1c",
          700: "#991b1b",
        },
        cream: "#f7f8f5",
      },
      boxShadow: {
        card: "0 18px 48px rgba(0, 0, 0, 0.26)",
      },
    },
  },
  plugins: [],
};
