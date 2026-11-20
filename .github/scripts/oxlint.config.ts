import { defineConfig } from "oxlint";

export default defineConfig({
  categories: {
    correctness: "error",
    suspicious: "error",
  },
  rules: {
    "no-underscore-dangle": [
      "error",
      { allow: ["_GITHUB_APP_APP_ID", "_GITHUB_APP_INSTALLATION_ID", "_GITHUB_APP_PEM_CONTENT"] },
    ],
  },
});
