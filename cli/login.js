#!/usr/bin/env node
const fs = require("fs");
const path = require("path");
const https = require("https");
const http = require("http");
const readline = require("readline");

const CONFIG_DIR = path.join(process.env.HOME || process.env.USERPROFILE, ".agent-hub");
const CONFIG_FILE = path.join(CONFIG_DIR, "config.json");

function request(url, body) {
  return new Promise((resolve, reject) => {
    const u = new URL(url);
    const data = JSON.stringify(body);
    const lib = u.protocol === "http:" ? http : https;
    const req = lib.request(
      {
        hostname: u.hostname,
        port: u.port || undefined,
        path: u.pathname + u.search,
        method: "POST",
        headers: { "Content-Type": "application/json", "Content-Length": Buffer.byteLength(data) },
      },
      (res) => {
        let d = "";
        res.on("data", (c) => (d += c));
        res.on("end", () => {
          try {
            resolve(JSON.parse(d));
          } catch {
            reject(new Error(`HTTP ${res.statusCode}: ${d}`));
          }
        });
      }
    );
    req.on("error", reject);
    req.write(data);
    req.end();
  });
}

async function main() {
  const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
  const ask = (q) => new Promise((r) => rl.question(q, r));

  console.log("\n  agent-hub login\n");

  const existing = fs.existsSync(CONFIG_FILE) ? JSON.parse(fs.readFileSync(CONFIG_FILE, "utf8")) : {};
  const hubUrl =
    (await ask(`  Hub URL [${existing.hub_url || "https://hub.stifer.xyz"}]: `)) ||
    existing.hub_url ||
    "https://hub.stifer.xyz";
  const email =
    (await ask(`  Email [${existing.email || "admin@stifer.xyz"}]: `)) ||
    existing.email ||
    "admin@stifer.xyz";
  const password = await ask("  Password: ");

  rl.close();

  process.stdout.write("  Logging in... ");
  const res = await request(`${hubUrl.replace(/\/$/, "")}/v1/hub/auth/login`, {
    email,
    password,
  });
  if (!res.data?.token) {
    console.log("FAILED");
    console.log(`  ${res.message || "Unknown error"}`);
    process.exit(1);
  }

  fs.mkdirSync(CONFIG_DIR, { recursive: true });
  const cfg = {
    hub_url: hubUrl.replace(/\/$/, ""),
    token: res.data.token,
    email: res.data.email || email,
    user_id: res.data.user_id,
    login_at: new Date().toISOString(),
  };
  if (existing.api_key) cfg.api_key = existing.api_key;
  if (existing.business_code) cfg.business_code = existing.business_code;
  fs.writeFileSync(CONFIG_FILE, JSON.stringify(cfg, null, 2));
  console.log("OK");
  console.log(`  Config saved to ${CONFIG_FILE}`);
  console.log("  MCP tools ready. Restart Claude Code to use.\n");
}

main().catch((e) => {
  console.error("Error:", e.message);
  process.exit(1);
});
