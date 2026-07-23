import { loadConfig, saveConfig, getBaseUrl, resolveUserToken } from "./config.js";

let adminToken = null;

export function getAdminToken() {
  if (adminToken) return adminToken;
  // prefer live env / disk
  const t = resolveUserToken();
  if (t) {
    adminToken = t;
    return adminToken;
  }
  const cfg = loadConfig();
  if (cfg.token) {
    adminToken = cfg.token;
  }
  return adminToken;
}

export function setAdminToken(token, extra = {}) {
  adminToken = token;
  saveConfig({
    hub_url: getBaseUrl(),
    token,
    login_at: new Date().toISOString(),
    ...extra,
  });
}

export function authHeaders() {
  const t = getAdminToken();
  if (!t) {
    throw new Error("Not logged in. Call hub_login() first (MCP device login). Human tools use JWT — no API key required.");
  }
  return { Authorization: `Bearer ${t}` };
}

// Load token from disk on module import
getAdminToken();
