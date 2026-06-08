#!/usr/bin/env node
const fs = require("fs");
const path = require("path");
const https = require("https");
const readline = require("readline");

const CONFIG_DIR = path.join(process.env.HOME || process.env.USERPROFILE, ".agent-hub");
const CONFIG_FILE = path.join(CONFIG_DIR, "config.json");

function request(url, body) {
  return new Promise((resolve, reject) => {
    const u = new URL(url);
    const data = JSON.stringify(body);
    const req = https.request(
      { hostname: u.hostname, path: u.pathname, method: "POST", headers: { "Content-Type": "application/json" } },
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
  const hubUrl = (await ask(`  Hub URL [${existing.hub_url || "https://hub.stifer.xyz"}]: `)) || existing.hub_url || "https://hub.stifer.xyz";
  const password = await ask("  Admin password: ");

  rl.close();

  process.stdout.write("  Logging in... ");
  const res = await request(`${hubUrl}/v1/hub/auth/login`, { password });
  if (!res.data?.token) {
    console.log("FAILED");
    console.log(`  ${res.message || "Unknown error"}`);
    process.exit(1);
  }

  fs.mkdirSync(CONFIG_DIR, { recursive: true });
  fs.writeFileSync(CONFIG_FILE, JSON.stringify({ hub_url: hubUrl, token: res.data.token, login_at: new Date().toISOString() }, null, 2));
  console.log("OK");
  console.log(`  Config saved to ${CONFIG_FILE}`);
  console.log("  MCP tools ready. Restart Claude Code to use.\n");
}

main().catch((e) => {
  console.error("Error:", e.message);
  process.exit(1);
});
