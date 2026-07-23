import fs from "node:fs";
import path from "node:path";
import os from "node:os";

const CONFIG_DIR = path.join(os.homedir(), ".agent-hub");
export const CONFIG_FILE = path.join(CONFIG_DIR, "config.json");

export function getBaseUrl() {
  return (process.env.HUB_API_URL || loadConfig().hub_url || "https://hub.stifer.xyz").replace(/\/$/, "");
}

export function loadConfig() {
  try {
    if (fs.existsSync(CONFIG_FILE)) {
      return JSON.parse(fs.readFileSync(CONFIG_FILE, "utf8"));
    }
  } catch {
    /* ignore */
  }
  return {};
}

export function saveConfig(patch) {
  const cur = loadConfig();
  const next = { ...cur, ...patch };
  fs.mkdirSync(CONFIG_DIR, { recursive: true });
  fs.writeFileSync(CONFIG_FILE, JSON.stringify(next, null, 2));
  return next;
}

/** Resolve business_code: args → env → config → project .mycompany → error (no default). */
export function resolveBusinessCode(args = {}) {
  if (args.business_code) return args.business_code;
  if (process.env.HUB_BUSINESS_CODE) return process.env.HUB_BUSINESS_CODE;
  const cfg = loadConfig();
  if (cfg.business_code) return cfg.business_code;
  const fromProject = readProjectHubClient()?.business_code;
  if (fromProject) return fromProject;
  throw new Error(
    "business_code required. Pass business_code, set HUB_BUSINESS_CODE, or put it in ~/.agent-hub/config.json or .mycompany/hub-client.json / .mycompany/config.json"
  );
}

/**
 * Optional machine API key. Returns null if missing (does not throw).
 * Only worker tools (heartbeat/locks/events/playbooks) should require a key.
 */
export function resolveApiKey(args = {}) {
  if (args.api_key) return args.api_key;
  if (process.env.HUB_API_KEY) return process.env.HUB_API_KEY;
  const cfg = loadConfig();
  if (cfg.api_key) return cfg.api_key;
  const fromProject = readProjectHubClient()?.api_key;
  if (fromProject) return fromProject;
  return null;
}

/** Require machine API key or throw a clear machine-path error. */
export function requireApiKey(args = {}) {
  const key = resolveApiKey(args);
  if (key) return key;
  throw new Error(
    "Machine API key missing for this worker tool. Set HUB_API_KEY / ~/.agent-hub/config.json api_key for CI/unattended workers. Human ops use hub_login (JWT) instead."
  );
}

/**
 * User JWT: env HUB_TOKEN / HUB_JWT → ~/.agent-hub/config.json token.
 * Returns null if missing.
 */
export function resolveUserToken(args = {}) {
  if (args.token) return args.token;
  if (process.env.HUB_TOKEN) return process.env.HUB_TOKEN;
  if (process.env.HUB_JWT) return process.env.HUB_JWT;
  const cfg = loadConfig();
  if (cfg.token) return cfg.token;
  return null;
}

function readProjectHubClient() {
  const roots = [process.cwd()];
  // walk up a few levels for monorepos
  let cur = process.cwd();
  for (let i = 0; i < 4; i++) {
    const parent = path.dirname(cur);
    if (parent === cur) break;
    cur = parent;
    roots.push(cur);
  }
  for (const root of roots) {
    for (const rel of [".mycompany/hub-client.json", ".mycompany/config.json"]) {
      const p = path.join(root, rel);
      try {
        if (fs.existsSync(p)) {
          return JSON.parse(fs.readFileSync(p, "utf8"));
        }
      } catch {
        /* ignore */
      }
    }
  }
  return null;
}
