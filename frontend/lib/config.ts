const configuredAppId = Number.parseInt(import.meta.env.VITE_APP_ID ?? "", 10);
const configuredApiBase = import.meta.env.VITE_API_BASE?.trim();

export const isDesktop = import.meta.env.VITE_SKILLHUB_DESKTOP === "true";

export const config = {
  apiBase: configuredApiBase || "",
  appId: Number.isInteger(configuredAppId) && configuredAppId > 0 ? configuredAppId : 1,
};
