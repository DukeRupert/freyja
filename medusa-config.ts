import { loadEnv, defineConfig } from "@medusajs/framework/utils";

loadEnv(process.env.NODE_ENV || "development", process.cwd());

module.exports = defineConfig({
  projectConfig: {
    databaseUrl: process.env.DATABASE_URL,
    http: {
      storeCors: process.env.STORE_CORS!,
      adminCors: process.env.ADMIN_CORS!,
      authCors: process.env.AUTH_CORS!,
      jwtSecret: process.env.JWT_SECRET || "supersecret",
      cookieSecret: process.env.COOKIE_SECRET || "supersecret",
    },
  },
  modules: [
    {
      resolve: "./src/modules/stripe_product",
      options: {
        api_secret: process.env.STRIPE_SECRET || "supersecret_stripe_key",
        webhook_secret: "supersecret_stripe_webhook_key",
      },
    },
    {
      resolve: "./src/modules/plan",
      options: {},
    },
  ],
});
