/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./src/**/*.{js,jsx}"],
  theme: {
    extend: {
      colors: {
        navy: {
          950: "#0b1d35",
          900: "#102a49",
          800: "#173b5c",
        },
        aqua: {
          500: "#16b8a6",
          600: "#0c9d91",
        },
        cream: "#f7f8f5",
      },
      boxShadow: {
        card: "0 16px 50px rgba(13, 35, 57, 0.08)",
      },
    },
  },
  plugins: [],
};
