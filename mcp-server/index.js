#!/usr/bin/env node
import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { CallToolRequestSchema, ListToolsRequestSchema } from "@modelcontextprotocol/sdk/types.js";

const BASE_URL = process.env.HUB_API_URL || "https://hub.stifer.xyz";
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
  if (!adminToken) throw new Error("Not logged in. Call hub_login() first to get an auth URL.");
  return { Authorization: `Bearer ${adminToken}` };
}

const TOOLS = [
  {
    name: "hub_login",
    description: "Step 1: hub_login() to get a browser URL. Open it, click Approve. Step 2: hub_login({code}) to finish.",
    inputSchema: {
      type: "object",
      properties: { code: { type: "string", description: "Device code to exchange (only needed in step 2)" } }
    }
  },
  { name: "hub_heartbeat",       description: "Worker heartbeat", inputSchema: { type: "object", properties: { worker_id:{type:"string"}, version:{type:"string"}, host:{type:"string"}, pid:{type:"integer"} }, required:["worker_id","version"] } },
  { name: "hub_acquire_lock",    description: "Acquire distributed lock", inputSchema: { type: "object", properties: { resource_key:{type:"string"}, worker_id:{type:"string"}, ttl_seconds:{type:"integer",default:300} }, required:["resource_key","worker_id"] } },
  { name: "hub_release_lock",    description: "Release lock",           inputSchema: { type: "object", properties: { holder_token:{type:"string"} }, required:["holder_token"] } },
  { name: "hub_renew_lock",      description: "Renew lock TTL",         inputSchema: { type: "object", properties: { holder_token:{type:"string"}, ttl_seconds:{type:"integer",default:300} }, required:["holder_token"] } },
  { name: "hub_append_event",    description: "Record event",           inputSchema: { type: "object", properties: { actor:{type:"string"}, event_type:{type:"string"}, payload:{type:"object"} }, required:["actor","event_type"] } },
  { name: "hub_create_playbook", description: "Create playbook",        inputSchema: { type: "object", properties: { category:{type:"string"}, title:{type:"string"}, content:{type:"string"}, tags:{type:"array",items:{type:"string"}}, worker_id:{type:"string"} }, required:["category","title","content","worker_id"] } },
  { name: "hub_search_playbooks",description: "Search playbooks",       inputSchema: { type: "object", properties: { query:{type:"string"}, category:{type:"string"}, limit:{type:"integer",default:20} }, required:["query"] } },
  { name: "hub_list_workers",    description: "List workers",           inputSchema: { type: "object", properties: { business_code:{type:"string"}, status:{type:"string"} } } },
  { name: "hub_list_locks",      description: "List active locks",      inputSchema: { type: "object", properties: { business_code:{type:"string"} } } },
  { name: "hub_list_events",     description: "List events",            inputSchema: { type: "object", properties: { business_code:{type:"string"}, event_type:{type:"string"}, limit:{type:"integer",default:20} }, required:["business_code"] } },
  { name: "hub_add_repo",        description: "Bind GitHub repo",       inputSchema: { type: "object", properties: { repo_url:{type:"string"}, default_branch:{type:"string"} }, required:["repo_url"] } },
  { name: "hub_sync_dag",        description: "Sync DAG task",          inputSchema: { type: "object", properties: { task_id:{type:"string"}, title:{type:"string"}, status:{type:"string"}, assigned_worker:{type:"string"} }, required:["task_id","title","status"] } },
  { name: "hub_get_dag",         description: "Get DAG tasks",          inputSchema: { type: "object", properties: {} } },
];

const server = new Server({ name: "agent-hub-mcp", version: "1.0.0" }, { capabilities: { tools: {} } });
server.setRequestHandler(ListToolsRequestSchema, async () => ({ tools: TOOLS }));

server.setRequestHandler(CallToolRequestSchema, async (req) => {
  const { name, arguments: args } = req.params;
  const bc = args?.business_code || "ai-medbox";
  const wh = { "X-API-Key": "agent-company-worker", "X-Business-Code": bc };

  try {
    switch (name) {
      case "hub_login": {
        if (args?.code) {
          const tr = await fetch(`${BASE_URL}/v1/hub/auth/device/token?code=${args.code}`);
          const tj = await tr.json();
          if (!tj.data?.token) throw new Error("Code not yet approved. Open the URL and click Approve first.");
          adminToken = tj.data.token;
          return { content: [{ type: "text", text: "Logged in! All hub tools ready." }] };
        }
        const dr = await fetch(`${BASE_URL}/v1/hub/auth/device`, { method: "POST" });
        const dj = await dr.json();
        if (!dr.ok) throw new Error(dj.message || "Failed");
        const url = dj.data.verification_url;
        const code = dj.data.code;
        return { content: [{ type: "text", text: `Open this URL in browser:\n\n  ${url}\n\nThen call hub_login({ code: "${code}" }) to finish.` }] };
      }

      case "hub_heartbeat": {
        await request("POST", "/v1/hub/workers/heartbeat", { business_code: bc, worker_id: args.worker_id, version: args.version, host: args.host||"mcp", pid: args.pid||0 }, wh);
        return { content: [{ type: "text", text: `Worker ${args.worker_id} online.` }] };
      }
      case "hub_acquire_lock": {
        const d = await request("POST", "/v1/hub/locks/acquire", { business_code: bc, resource_key: args.resource_key, worker_id: args.worker_id, ttl_seconds: args.ttl_seconds||300 }, wh);
        return { content: [{ type: "text", text: `Lock acquired. Token: ${d.holder_token}` }] };
      }
      case "hub_release_lock": { await request("POST", "/v1/hub/locks/release", { holder_token: args.holder_token }, wh); return { content: [{ type: "text", text: "Released." }] }; }
      case "hub_renew_lock":   { await request("POST", "/v1/hub/locks/renew", { holder_token: args.holder_token, ttl_seconds: args.ttl_seconds||300 }, wh); return { content: [{ type: "text", text: "Renewed." }] }; }
      case "hub_append_event": { const d = await request("POST", "/v1/hub/events", { business_code: bc, actor: args.actor, event_type: args.event_type, payload: args.payload||{} }, wh); return { content: [{ type: "text", text: `Event ${d.id} recorded.` }] }; }
      case "hub_create_playbook": { const d = await request("POST", "/v1/hub/playbooks", { business_code: bc, category: args.category, title: args.title, content: args.content, tags: args.tags||[], worker_id: args.worker_id }, wh); return { content: [{ type: "text", text: `Playbook ${d.id} created.` }] }; }

      case "hub_search_playbooks": {
        const p = new URLSearchParams({ q: args.query, limit: String(args.limit||20) });
        if (args.category) p.set("category", args.category);
        const r = await fetch(`${BASE_URL}/v1/hub/playbooks/search?${p}`, { headers: wh });
        const list = (await r.json()).data || [];
        return { content: [{ type: "text", text: list.length ? list.map(x => `[${x.category}] ${x.title}`).join("\n") : "No results." }] };
      }

      case "hub_list_workers": { const r = await fetch(`${BASE_URL}/v1/hub/workers?${new URLSearchParams(args.business_code?{business:args.business_code}:{})}`, { headers: auth() }); const list = (await r.json()).data||[]; return { content: [{ type: "text", text: list.map(w=>`${w.worker_id} ${w.status}`).join("\n")||"No workers." }] }; }
      case "hub_list_locks":   { const r = await fetch(`${BASE_URL}/v1/hub/locks?${new URLSearchParams(args.business_code?{business:args.business_code}:{})}`, { headers: auth() }); const list = (await r.json()).data||[]; return { content: [{ type: "text", text: list.map(l=>`${l.resource_key} ${l.holder_worker_id}`).join("\n")||"No locks." }] }; }
      case "hub_list_events":  { const p = new URLSearchParams({ business: args.business_code, limit: String(args.limit||20) }); if(args.event_type) p.set("type",args.event_type); const r = await fetch(`${BASE_URL}/v1/hub/events?${p}`, { headers: auth() }); const list = (await r.json()).data||[]; return { content: [{ type: "text", text: list.map(e=>`${(e.created_at||"").slice(0,19)} ${e.event_type}`).join("\n")||"No events." }] }; }
      case "hub_add_repo":    { await request("POST", `/v1/hub/repos/${bc}`, { repo_url: args.repo_url, default_branch: args.default_branch||"main" }, auth()); return { content: [{ type: "text", text: `Repo ${args.repo_url} bound.` }] }; }
      case "hub_sync_dag":    { await request("POST", `/v1/hub/dag/${bc}`, { task_id: args.task_id, title: args.title, status: args.status, assigned_worker: args.assigned_worker||"" }, auth()); return { content: [{ type: "text", text: `DAG ${args.task_id} → ${args.status}` }] }; }
      case "hub_get_dag":     { const r = await fetch(`${BASE_URL}/v1/hub/dag/${bc}`, { headers: auth() }); const list = (await r.json()).data||[]; return { content: [{ type: "text", text: list.map(t=>`${t.status==="completed"?"✓":"⏳"} ${t.task_id} ${t.title}`).join("\n")||"No tasks." }] }; }

      default: return { content: [{ type: "text", text: `Unknown tool: ${name}` }], isError: true };
    }
  } catch (err) {
    return { content: [{ type: "text", text: err.message }], isError: true };
  }
});

const transport = new StdioServerTransport();
await server.connect(transport);
