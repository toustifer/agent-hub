// sync-workers.cjs — 读取 .mycompany/workers/*/experience.md → hub JSON → 上传
const fs = require("fs");
const path = require("path");
const https = require("https");

const HUB = process.env.HUB_URL || "https://hub.stifer.xyz";
const API_KEY = process.env.HUB_API_KEY || "agent-company-worker";
const BUSINESS = process.env.HUB_BUSINESS || "ai-medbox";
const WORKERS_DIR = process.argv[2] || process.cwd() + "/.mycompany/workers";

function request(method, path, body) {
  return new Promise((resolve, reject) => {
    const u = new URL(HUB + path);
    const data = JSON.stringify(body);
    const req = https.request(
      { hostname: u.hostname, path: u.pathname, method, headers: { "Content-Type": "application/json", "X-API-Key": API_KEY, "X-Business-Code": BUSINESS } },
      (res) => { let d = ""; res.on("data", c => d += c); res.on("end", () => { try { resolve(JSON.parse(d)); } catch { reject(new Error(`HTTP ${res.statusCode}: ${d}`)); } }); }
    );
    req.on("error", reject);
    req.write(data);
    req.end();
  });
}

function parseMarkdownExperience(md) {
  const out = { patterns: [], gotchas: [], decisions: [] };
  let section = null;
  let current = null;

  for (const line of md.split("\n")) {
    if (line.startsWith("## 模式与最佳实践") || line.startsWith("## Patterns")) { section = "patterns"; continue; }
    if (line.startsWith("## 踩过的坑") || line.startsWith("## Gotchas")) { section = "gotchas"; continue; }
    if (line.startsWith("## 架构决策") || line.startsWith("## Decisions")) { section = "decisions"; continue; }
    if (line.startsWith("## ")) { section = null; current = null; continue; }

    if (section && line.startsWith("- **")) {
      const m = line.match(/\*\*(.+?)\*\*[：:]\s*(.+)/);
      if (m) { current = { title: m[1].trim(), content: m[2].trim(), tags: [] }; out[section].push(current); }
    } else if (current && line.startsWith("  -")) {
      current.content += "\n" + line.trim();
    }
  }
  return out;
}

async function main() {
  const entries = fs.readdirSync(WORKERS_DIR, { withFileTypes: true }).filter(d => d.isDirectory());
  const workers = [];

  // Read leader.json to get owner for each worker
  const leaderPath = path.join(WORKERS_DIR, "..", "leader", "leader.json");
  let leaderWorkers = {};
  if (fs.existsSync(leaderPath)) {
    const leader = JSON.parse(fs.readFileSync(leaderPath, "utf8"));
    for (const w of (leader.workers || [])) {
      if (w.owner) leaderWorkers[w.id] = w.owner;
    }
  }

  for (const e of entries) {
    const expJsonPath = path.join(WORKERS_DIR, e.name, "experience.json");
    const expMdPath = path.join(WORKERS_DIR, e.name, "experience.md");

    let patterns = [], gotchas = [], decisions = [], scope = "", handbook = {};

    // Read handbook.json
    const handbookPath = path.join(WORKERS_DIR, e.name, "handbook.json");
    if (fs.existsSync(handbookPath)) {
      handbook = JSON.parse(fs.readFileSync(handbookPath, "utf8"));
    }

    if (fs.existsSync(expJsonPath)) {
      const j = JSON.parse(fs.readFileSync(expJsonPath, "utf8"));
      scope = j.scope || "";
      patterns = j.patterns || [];
      gotchas = j.gotchas || [];
      decisions = j.decisions || [];
    } else if (fs.existsSync(expMdPath)) {
      const md = fs.readFileSync(expMdPath, "utf8");
      const exp = parseMarkdownExperience(md);
      const scopeMatch = md.match(/业务概述\s*\n-?\s*(.+)/) || md.match(/## 一、业务概述\s*\n\s*\n(.+)/s);
      scope = scopeMatch ? scopeMatch[1].trim().slice(0, 200) : "";
      patterns = exp.patterns;
      gotchas = exp.gotchas;
      decisions = exp.decisions;
    } else {
      console.log(`SKIP ${e.name}: no experience.json or .md`);
      continue;
    }

    workers.push({ worker_id: e.name, version: "1.0.0", host: "windows-dev", pid: 0, owner: leaderWorkers[e.name] || "", scope, handbook, patterns, gotchas, decisions });
    console.log(`READ ${e.name}: ${patterns.length}p ${gotchas.length}g ${decisions.length}d`);
  }

  if (workers.length === 0) { console.log("No workers found."); return; }

  console.log(`\nSyncing ${workers.length} workers to ${HUB}...`);
  const result = await request("POST", "/v1/hub/sync/workers", { business_code: BUSINESS, workers });

  console.log(`\nDone: ${result.data.workers_synced} workers, ${result.data.playbooks_synced} playbooks`);
  if (result.data.errors.length) {
    console.log("Errors:");
    for (const err of result.data.errors) console.log("  " + err);
  }
}

main().catch(e => { console.error(e.message); process.exit(1); });
