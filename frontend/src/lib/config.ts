const trimTrailingSlash = (value: string) => value.replace(/\/$/, "");

export const config = {
  apiBaseUrl: trimTrailingSlash(import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080/api/v1"),
  webSocketBaseUrl: trimTrailingSlash(import.meta.env.VITE_WS_BASE_URL ?? "ws://localhost:8080/ws"),
};
