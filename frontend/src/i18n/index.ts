import { ref } from 'vue'

const locale = ref(localStorage.getItem('hub-locale') || 'zh')

const msgs: Record<string, Record<string, string>> = {
  // MainLayout
  'app.name': { zh: 'Agent Hub', en: 'Agent Hub' },
  'nav.teams': { zh: '我的团队', en: 'My Teams' },
  'nav.logout': { zh: '退出登录', en: 'Logout' },

  // Login
  'login.title': { zh: '登录 Agent Hub', en: 'Sign in to Agent Hub' },
  'login.email': { zh: '邮箱', en: 'Email' },
  'login.password': { zh: '密码', en: 'Password' },
  'login.login': { zh: '登录', en: 'Login' },
  'login.register': { zh: '注册', en: 'Register' },
  'login.createAccount': { zh: '创建账号', en: 'Create Account' },
  'login.switchToLogin': { zh: '已有账号？登录', en: 'Already have an account? Sign in' },
  'login.switchToRegister': { zh: '没有账号？创建', en: 'No account? Create one' },
  'login.requestFailed': { zh: '请求失败', en: 'Request failed' },

  // Dashboard
  'dash.title': { zh: '我的团队', en: 'My Teams' },
  'dash.newTeam': { zh: '+ 新建团队', en: '+ New Team' },
  'dash.noTeams': { zh: '暂无团队，点击创建！', en: 'No teams yet. Create one!' },
  'dash.code': { zh: '标识码', en: 'Code' },
  'dash.name': { zh: '名称', en: 'Name' },
  'dash.desc': { zh: '描述', en: 'Description' },
  'dash.codePlaceholder': { zh: '如: my-project', en: 'e.g. my-project' },
  'dash.namePlaceholder': { zh: '团队名称', en: 'Team name' },
  'dash.cancel': { zh: '取消', en: 'Cancel' },
  'dash.create': { zh: '创建', en: 'Create' },
  'dash.created': { zh: '团队创建成功！', en: 'Team Created!' },
  'dash.saveKey': { zh: '请保存此 API Key，仅显示一次', en: 'Save this API key — it won\'t be shown again' },
  'dash.connectInfo': { zh: '在终端或 CI 中设置环境变量 HUB_API_KEY 来连接此团队。', en: 'Set the HUB_API_KEY env var in your terminal or CI to connect this team.' },
  'dash.viewDetail': { zh: '查看详情', en: 'View' },
  'dash.done': { zh: '完成', en: 'Done' },

  // TeamPage
  'team.overview': { zh: '概览', en: 'Overview' },
  'team.totalWorkers': { zh: 'Worker总数', en: 'Workers' },
  'team.onlineWorkers': { zh: '在线', en: 'Online' },
  'team.activeLocks': { zh: '活跃锁', en: 'Locks' },
  'team.pendingTasks': { zh: '待处理任务', en: 'Pending' },
  'team.recentActivity': { zh: '最近动态', en: 'Recent Activity' },
  'team.workerList': { zh: 'Worker 列表', en: 'Worker List' },
  'team.noDescription': { zh: '暂无描述', en: 'No description' },
  'team.workers': { zh: 'Workers', en: 'Workers' },
  'team.locks': { zh: '分布式锁', en: 'Locks' },
  'team.playbooks': { zh: '经验库', en: 'Playbooks' },
  'team.events': { zh: '事件流', en: 'Events' },
  'team.workerId': { zh: 'Worker ID', en: 'Worker ID' },
  'team.version': { zh: '版本', en: 'Version' },
  'team.owner': { zh: '持有人', en: 'Owner' },
  'team.host': { zh: '主机', en: 'Host' },
  'team.status': { zh: '状态', en: 'Status' },
  'team.lastHeartbeat': { zh: '最后心跳', en: 'Last Heartbeat' },
  'team.resource': { zh: '资源', en: 'Resource' },
  'team.holder': { zh: '持有者', en: 'Holder' },
  'team.acquired': { zh: '获取时间', en: 'Acquired' },
  'team.expires': { zh: '过期时间', en: 'Expires' },
  'team.title': { zh: '标题', en: 'Title' },
  'team.category': { zh: '分类', en: 'Category' },
  'team.tags': { zh: '标签', en: 'Tags' },
  'team.author': { zh: '作者', en: 'Author' },
  'team.eventType': { zh: '事件类型', en: 'Event Type' },
  'team.actor': { zh: '执行者', en: 'Actor' },
  'team.workerDetail': { zh: 'Worker 详情', en: 'Worker Detail' },
  'team.pid': { zh: 'PID', en: 'PID' },
  'team.playbooksCap': { zh: '经验库', en: 'Playbooks' },
  'team.recentEvents': { zh: '最近事件', en: 'Recent Events' },
  'team.noPlaybooks': { zh: '暂无经验记录。从 .mycompany/ 同步 experience.json 后这里会显示内容。', en: 'No playbooks yet. Sync experience.json from .mycompany/' },
  'team.noEvents': { zh: '暂无事件。', en: 'No events yet.' },
  'team.dag': { zh: '任务DAG', en: 'Tasks' },
  'team.taskId': { zh: '任务ID', en: 'Task ID' },
  'team.taskTitle': { zh: '标题', en: 'Title' },
  'team.taskStatus': { zh: '状态', en: 'Status' },
  'team.taskWorker': { zh: '指派', en: 'Assigned' },
  'team.handbook': { zh: '业务手册', en: 'Handbook' },
  'team.businessFlow': { zh: '核心链路', en: 'Core Flow' },
  'team.codeMap': { zh: '代码地图', en: 'Code Map' },
  'team.dangerZones': { zh: '高危区域', en: 'Danger Zones' },
  'team.noDag': { zh: '暂无任务', en: 'No tasks yet' },
  'team.search': { zh: '搜索...', en: 'Search...' },
  'team.filterBiz': { zh: '按业务筛选', en: 'Filter by business' },
  'team.searchPb': { zh: '搜索经验库...', en: 'Search playbooks...' },

  // BusinessList
  'biz.filterStatus': { zh: '按状态筛选', en: 'Filter by status' },
  'biz.active': { zh: '活跃', en: 'Active' },
  'biz.pending': { zh: '待处理', en: 'Pending' },
  'biz.suspended': { zh: '已暂停', en: 'Suspended' },
  'biz.new': { zh: '新建业务', en: 'New Business' },
  'biz.repoUrl': { zh: '仓库 URL', en: 'Repo URL' },
  'biz.ownerId': { zh: '所有者 ID', en: 'Owner User ID' },
  'biz.created': { zh: '创建时间', en: 'Created' },
  'biz.description': { zh: '描述', en: 'Description' },

  // DeviceAuth
  'device.title': { zh: '授权 Claude Code', en: 'Authorize Claude Code' },
  'device.desc': { zh: 'Claude Code 正在请求访问 Agent Hub 的权限。如果这是你自己发起的，请点击批准。', en: 'Claude Code is requesting access to Agent Hub. If you initiated this, please approve.' },
  'device.enterCode': { zh: '请在 Claude Code 中输入以下验证码：', en: 'Enter this code in Claude Code:' },
  'device.approve': { zh: '批准', en: 'Approve' },
  'device.deny': { zh: '拒绝', en: 'Deny' },
  'device.authorized': { zh: '已授权', en: 'Authorized' },
  'device.closePage': { zh: '授权成功，你可以关闭此页面。', en: 'You can close this page.' },
  'device.denied': { zh: '已拒绝', en: 'Denied' },
  'device.denyMsg': { zh: '授权已被拒绝。', en: 'Authorization was denied.' },

  // EventStream
  'event.bizPlaceholder': { zh: '业务标识', en: 'Business' },
  'event.connected': { zh: '已连接', en: 'Connected' },
  'event.disconnected': { zh: '已断开', en: 'Disconnected' },
  'event.pause': { zh: '暂停', en: 'Pause' },
  'event.resume': { zh: '恢复', en: 'Resume' },
  'event.waiting': { zh: '等待事件...', en: 'Waiting for events...' },

  // Community Marketplace
  'community.title': { zh: '社区市场', en: 'Community' },
  'community.search': { zh: '搜索 Worker...', en: 'Search workers...' },
  'community.domain': { zh: '领域筛选', en: 'Domain' },
  'community.allDomains': { zh: '全部领域', en: 'All Domains' },
  'community.sort': { zh: '排序', en: 'Sort' },
  'community.popular': { zh: '最热门', en: 'Most Popular' },
  'community.latest': { zh: '最新发布', en: 'Latest' },
  'community.installs': { zh: '次安装', en: 'installs' },
  'community.noWorkers': { zh: '暂无社区 Worker，成为第一个发布者！', en: 'No community workers yet. Be the first to publish!' },
  'community.install': { zh: '安装到我的项目', en: 'Install to My Project' },
  'community.selectProject': { zh: '选择目标项目', en: 'Select Target Project' },
  'community.installSuccess': { zh: '安装成功', en: 'Installed Successfully' },
  'community.publish': { zh: '发布到社区', en: 'Publish to Community' },
  'community.publishWorker': { zh: '发布 Worker 到社区', en: 'Publish Worker to Community' },
  'community.deidentify': { zh: '自动去敏（移除文件路径、URL等敏感信息）', en: 'Auto de-identify (remove file paths, URLs, etc.)' },
  'community.publisher': { zh: '发布者', en: 'Publisher' },
  'community.workerDetail': { zh: 'Worker 详情', en: 'Worker Detail' },
  'community.back': { zh: '← 返回市场', en: '← Back to Market' },
  'community.reviews': { zh: '评价', en: 'Reviews' },
  'community.noReviews': { zh: '暂无评价', en: 'No reviews yet' },
  'community.domainFilter': { zh: '领域', en: 'Domain' },
  'community.publishSuccess': { zh: '发布成功！', en: 'Published!' },
}

export function useI18n() {
  function t(key: string): string {
    const m = msgs[key]
    if (!m) return key
    return m[locale.value] || m.en || key
  }

  function setLocale(l: string) {
    locale.value = l
    localStorage.setItem('hub-locale', l)
  }

  return { locale, t, setLocale }
}
