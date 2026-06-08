'use strict';

/**
 * hub-client.js — agent-hub Node.js 客户端 SDK
 *
 * 纯 CommonJS 模块，零外部依赖（仅使用 Node.js 内置 http/https/crypto）。
 *
 * @example
 *   const { HubClient } = require('./hub-client');
 *   const hub = new HubClient({
 *     baseUrl: 'https://hub.stifer.xyz',
 *     apiKey: 'your-api-key',
 *     businessCode: 'siruoning',
 *   });
 *
 *   // Worker 心跳
 *   await hub.heartbeat({ worker_id: 'medication', version: '1.0.0', host: 'dev', pid: 1234 });
 *
 *   // 分布式锁
 *   const lock = await hub.acquireLock({ resource_key: 'task-1', worker_id: 'medication', ttl_seconds: 300 });
 *   await hub.renewLock({ holder_token: lock.holder_token, ttl_seconds: 300 });
 *   await hub.releaseLock({ holder_token: lock.holder_token });
 *
 *   // Playbook 知识库
 *   await hub.createPlaybook({ category: 'patterns', title: 'My Pattern', content: '# Content', tags: ['api'] });
 *   const results = await hub.searchPlaybooks({ q: 'pattern', limit: 10 });
 */

const https = require('https');
const http = require('http');
const crypto = require('crypto');

class HubClient {
  /**
   * @param {Object} options
   * @param {string} options.baseUrl    — hub 服务地址，例如 https://hub.stifer.xyz
   * @param {string} [options.apiKey]   — Worker 端 API Key（同时设置 X-API-Key 头）
   * @param {string} [options.businessCode] — 业务代号（同时设置 X-Business-Code 头）
   * @param {string} [options.token]    — JWT token（同时设置 Authorization: Bearer 头）
   */
  constructor({ baseUrl, apiKey, businessCode, token } = {}) {
    this.baseUrl = (baseUrl || '').replace(/\/+$/, '');
    this.apiKey = apiKey || '';
    this.businessCode = businessCode || '';
    this.token = token || '';
  }

  // ---------------------------------------------------------------------------
  // 内部工具方法
  // ---------------------------------------------------------------------------

  /**
   * 生成 UUID v4 字符串，可用于 holder_token 建议值等场景。
   * @returns {string}
   */
  _generateToken() {
    return crypto.randomUUID();
  }

  /**
   * 发起 HTTP 请求。
   *
   * @param {string} method  — HTTP 方法
   * @param {string} path    — 请求路径（含 query string）
   * @param {Object} [body]  — JSON 请求体
   * @returns {Promise<Object>} 解析后的响应 data 字段
   */
  _request(method, path, body) {
    return new Promise((resolve, reject) => {
      const fullUrl = new URL(path, this.baseUrl || 'http://localhost');
      const isHttps = fullUrl.protocol === 'https:';
      const transport = isHttps ? https : http;

      const headers = {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      };
      if (this.token) {
        headers['Authorization'] = `Bearer ${this.token}`;
      }
      if (this.apiKey) {
        headers['X-API-Key'] = this.apiKey;
      }
      if (this.businessCode) {
        headers['X-Business-Code'] = this.businessCode;
      }

      const options = {
        hostname: fullUrl.hostname,
        port: fullUrl.port || (isHttps ? 443 : 80),
        path: fullUrl.pathname + fullUrl.search,
        method,
        headers,
      };

      const req = transport.request(options, (res) => {
        let data = '';
        res.on('data', (chunk) => { data += chunk; });
        res.on('end', () => {
          let parsed;
          try {
            parsed = JSON.parse(data);
          } catch (_e) {
            parsed = { message: data };
          }

          if (res.statusCode >= 200 && res.statusCode < 300) {
            // 服务端统一用 { data: ... } 包裹响应
            resolve(parsed.data !== undefined ? parsed.data : parsed);
          } else if (res.statusCode === 409) {
            const err = new Error(parsed.message || 'Lock conflict');
            err.code = 409;
            err.statusCode = 409;
            err.response = parsed;
            reject(err);
          } else {
            const err = new Error(parsed.message || `HTTP ${res.statusCode}`);
            err.code = res.statusCode;
            err.statusCode = res.statusCode;
            err.response = parsed;
            reject(err);
          }
        });
      });

      req.on('error', (err) => {
        reject(new Error(`Network error: ${err.message}`));
      });

      if (body !== undefined && body !== null) {
        req.write(JSON.stringify(body));
      }
      req.end();
    });
  }

  // ---------------------------------------------------------------------------
  // Worker 心跳
  // ---------------------------------------------------------------------------

  /**
   * 上报 Worker 心跳。
   * @param {Object} params
   * @param {string} params.worker_id
   * @param {string} params.version
   * @param {string} params.host
   * @param {number} params.pid
   * @returns {Promise<Object>}
   */
  heartbeat({ worker_id, version, host, pid }) {
    return this._request('POST', '/v1/hub/workers/heartbeat', {
      business_code: this.businessCode,
      worker_id,
      version,
      host,
      pid,
    });
  }

  // ---------------------------------------------------------------------------
  // 分布式锁
  // ---------------------------------------------------------------------------

  /**
   * 获取分布式锁。
   * @param {Object} params
   * @param {string} params.resource_key
   * @param {string} params.worker_id
   * @param {number} [params.ttl_seconds=300]
   * @returns {Promise<{holder_token: string, expires_at: string}>}
   * @throws {Error} 409 — 锁已被其他 worker 持有
   */
  acquireLock({ resource_key, worker_id, ttl_seconds }) {
    return this._request('POST', '/v1/hub/locks/acquire', {
      business_code: this.businessCode,
      resource_key,
      worker_id,
      ttl_seconds: ttl_seconds || 300,
    });
  }

  /**
   * 续期分布式锁。
   * @param {Object} params
   * @param {string} params.holder_token
   * @param {number} [params.ttl_seconds=300]
   * @returns {Promise<Object>}
   */
  renewLock({ holder_token, ttl_seconds }) {
    return this._request('POST', '/v1/hub/locks/renew', {
      holder_token,
      ttl_seconds: ttl_seconds || 300,
    });
  }

  /**
   * 释放分布式锁。
   * @param {Object} params
   * @param {string} params.holder_token
   * @returns {Promise<Object>}
   */
  releaseLock({ holder_token }) {
    return this._request('POST', '/v1/hub/locks/release', {
      holder_token,
    });
  }

  // ---------------------------------------------------------------------------
  // Playbook 知识库
  // ---------------------------------------------------------------------------

  /**
   * 创建一条 Playbook 记录。
   * @param {Object} params
   * @param {string} params.category
   * @param {string} params.title
   * @param {string} params.content
   * @param {string[]} [params.tags]
   * @param {string} [params.worker_id]
   * @returns {Promise<Object>}
   */
  createPlaybook({ category, title, content, tags, worker_id }) {
    return this._request('POST', '/v1/hub/playbooks', {
      business_code: this.businessCode,
      category,
      title,
      content,
      tags: tags || [],
      worker_id: worker_id || this.businessCode,
    });
  }

  /**
   * 搜索 Playbook（支持全文检索 tsvector）。
   * @param {Object} [params]
   * @param {string} [params.q]         — 搜索关键词
   * @param {string} [params.category]  — 分类过滤
   * @param {number} [params.limit=20]
   * @param {number} [params.offset=0]
   * @returns {Promise<Object[]>}
   */
  searchPlaybooks({ q, category, limit, offset } = {}) {
    const params = new URLSearchParams();
    if (q) params.set('q', q);
    if (category) params.set('category', category);
    if (limit !== undefined) params.set('limit', limit);
    if (offset !== undefined) params.set('offset', offset);
    const qs = params.toString();
    return this._request('GET', `/v1/hub/playbooks/search${qs ? '?' + qs : ''}`);
  }

  /**
   * 根据 ID 获取单条 Playbook。
   * @param {number|string} id
   * @returns {Promise<Object>}
   */
  getPlaybook(id) {
    return this._request('GET', `/v1/hub/playbooks/${encodeURIComponent(id)}`);
  }

  // ---------------------------------------------------------------------------
  // 事件
  // ---------------------------------------------------------------------------

  /**
   * 追加一条事件记录。
   * @param {Object} params
   * @param {string} params.event_type
   * @param {string} [params.actor]     — 事件发起者，默认使用 businessCode
   * @param {Object} [params.payload]   — 事件携带的 JSON 数据
   * @returns {Promise<Object>}
   */
  appendEvent({ event_type, actor, payload }) {
    return this._request('POST', '/v1/hub/events', {
      business_code: this.businessCode,
      actor: actor || this.businessCode,
      event_type,
      payload: payload || {},
    });
  }

  // ---------------------------------------------------------------------------
  // Business
  // ---------------------------------------------------------------------------

  /**
   * 根据 code 获取业务详情。
   * @param {string} code
   * @returns {Promise<Object>}
   */
  getBusinessByCode(code) {
    return this._request('GET', `/v1/hub/businesses/${encodeURIComponent(code)}`);
  }
}

module.exports = { HubClient };
