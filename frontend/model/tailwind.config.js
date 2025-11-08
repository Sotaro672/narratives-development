// frontend/model/tailwind.config.js
const sharedConfig = require("../shared/tailwind.config.js");

/** @type {import('tailwindcss').Config} */
module.exports = {
  ...sharedConfig,
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
    "../shared/ui/**/*.{js,ts,jsx,tsx}",
  ],
};
