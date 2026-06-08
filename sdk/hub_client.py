"""
hub_client.py — agent-hub Python 客户端 SDK

纯标准库实现（urllib + json），兼容 Python 3.8+。

@example
    from hub_client import HubClient, HubLockError

    hub = HubClient(
        base_url="https://hub.stifer.xyz",
        api_key="your-api-key",
        business_code="siruoning",
    )

    # Worker 心跳
    hub.heartbeat(worker_id="medication", version="1.0.0", host="dev", pid=1234)

    # 分布式锁
    lock = hub.acquire_lock(resource_key="task-1", worker_id="medication", ttl_seconds=300)
    hub.renew_lock(holder_token=lock["holder_token"], ttl_seconds=300)
    hub.release_lock(holder_token=lock["holder_token"])

    # Playbook 知识库
    hub.create_playbook(category="patterns", title="My Pattern", content="# Content", tags=["api"])
    results = hub.search_playbooks(q="pattern", limit=10)
"""

import json
import uuid
import urllib.request
import urllib.error
from urllib.parse import urlencode


class HubLockError(Exception):
    """分布式锁冲突异常（HTTP 409）。

    Attributes:
        resource_key: 冲突的资源键
        holder_worker_id: 当前持有锁的 worker 标识
        response: 完整的服务端响应字典
    """

    def __init__(self, message, resource_key=None, holder_worker_id=None, response=None):
        super().__init__(message)
        self.resource_key = resource_key
        self.holder_worker_id = holder_worker_id
        self.response = response

    def __repr__(self):
        return (
            f"HubLockError(message={self.args[0]!r}, "
            f"resource_key={self.resource_key!r}, "
            f"holder_worker_id={self.holder_worker_id!r})"
        )


class HubClient:
    """agent-hub Python 客户端。

    Args:
        base_url: hub 服务地址，例如 https://hub.stifer.xyz
        api_key: Worker 端 API Key（可选，设置 X-API-Key 请求头）
        business_code: 业务代号（可选，设置 X-Business-Code 请求头）
        token: JWT token（可选，设置 Authorization: Bearer 请求头）
    """

    def __init__(self, base_url, api_key=None, business_code=None, token=None):
        self.base_url = (base_url or "").rstrip("/")
        self.api_key = api_key or ""
        self.business_code = business_code or ""
        self.token = token or ""

    # ------------------------------------------------------------------
    # 内部工具方法
    # ------------------------------------------------------------------

    def _generate_token(self):
        """生成 UUID v4 字符串，可用于 holder_token 建议值等场景。"""
        return str(uuid.uuid4())

    def _request(self, method, path, body=None):
        """发起 HTTP 请求。

        Args:
            method: HTTP 方法（GET / POST 等）
            path: 请求路径（含 query string）
            body: JSON 可序列化的请求体字典

        Returns:
            解析后的响应 data 字段

        Raises:
            HubLockError: HTTP 409 锁冲突
            Exception: 其他 HTTP 错误或网络错误
        """
        url = self.base_url + path if self.base_url else path

        headers = {
            "Content-Type": "application/json",
            "Accept": "application/json",
        }
        if self.token:
            headers["Authorization"] = "Bearer " + self.token
        if self.api_key:
            headers["X-API-Key"] = self.api_key
        if self.business_code:
            headers["X-Business-Code"] = self.business_code

        data = json.dumps(body).encode("utf-8") if body is not None else None
        req = urllib.request.Request(url, data=data, headers=headers, method=method)

        try:
            with urllib.request.urlopen(req) as resp:
                raw = resp.read().decode("utf-8")
                parsed = json.loads(raw)
                # 服务端统一用 { data: ... } 包裹响应
                return parsed.get("data", parsed)
        except urllib.error.HTTPError as e:
            raw = e.read().decode("utf-8")
            try:
                parsed = json.loads(raw)
            except json.JSONDecodeError:
                parsed = {"message": raw}

            if e.code == 409:
                info = parsed.get("data", {}) if isinstance(parsed, dict) else {}
                raise HubLockError(
                    message=parsed.get("message", "Lock conflict") if isinstance(parsed, dict) else str(parsed),
                    resource_key=body.get("resource_key") if isinstance(body, dict) else None,
                    holder_worker_id=info.get("holder_worker_id"),
                    response=parsed,
                )
            msg = parsed.get("message", raw) if isinstance(parsed, dict) else str(parsed)
            raise Exception("HTTP %d: %s" % (e.code, msg))
        except urllib.error.URLError as e:
            raise Exception("Network error: %s" % str(e.reason))

    # ------------------------------------------------------------------
    # Worker 心跳
    # ------------------------------------------------------------------

    def heartbeat(self, worker_id, version, host, pid):
        """上报 Worker 心跳。

        Args:
            worker_id: Worker 标识
            version: Worker 版本号
            host: 运行主机标识
            pid: 进程 ID

        Returns:
            dict: Worker 信息
        """
        return self._request("POST", "/v1/hub/workers/heartbeat", {
            "business_code": self.business_code,
            "worker_id": worker_id,
            "version": version,
            "host": host,
            "pid": pid,
        })

    # ------------------------------------------------------------------
    # 分布式锁
    # ------------------------------------------------------------------

    def acquire_lock(self, resource_key, worker_id, ttl_seconds=300):
        """获取分布式锁。

        Args:
            resource_key: 资源标识
            worker_id: 请求锁的 worker 标识
            ttl_seconds: 锁存活时间（秒），默认 300

        Returns:
            dict: {"holder_token": str, "expires_at": str}

        Raises:
            HubLockError: 锁已被其他 worker 持有（HTTP 409）
        """
        return self._request("POST", "/v1/hub/locks/acquire", {
            "business_code": self.business_code,
            "resource_key": resource_key,
            "worker_id": worker_id,
            "ttl_seconds": ttl_seconds,
        })

    def renew_lock(self, holder_token, ttl_seconds=300):
        """续期分布式锁。

        Args:
            holder_token: acquire_lock 返回的 holder_token
            ttl_seconds: 续期时长（秒），默认 300

        Returns:
            dict: "ok" 确认
        """
        return self._request("POST", "/v1/hub/locks/renew", {
            "holder_token": holder_token,
            "ttl_seconds": ttl_seconds,
        })

    def release_lock(self, holder_token):
        """释放分布式锁。

        Args:
            holder_token: acquire_lock 返回的 holder_token

        Returns:
            dict: "ok" 确认
        """
        return self._request("POST", "/v1/hub/locks/release", {
            "holder_token": holder_token,
        })

    # ------------------------------------------------------------------
    # Playbook 知识库
    # ------------------------------------------------------------------

    def create_playbook(self, category, title, content, tags=None, worker_id=None):
        """创建一条 Playbook 记录。

        Args:
            category: 分类（如 patterns、solutions、troubleshooting）
            title: 标题
            content: Markdown 正文
            tags: 标签列表（可选）
            worker_id: 创建者 worker 标识（可选，默认使用 business_code）

        Returns:
            dict: 创建的 Playbook 记录
        """
        return self._request("POST", "/v1/hub/playbooks", {
            "business_code": self.business_code,
            "category": category,
            "title": title,
            "content": content,
            "tags": tags or [],
            "worker_id": worker_id or self.business_code,
        })

    def search_playbooks(self, q=None, category=None, limit=20, offset=0):
        """搜索 Playbook（支持全文检索 tsvector）。

        Args:
            q: 搜索关键词（可选）
            category: 分类过滤（可选）
            limit: 每页条数，默认 20
            offset: 偏移量，默认 0

        Returns:
            list[dict]: Playbook 列表
        """
        params = {}
        if q:
            params["q"] = q
        if category:
            params["category"] = category
        if limit:
            params["limit"] = str(limit)
        if offset:
            params["offset"] = str(offset)
        query = "?" + urlencode(params) if params else ""
        return self._request("GET", "/v1/hub/playbooks/search" + query)

    def get_playbook(self, id_):
        """根据 ID 获取单条 Playbook。

        Args:
            id_: Playbook ID

        Returns:
            dict: Playbook 详情
        """
        return self._request("GET", "/v1/hub/playbooks/%s" % id_)

    # ------------------------------------------------------------------
    # 事件
    # ------------------------------------------------------------------

    def append_event(self, event_type, actor=None, payload=None):
        """追加一条事件记录。

        Args:
            event_type: 事件类型（如 task_completed、deploy 等）
            actor: 事件发起者（可选，默认使用 business_code）
            payload: 事件携带的 JSON 数据（可选）

        Returns:
            dict: 创建的事件记录
        """
        return self._request("POST", "/v1/hub/events", {
            "business_code": self.business_code,
            "actor": actor or self.business_code,
            "event_type": event_type,
            "payload": payload or {},
        })

    # ------------------------------------------------------------------
    # Business
    # ------------------------------------------------------------------

    def get_business_by_code(self, code):
        """根据 code 获取业务详情。

        Args:
            code: 业务代号

        Returns:
            dict: Business 详情
        """
        return self._request("GET", "/v1/hub/businesses/%s" % code)
