import path from "node:path";

import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import { VitePWA } from "vite-plugin-pwa";

const apiTarget = "http://localhost:8080";

const themeColor = "#f2f2f7";

const remoteImageMaxAgeSeconds = 7 * 24 * 60 * 60;

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    VitePWA({
      registerType: "autoUpdate",
      includeAssets: [
        "favicon.ico",
        "apple-touch-icon-180x180.png",
        "apple-touch-icon-180x180-dark.png",
        "logo.svg",
      ],
      manifest: {
        name: "fin",
        short_name: "fin",
        description: "Personal finance tracker",
        display: "standalone",
        start_url: "/",
        scope: "/",
        theme_color: themeColor,
        background_color: themeColor,
        icons: [
          { src: "pwa-64x64.png", sizes: "64x64", type: "image/png" },
          { src: "pwa-192x192.png", sizes: "192x192", type: "image/png" },
          { src: "pwa-512x512.png", sizes: "512x512", type: "image/png" },
          {
            src: "maskable-icon-512x512.png",
            sizes: "512x512",
            type: "image/png",
            purpose: "maskable",
          },
        ],
      },
      workbox: {
        navigateFallback: "/index.html",
        navigateFallbackDenylist: [
          /^\/api\//u,
          /^\/auth\//u,
          /^\/swagger/u,
          /^\/openapi/u,
          /^\/version$/u,
          /^\/healthz$/u,
          /^\/readyz$/u,
          /^\/metrics$/u,
        ],
        runtimeCaching: [
          {
            urlPattern: ({ request, sameOrigin }) => request.destination === "image" && !sameOrigin,
            handler: "CacheFirst",
            options: {
              cacheName: "remote-images",
              expiration: { maxAgeSeconds: remoteImageMaxAgeSeconds, maxEntries: 200 },
              cacheableResponse: { statuses: [0, 200] },
            },
          },
        ],
      },
    }),
  ],
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "src"),
    },
  },
  server: {
    proxy: {
      "/api": apiTarget,
      "/version": apiTarget,
      "/healthz": apiTarget,
      "/readyz": apiTarget,
      "/metrics": apiTarget,
      "/auth": apiTarget,
      "/openapi.json": apiTarget,
      "/swagger": apiTarget,
    },
  },
});
