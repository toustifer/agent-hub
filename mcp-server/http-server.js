#!/usr/bin/env node
import http from "node:http";

const BASE_URL = process.env.HUB_API_URL || "https://hub.stifer.xyz";
const PORT = parseInt(process.env.PORT || "9001");
let adminToken = null;

async function request(method, path, body, headers = {}) {
  const url = `${BASE_URL}${path}`;
  const opts = { method, headers: { "Content-Type": "application/json", ...headers } };
  if (body) opts.body = JSON.stringify(body);
  const res = await fetch(url, opts);
  const data = await res.json();
  if (!res.ok) throw new Error(`[${res.status}] ${data.message || JSON.stringify(data)}`);
  return data.data;
}

function auth() {
  if (!adminToken) throw new Error("Not logged in. Call hub_login() first.");
  return { Authorization: `Bearer ${adminToken}` };
}

const TOOLS = [
  { name: "hub_login", description: "Login to Agent Hub", inputSchema: { type: "object", properties: { code: { type: "string" } } } },
  { name: "hub_heartbeat",       description: "Worker heartbeat", inputSchema: { type: "object", properties: { business_code:{type:"string"}, worker_id:{type:"string"}, version:{type:"string"}, host:{type:"string"}, pid:{type:"integer"} }, required:["business_code","worker_id","version"] } },
  { name: "hub_acquire_lock",    description: "Acquire distributed lock", inputSchema: { type: "object", properties: { business_code:{type:"string"}, resource_key:{type:"string"}, worker_id:{type:"string"}, ttl_seconds:{type:"integer",default:300} }, required:["business_code","resource_key","worker_id"] } },
  { name: "hub_release_lock",    description: "Release lock",           inputSchema: { type: "object", properties: { holder_token:{type:"string"} }, required:["holder_token"] } },
  { name: "hub_renew_lock",      description: "Renew lock TTL",         inputSchema: { type: "object", properties: { holder_token:{type:"string"}, ttl_seconds:{type:"integer",default:300} }, required:["holder_token"] } },
  { name: "hub_append_event",    description: "Record event",           inputSchema: { type: "object", properties: { business_code:{type:"string"}, actor:{type:"string"}, event_type:{type:"string"}, payload:{type:"object"} }, required:["business_code","actor","event_type"] } },
  { name: "hub_create_playbook", description: "Create playbook",        inputSchema: { type: "object", properties: { business_code:{type:"string"}, category:{type:"string"}, title:{type:"string"}, content:{type:"string"}, tags:{type:"array",items:{type:"string"}}, worker_id:{type:"string"} }, required:["business_code","category","title","content","worker_id"] } },
  { name: "hub_search_playbooks",description: "Search playbooks",       inputSchema: { type: "object", properties: { business_code:{type:"string"}, query:{type:"string"}, category:{type:"string"}, limit:{type:"integer",default:20} }, required:["business_code","query"] } },
  { name: "hub_list_workers",    description: "List workers",           inputSchema: { type: "object", properties: { business_code:{type:"string"}, status:{type:"string"} }, required:["business_code"] } },
  { name: "hub_list_locks",      description: "List active locks",      inputSchema: { type: "object", properties: { business_code:{type:"string"} }, required:["business_code"] } },
  { name: "hub_list_events",     description: "List events",            inputSchema: { type: "object", properties: { business_code:{type:"string"}, event_type:{type:"string"}, limit:{type:"integer",default:20} }, required:["business_code"] } },
  { name: "hub_add_repo",        description: "Bind GitHub repo",       inputSchema: { type: "object", properties: { business_code:{type:"string"}, repo_url:{type:"string"}, default_branch:{type:"string"} }, required:["business_code","repo_url"] } },
  { name: "hub_sync_dag",        description: "Sync DAG task",          inputSchema: { type: "object", properties: { business_code:{type:"string"}, task_id:{type:"string"}, title:{type:"string"}, status:{type:"string"}, assigned_worker:{type:"string"} }, required:["business_code","task_id","title","status"] } },
  { name: "hub_get_dag",         description: "Get DAG tasks",          inputSchema: { type: "object", properties: { business_code:{type:"string"} }, required:["business_code"] } },
  { name: "hub_invite_member",   description: "Invite a member to a business", inputSchema: { type: "object", properties: { business_code:{type:"string"}, email:{type:"string"}, role:{type:"string",default:"member"} }, required:["business_code","email"] } },
  { name: "hub_accept_invitation", description: "Accept an invitation to join a business", inputSchema: { type: "object", properties: { business_code:{type:"string"} }, required:["business_code"] } },
  { name: "hub_list_my_businesses", description: "List businesses I belong to", inputSchema: { type: "object", properties: {} } },
];

async function handleToolCall(name, args) {
  const bc = args?.business_code;
  if (!bc) throw new Error("business_code is required. Read it from .mycompany/config.json in your project.");
  const wh = { "X-API-Key": "agent-company-worker", "X-Business-Code": bc };

  switch (name) {
    case "hub_login": {
      if (args?.code) {
        const tr = await fetch(`${BASE_URL}/v1/hub/auth/device/token?code=${args.code}`);
        const tj = await tr.json();
        if (!tj.data?.token) throw new Error("Code not yet approved.");
        adminToken = tj.data.token;
        return "Logged in! All hub tools ready.";
      }
      const dr = await fetch(`${BASE_URL}/v1/hub/auth/device`, { method: "POST" });
      const dj = await dr.json();
      if (!dr.ok) throw new Error(dj.message || "Failed");
      return `Open this URL:\n\n  ${dj.data.verification_url}\n\nThen call hub_login({ code: "${dj.data.code}" }) to finish.`;
    }
    case "hub_heartbeat": { await request("POST", "/v1/hub/workers/heartbeat", { business_code: bc, worker_id: args.worker_id, version: args.version, host: args.host||"mcp", pid: args.pid||0 }, wh); return `Worker ${args.worker_id} online.`; }
    case "hub_acquire_lock": { const d = await request("POST", "/v1/hub/locks/acquire", { business_code: bc, resource_key: args.resource_key, worker_id: args.worker_id, ttl_seconds: args.ttl_seconds||300 }, wh); return `Lock acquired. Token: ${d.holder_token}`; }
    case "hub_release_lock": { await request("POST", "/v1/hub/locks/release", { holder_token: args.holder_token }, wh); return "Released."; }
    case "hub_renew_lock":   { await request("POST", "/v1/hub/locks/renew", { holder_token: args.holder_token, ttl_seconds: args.ttl_seconds||300 }, wh); return "Renewed."; }
    case "hub_append_event": { const d = await request("POST", "/v1/hub/events", { business_code: bc, actor: args.actor, event_type: args.event_type, payload: args.payload||{} }, wh); return `Event ${d.id} recorded.`; }
    case "hub_create_playbook": { const d = await request("POST", "/v1/hub/playbooks", { business_code: bc, category: args.category, title: args.title, content: args.content, tags: args.tags||[], worker_id: args.worker_id }, wh); return `Playbook ${d.id} created.`; }
    case "hub_search_playbooks": { const p = new URLSearchParams({ q: args.query, limit: String(args.limit||20) }); if (args.category) p.set("category", args.category); const r = await fetch(`${BASE_URL}/v1/hub/playbooks/search?${p}`, { headers: wh }); const list = (await r.json()).data || []; return list.length ? list.map(x => `[${x.category}] ${x.title}`).join("\n") : "No results."; }
    case "hub_list_workers": { const r = await fetch(`${BASE_URL}/v1/hub/workers?${new URLSearchParams(args.business_code?{business:args.business_code}:{})}`, { headers: auth() }); const list = (await r.json()).data||[]; return list.map(w=>`${w.worker_id} ${w.status}`).join("\n")||"No workers."; }
    case "hub_list_locks":   { const r = await fetch(`${BASE_URL}/v1/hub/locks?${new URLSearchParams(args.business_code?{business:args.business_code}:{})}`, { headers: auth() }); const list = (await r.json()).data||[]; return list.map(l=>`${l.resource_key} ${l.holder_worker_id}`).join("\n")||"No locks."; }
    case "hub_list_events":  { const p = new URLSearchParams({ business: args.business_code, limit: String(args.limit||20) }); if(args.event_type) p.set("type",args.event_type); const r = await fetch(`${BASE_URL}/v1/hub/events?${p}`, { headers: auth() }); const list = (await r.json()).data||[]; return list.map(e=>`${(e.created_at||"").slice(0,19)} ${e.event_type}`).join("\n")||"No events."; }
    case "hub_add_repo":    { await request("POST", `/v1/hub/repos/${bc}`, { repo_url: args.repo_url, default_branch: args.default_branch||"main" }, auth()); return `Repo ${args.repo_url} bound.`; }
    case "hub_sync_dag":    { await request("POST", `/v1/hub/dag/${bc}`, { task_id: args.task_id, title: args.title, status: args.status, assigned_worker: args.assigned_worker||"" }, auth()); return `DAG ${args.task_id} → ${args.status}`; }
    case "hub_get_dag":     { const r = await fetch(`${BASE_URL}/v1/hub/dag/${bc}`, { headers: auth() }); const list = (await r.json()).data||[]; return list.map(t=>`${t.status==="completed"?"✓":"⏳"} ${t.task_id} ${t.title}`).join("\n")||"No tasks."; }
    case "hub_invite_member": { const d = await request("POST", `/v1/hub/businesses/${bc}/invite`, { email: args.email, role: args.role||"member" }, auth()); return `Invitation sent to ${args.email}. They can accept at: ${d.invite_url}`; }
    case "hub_accept_invitation": { await request("POST", `/v1/hub/businesses/${bc}/join`, {}, auth()); return `Joined business: ${bc}`; }
    case "hub_list_my_businesses": { const r = await fetch(`${BASE_URL}/v1/hub/me/businesses`, { headers: auth() }); const list = (await r.json()).data||[]; return list.map(b=>`${b.code} — ${b.name} [${b.role}]`).join("\n")||"No businesses."; }
    default: throw new Error(`Unknown tool: ${name}`);
  }
}

http.createServer(async (req, res) => {
  // CORS
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.setHeader("Access-Control-Allow-Headers", "Content-Type, Authorization");
  res.setHeader("Access-Control-Allow-Methods", "POST, GET, OPTIONS");

  if (req.method === "OPTIONS") { res.writeHead(204).end(); return; }
  if (req.method === "GET") { res.writeHead(200, { "Content-Type": "text/plain" }).end("Agent Hub MCP v1.0"); return; }

  if (req.method !== "POST") { res.writeHead(405).end(JSON.stringify({ error: "Method not allowed" })); return; }

  // Read body
  let body = "";
  req.on("data", chunk => body += chunk);
  req.on("end", async () => {
    let rpc;
    try { rpc = JSON.parse(body); } catch {
      res.writeHead(400, { "Content-Type": "application/json" }).end(JSON.stringify({ jsonrpc: "2.0", error: { code: -32700, message: "Parse error" }, id: null }));
      return;
    }

    const { method, params, id } = rpc || {};

    if (method === "tools/list") {
      res.writeHead(200, { "Content-Type": "application/json" }).end(JSON.stringify({ jsonrpc: "2.0", result: { tools: TOOLS }, id }));
      return;
    }

    if (method === "tools/call") {
      try {
        const text = await handleToolCall(params?.name, params?.arguments || {});
        res.writeHead(200, { "Content-Type": "application/json" }).end(JSON.stringify({ jsonrpc: "2.0", result: { content: [{ type: "text", text }] }, id }));
      } catch (err) {
        res.writeHead(200, { "Content-Type": "application/json" }).end(JSON.stringify({ jsonrpc: "2.0", result: { content: [{ type: "text", text: err.message }], isError: true }, id }));
      }
      return;
    }

    if (method === "initialize") {
      res.writeHead(200, { "Content-Type": "application/json" }).end(JSON.stringify({
        jsonrpc: "2.0",
        result: { protocolVersion: "2024-11-05", capabilities: { tools: {} }, serverInfo: { name: "agent-hub-mcp", version: "1.0.0" } },
        id
      }));
      return;
    }

    // notifications (no id)
    if (method === "notifications/initialized" || method === "initialized") {
      res.writeHead(200, { "Content-Type": "application/json" }).end(JSON.stringify({ jsonrpc: "2.0", result: {} }));
      return;
    }

    res.writeHead(400, { "Content-Type": "application/json" }).end(JSON.stringify({ jsonrpc: "2.0", error: { code: -32601, message: `Method not found: ${method}` }, id }));
  });
}).listen(PORT, () => {
  console.log(`MCP HTTP server listening on port ${PORT}`);
});
