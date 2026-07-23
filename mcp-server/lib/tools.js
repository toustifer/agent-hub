import { getBaseUrl, resolveBusinessCode, requireApiKey } from "./config.js";
import { authHeaders, setAdminToken, getAdminToken } from "./auth.js";

const BASE_URL = () => getBaseUrl();

async function request(method, path, body, headers = {}) {
  const url = `${BASE_URL()}${path}`;
  const opts = { method, headers: { "Content-Type": "application/json", ...headers } };
  if (body) opts.body = JSON.stringify(body);
  const res = await fetch(url, opts);
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(`[${res.status}] ${data.message || JSON.stringify(data)}`);
  return data.data;
}

function workerHeaders(args) {
  const bc = resolveBusinessCode(args);
  const key = requireApiKey(args);
  return { "X-API-Key": key, "X-Business-Code": bc, business_code: bc };
}

/** Tools that do not require business_code. */
const NO_BC = new Set(["hub_login", "hub_list_my_businesses"]);

export const TOOLS = [
  {
    name: "hub_login",
    description:
      "Step 1: hub_login() to get a browser URL. Open it, log in, click Approve. Step 2: hub_login({code}) to finish. Token is saved to ~/.agent-hub/config.json.",
    inputSchema: {
      type: "object",
      properties: { code: { type: "string", description: "Device code to exchange (only needed in step 2)" } },
    },
  },
  {
    name: "hub_list_my_businesses",
    description: "List businesses I belong to (JWT user)",
    inputSchema: { type: "object", properties: {} },
  },
  {
    name: "hub_heartbeat",
    description: "Worker heartbeat",
    inputSchema: {
      type: "object",
      properties: {
        business_code: { type: "string" },
        worker_id: { type: "string" },
        version: { type: "string" },
        host: { type: "string" },
        pid: { type: "integer" },
      },
      required: ["worker_id", "version"],
    },
  },
  {
    name: "hub_acquire_lock",
    description: "Acquire distributed lock",
    inputSchema: {
      type: "object",
      properties: {
        business_code: { type: "string" },
        resource_key: { type: "string" },
        worker_id: { type: "string" },
        ttl_seconds: { type: "integer", default: 300 },
      },
      required: ["resource_key", "worker_id"],
    },
  },
  {
    name: "hub_release_lock",
    description: "Release lock",
    inputSchema: {
      type: "object",
      properties: { holder_token: { type: "string" } },
      required: ["holder_token"],
    },
  },
  {
    name: "hub_renew_lock",
    description: "Renew lock TTL",
    inputSchema: {
      type: "object",
      properties: { holder_token: { type: "string" }, ttl_seconds: { type: "integer", default: 300 } },
      required: ["holder_token"],
    },
  },
  {
    name: "hub_append_event",
    description: "Record event",
    inputSchema: {
      type: "object",
      properties: {
        business_code: { type: "string" },
        actor: { type: "string" },
        event_type: { type: "string" },
        payload: { type: "object" },
      },
      required: ["actor", "event_type"],
    },
  },
  {
    name: "hub_create_playbook",
    description: "Create playbook",
    inputSchema: {
      type: "object",
      properties: {
        business_code: { type: "string" },
        category: { type: "string" },
        title: { type: "string" },
        content: { type: "string" },
        tags: { type: "array", items: { type: "string" } },
        worker_id: { type: "string" },
      },
      required: ["category", "title", "content", "worker_id"],
    },
  },
  {
    name: "hub_search_playbooks",
    description: "Search playbooks",
    inputSchema: {
      type: "object",
      properties: {
        business_code: { type: "string" },
        query: { type: "string" },
        category: { type: "string" },
        limit: { type: "integer", default: 20 },
      },
      required: ["query"],
    },
  },
  {
    name: "hub_list_workers",
    description: "List workers",
    inputSchema: {
      type: "object",
      properties: { business_code: { type: "string" }, status: { type: "string" } },
    },
  },
  {
    name: "hub_list_locks",
    description: "List active locks",
    inputSchema: { type: "object", properties: { business_code: { type: "string" } } },
  },
  {
    name: "hub_list_events",
    description: "List events",
    inputSchema: {
      type: "object",
      properties: {
        business_code: { type: "string" },
        event_type: { type: "string" },
        limit: { type: "integer", default: 20 },
      },
    },
  },
  {
    name: "hub_add_repo",
    description: "Bind GitHub repo",
    inputSchema: {
      type: "object",
      properties: {
        business_code: { type: "string" },
        repo_url: { type: "string" },
        default_branch: { type: "string" },
      },
      required: ["repo_url"],
    },
  },
  {
    name: "hub_sync_dag",
    description: "Sync DAG task",
    inputSchema: {
      type: "object",
      properties: {
        business_code: { type: "string" },
        task_id: { type: "string" },
        title: { type: "string" },
        status: { type: "string" },
        assigned_worker: { type: "string" },
      },
      required: ["task_id", "title", "status"],
    },
  },
  {
    name: "hub_get_dag",
    description: "Get DAG tasks",
    inputSchema: { type: "object", properties: { business_code: { type: "string" } } },
  },
  {
    name: "hub_invite_member",
    description: "Invite a member by email (returns invite_url with one-time token)",
    inputSchema: {
      type: "object",
      properties: {
        business_code: { type: "string" },
        email: { type: "string" },
        role: { type: "string", default: "member" },
      },
      required: ["email"],
    },
  },
  {
    name: "hub_accept_invitation",
    description: "Accept invite by token (preferred) or join open business by code",
    inputSchema: {
      type: "object",
      properties: {
        token: { type: "string", description: "Invite token from invite_url" },
        business_code: { type: "string", description: "Only works when join_policy=open" },
      },
    },
  },
  {
    name: "hub_create_link_request",
    description: "Request to join a business (admin approval)",
    inputSchema: {
      type: "object",
      properties: { business_code: { type: "string" }, device_info: { type: "string" } },
      required: ["business_code"],
    },
  },
  {
    name: "hub_list_link_requests",
    description: "List pending link requests (admin only)",
    inputSchema: {
      type: "object",
      properties: { business_code: { type: "string" } },
      required: ["business_code"],
    },
  },
  {
    name: "hub_review_link_request",
    description: "Approve or reject a link request (admin only)",
    inputSchema: {
      type: "object",
      properties: {
        business_code: { type: "string" },
        request_id: { type: "integer" },
        action: { type: "string", enum: ["approve", "reject"] },
      },
      required: ["business_code", "request_id", "action"],
    },
  },
  {
    name: "hub_report_branches",
    description:
      "Report local git branch tips (+ optional task/dag/worker bindings) to Hub branch index. Soft-fail friendly; use after prepare/submit.",
    inputSchema: {
      type: "object",
      properties: {
        business_code: { type: "string" },
        repo_url: { type: "string" },
        reporter: { type: "string" },
        branches: {
          type: "array",
          items: {
            type: "object",
            properties: {
              name: { type: "string" },
              tip_sha: { type: "string" },
              source: { type: "string", description: "report|github_api|ls_remote" },
            },
            required: ["name"],
          },
        },
        bindings: {
          type: "array",
          items: {
            type: "object",
            properties: {
              bind_type: { type: "string", description: "dag|task|worker|user" },
              bind_id: { type: "string" },
              branch_name: { type: "string" },
              head_sha: { type: "string" },
              worktree_host: { type: "string" },
              status: { type: "string" },
            },
            required: ["bind_type", "bind_id", "branch_name"],
          },
        },
      },
    },
  },
  {
    name: "hub_list_branches",
    description: "List Hub branch index + who/what is bound to each branch for a business",
    inputSchema: {
      type: "object",
      properties: { business_code: { type: "string" } },
    },
  },
  {
    name: "hub_bind_branch",
    description: "Bind a dag|task|worker|user to a branch name on Hub",
    inputSchema: {
      type: "object",
      properties: {
        business_code: { type: "string" },
        bind_type: { type: "string" },
        bind_id: { type: "string" },
        branch_name: { type: "string" },
        head_sha: { type: "string" },
        worktree_host: { type: "string" },
        status: { type: "string" },
      },
      required: ["bind_type", "bind_id", "branch_name"],
    },
  },
  {
    name: "hub_refresh_branches",
    description: "Refresh branch tips from GitHub API (GITHUB_TOKEN) or git ls-remote; source wins over report",
    inputSchema: {
      type: "object",
      properties: {
        business_code: { type: "string" },
        repo_url: { type: "string" },
        default_only: { type: "boolean" },
      },
    },
  },
];

// Human-side tools: JWT only (authHeaders). Machine tools: workerHeaders (API key).
const MACHINE_TOOLS = new Set([
  "hub_heartbeat",
  "hub_acquire_lock",
  "hub_release_lock",
  "hub_renew_lock",
  "hub_append_event",
  "hub_create_playbook",
]);

export async function handleToolCall(name, args = {}) {
  let bc = null;
  let wh = null;

  switch (name) {
    case "hub_login": {
      if (args?.code) {
        const tr = await fetch(`${BASE_URL()}/v1/hub/auth/device/token?code=${encodeURIComponent(args.code)}`);
        const tj = await tr.json();
        if (!tj.data?.token) throw new Error("Code not yet approved. Open the URL, log in, and click Approve first.");
        setAdminToken(tj.data.token);
        return "Logged in! Token saved to ~/.agent-hub/config.json. All hub tools ready.";
      }
      const dr = await fetch(`${BASE_URL()}/v1/hub/auth/device`, { method: "POST" });
      const dj = await dr.json();
      if (!dr.ok) throw new Error(dj.message || "Failed to create device code");
      const url = dj.data.verification_url;
      const code = dj.data.code;
      return `Open this URL in browser (you must be logged in as a real user):\n\n  ${url}\n\nThen call hub_login({ code: "${code}" }) to finish.`;
    }

    case "hub_list_my_businesses": {
      const r = await fetch(`${BASE_URL()}/v1/hub/me/businesses`, { headers: authHeaders() });
      const list = (await r.json()).data || [];
      return list.map((b) => `${b.code} — ${b.name} [${b.role}]`).join("\n") || "No businesses.";
    }

    case "hub_heartbeat": {
      const headers = workerHeaders(args);
      bc = headers.business_code;
      wh = { "X-API-Key": headers["X-API-Key"], "X-Business-Code": headers["X-Business-Code"] };
      await request(
        "POST",
        "/v1/hub/workers/heartbeat",
        {
          business_code: bc,
          worker_id: args.worker_id,
          version: args.version,
          host: args.host || "mcp",
          pid: args.pid || 0,
        },
        wh
      );
      return `Worker ${args.worker_id} online.`;
    }

    case "hub_acquire_lock": {
      const headers = workerHeaders(args);
      bc = headers.business_code;
      wh = { "X-API-Key": headers["X-API-Key"], "X-Business-Code": headers["X-Business-Code"] };
      const d = await request(
        "POST",
        "/v1/hub/locks/acquire",
        {
          business_code: bc,
          resource_key: args.resource_key,
          worker_id: args.worker_id,
          ttl_seconds: args.ttl_seconds || 300,
        },
        wh
      );
      return `Lock acquired. Token: ${d.holder_token}`;
    }

    case "hub_release_lock": {
      // worker path — needs any valid api key; try resolve, else error
      const headers = workerHeaders(args.business_code ? args : { ...args, business_code: resolveBusinessCode(args) });
      wh = { "X-API-Key": headers["X-API-Key"], "X-Business-Code": headers["X-Business-Code"] };
      await request("POST", "/v1/hub/locks/release", { holder_token: args.holder_token }, wh);
      return "Released.";
    }

    case "hub_renew_lock": {
      const headers = workerHeaders(args.business_code ? args : { ...args, business_code: resolveBusinessCode(args) });
      wh = { "X-API-Key": headers["X-API-Key"], "X-Business-Code": headers["X-Business-Code"] };
      await request(
        "POST",
        "/v1/hub/locks/renew",
        { holder_token: args.holder_token, ttl_seconds: args.ttl_seconds || 300 },
        wh
      );
      return "Renewed.";
    }

    case "hub_append_event": {
      const headers = workerHeaders(args);
      bc = headers.business_code;
      wh = { "X-API-Key": headers["X-API-Key"], "X-Business-Code": headers["X-Business-Code"] };
      const d = await request(
        "POST",
        "/v1/hub/events",
        {
          business_code: bc,
          actor: args.actor,
          event_type: args.event_type,
          payload: args.payload || {},
        },
        wh
      );
      return `Event ${d.id} recorded.`;
    }

    case "hub_create_playbook": {
      const headers = workerHeaders(args);
      bc = headers.business_code;
      wh = { "X-API-Key": headers["X-API-Key"], "X-Business-Code": headers["X-Business-Code"] };
      const d = await request(
        "POST",
        "/v1/hub/playbooks",
        {
          business_code: bc,
          category: args.category,
          title: args.title,
          content: args.content,
          tags: args.tags || [],
          worker_id: args.worker_id,
        },
        wh
      );
      return `Playbook ${d.id} created.`;
    }

    case "hub_search_playbooks": {
      // Prefer JWT; fall back to machine key if present
      let headers;
      try {
        headers = authHeaders();
      } catch {
        const whd = workerHeaders(args);
        headers = { "X-API-Key": whd["X-API-Key"], "X-Business-Code": whd["X-Business-Code"] };
      }
      const p = new URLSearchParams({ q: args.query, limit: String(args.limit || 20) });
      if (args.category) p.set("category", args.category);
      const r = await fetch(`${BASE_URL()}/v1/hub/playbooks/search?${p}`, { headers });
      const list = (await r.json()).data || [];
      return list.length ? list.map((x) => `[${x.category}] ${x.title}`).join("\n") : "No results.";
    }

    case "hub_list_workers": {
      bc = resolveBusinessCode(args);
      const r = await fetch(
        `${BASE_URL()}/v1/hub/workers?${new URLSearchParams(bc ? { business: bc } : {})}`,
        { headers: authHeaders() }
      );
      const list = (await r.json()).data || [];
      return list.map((w) => `${w.worker_id} ${w.status}`).join("\n") || "No workers.";
    }

    case "hub_list_locks": {
      bc = resolveBusinessCode(args);
      const r = await fetch(
        `${BASE_URL()}/v1/hub/locks?${new URLSearchParams(bc ? { business: bc } : {})}`,
        { headers: authHeaders() }
      );
      const list = (await r.json()).data || [];
      return list.map((l) => `${l.resource_key} ${l.holder_worker_id}`).join("\n") || "No locks.";
    }

    case "hub_list_events": {
      bc = resolveBusinessCode(args);
      const p = new URLSearchParams({ business: bc, limit: String(args.limit || 20) });
      if (args.event_type) p.set("type", args.event_type);
      const r = await fetch(`${BASE_URL()}/v1/hub/events?${p}`, { headers: authHeaders() });
      const list = (await r.json()).data || [];
      return list.map((e) => `${(e.created_at || "").slice(0, 19)} ${e.event_type}`).join("\n") || "No events.";
    }

    case "hub_add_repo": {
      bc = resolveBusinessCode(args);
      await request(
        "POST",
        `/v1/hub/repos/${bc}`,
        { repo_url: args.repo_url, default_branch: args.default_branch || "main" },
        authHeaders()
      );
      return `Repo ${args.repo_url} bound.`;
    }

    case "hub_sync_dag": {
      bc = resolveBusinessCode(args);
      await request(
        "POST",
        `/v1/hub/dag/${bc}`,
        {
          task_id: args.task_id,
          title: args.title,
          status: args.status,
          assigned_worker: args.assigned_worker || "",
        },
        authHeaders()
      );
      return `DAG ${args.task_id} → ${args.status}`;
    }

    case "hub_get_dag": {
      bc = resolveBusinessCode(args);
      const r = await fetch(`${BASE_URL()}/v1/hub/dag/${bc}`, { headers: authHeaders() });
      const list = (await r.json()).data || [];
      return list.map((t) => `${t.status === "completed" ? "✓" : "⏳"} ${t.task_id} ${t.title}`).join("\n") || "No tasks.";
    }

    case "hub_invite_member": {
      bc = resolveBusinessCode(args);
      const d = await request(
        "POST",
        `/v1/hub/businesses/${bc}/invite`,
        { email: args.email, role: args.role || "member" },
        authHeaders()
      );
      return `Invitation for ${args.email}. Share (token once): ${d.invite_url}`;
    }

    case "hub_accept_invitation": {
      if (args.token) {
        await request("POST", "/v1/hub/invites/accept", { token: args.token }, authHeaders());
        return "Invite accepted.";
      }
      if (!args.business_code) {
        throw new Error("Provide token (preferred) or business_code (only if join_policy=open)");
      }
      bc = args.business_code;
      await request("POST", `/v1/hub/businesses/${bc}/join`, {}, authHeaders());
      return `Joined business: ${bc}`;
    }

    case "hub_create_link_request": {
      bc = resolveBusinessCode(args);
      await request(
        "POST",
        `/v1/hub/businesses/${bc}/link-requests`,
        { device_info: args.device_info || "" },
        authHeaders()
      );
      return `Link request created for ${bc}. An admin must approve it.`;
    }

    case "hub_list_link_requests": {
      bc = resolveBusinessCode(args);
      const r = await fetch(`${BASE_URL()}/v1/hub/businesses/${bc}/link-requests`, { headers: authHeaders() });
      const list = (await r.json()).data || [];
      return list.length
        ? list.map((lr) => `[${lr.id}] ${lr.name || lr.email} — ${(lr.created_at || "").slice(0, 16)}`).join("\n")
        : "No pending link requests.";
    }

    case "hub_review_link_request": {
      bc = resolveBusinessCode(args);
      await request(
        "POST",
        `/v1/hub/businesses/${bc}/link-requests/${args.request_id}/review`,
        { action: args.action },
        authHeaders()
      );
      return `Link request ${args.request_id} ${args.action}d.`;
    }

    case "hub_report_branches": {
      bc = resolveBusinessCode(args);
      const d = await request(
        "POST",
        `/v1/hub/repos/${bc}/branches/report`,
        {
          repo_url: args.repo_url || "",
          reporter: args.reporter || "",
          branches: args.branches || [],
          bindings: args.bindings || [],
        },
        authHeaders()
      );
      return `Branches upserted=${d.branches_upserted || 0} bindings=${d.bindings_upserted || 0} repo_id=${d.repo_id || "?"}`;
    }

    case "hub_list_branches": {
      bc = resolveBusinessCode(args);
      const r = await fetch(`${BASE_URL()}/v1/hub/repos/${bc}/branches`, { headers: authHeaders() });
      const body = await r.json().catch(() => ({}));
      if (!r.ok) throw new Error(`[${r.status}] ${body.message || JSON.stringify(body)}`);
      const data = body.data || {};
      const branches = data.branches || [];
      const bindings = data.bindings || [];
      const lines = branches.map(
        (b) =>
          `${b.name} @ ${(b.tip_sha || "").slice(0, 8) || "--------"} [${b.source || "?"}]${b.is_default ? " (default)" : ""}`
      );
      const bindLines = bindings.map(
        (b) => `  bind ${b.bind_type}:${b.bind_id} → ${b.branch_name} [${b.status}]`
      );
      return (
        (lines.length ? lines.join("\n") : "No branches reported.") +
        (bindLines.length ? "\nBindings:\n" + bindLines.join("\n") : "")
      );
    }

    case "hub_bind_branch": {
      bc = resolveBusinessCode(args);
      await request(
        "POST",
        `/v1/hub/repos/${bc}/branches/bind`,
        {
          bind_type: args.bind_type,
          bind_id: args.bind_id,
          branch_name: args.branch_name,
          head_sha: args.head_sha || "",
          worktree_host: args.worktree_host || "",
          status: args.status || "active",
        },
        authHeaders()
      );
      return `Bound ${args.bind_type}:${args.bind_id} → ${args.branch_name}`;
    }

    case "hub_refresh_branches": {
      bc = resolveBusinessCode(args);
      const d = await request(
        "POST",
        `/v1/hub/repos/${bc}/branches/refresh`,
        { repo_url: args.repo_url || "", default_only: !!args.default_only },
        authHeaders()
      );
      return `Refreshed branches_upserted=${d.branches_upserted || 0} source=${d.source || "?"} repo=${d.repo_url || ""}`;
    }

    default:
      throw new Error(`Unknown tool: ${name}`);
  }
}

// re-export for shells that want to check login state
export { getAdminToken };
