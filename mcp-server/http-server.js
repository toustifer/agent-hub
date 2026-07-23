#!/usr/bin/env node
import http from "node:http";
import { TOOLS, handleToolCall } from "./lib/tools.js";

const PORT = parseInt(process.env.PORT || "9001", 10);

http
  .createServer(async (req, res) => {
    res.setHeader("Access-Control-Allow-Origin", "*");
    res.setHeader("Access-Control-Allow-Headers", "Content-Type, Authorization");
    res.setHeader("Access-Control-Allow-Methods", "POST, GET, OPTIONS");

    if (req.method === "OPTIONS") {
      res.writeHead(204).end();
      return;
    }
    if (req.method === "GET") {
      res.writeHead(200, { "Content-Type": "application/json" }).end(
        JSON.stringify({ name: "agent-hub-mcp", tools: TOOLS.map((t) => t.name) })
      );
      return;
    }
    if (req.method !== "POST") {
      res.writeHead(405).end(JSON.stringify({ error: "Method not allowed" }));
      return;
    }

    let body = "";
    for await (const chunk of req) body += chunk;

    let msg;
    try {
      msg = JSON.parse(body || "{}");
    } catch {
      res.writeHead(400).end(JSON.stringify({ error: "invalid json" }));
      return;
    }

    // Minimal JSON-RPC for tools/list and tools/call
    const id = msg.id ?? null;
    try {
      if (msg.method === "tools/list" || msg.method === "list_tools") {
        res.writeHead(200, { "Content-Type": "application/json" }).end(
          JSON.stringify({ jsonrpc: "2.0", id, result: { tools: TOOLS } })
        );
        return;
      }
      if (msg.method === "tools/call" || msg.method === "call_tool") {
        const name = msg.params?.name || msg.params?.tool;
        const args = msg.params?.arguments || msg.params?.args || {};
        const text = await handleToolCall(name, args);
        res.writeHead(200, { "Content-Type": "application/json" }).end(
          JSON.stringify({
            jsonrpc: "2.0",
            id,
            result: { content: [{ type: "text", text: String(text) }] },
          })
        );
        return;
      }
      // direct { name, arguments } shortcut
      if (msg.name) {
        const text = await handleToolCall(msg.name, msg.arguments || msg.args || {});
        res.writeHead(200, { "Content-Type": "application/json" }).end(JSON.stringify({ ok: true, text: String(text) }));
        return;
      }
      res.writeHead(400).end(JSON.stringify({ jsonrpc: "2.0", id, error: { message: "unknown method" } }));
    } catch (e) {
      res.writeHead(200, { "Content-Type": "application/json" }).end(
        JSON.stringify({
          jsonrpc: "2.0",
          id,
          error: { message: e.message },
        })
      );
    }
  })
  .listen(PORT, () => {
    console.error(`agent-hub MCP HTTP on :${PORT} (${TOOLS.length} tools)`);
  });
