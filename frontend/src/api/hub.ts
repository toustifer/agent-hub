import axios from 'axios'

const api = axios.create({
  baseURL: import.meta.env.VITE_HUB_API || 'https://hub.stifer.xyz',
  timeout: 10000,
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      if (window.location.pathname !== '/login') window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

export const auth = {
  register: (email: string, password: string) => api.post('/v1/hub/auth/register', { email, password }),
  login: (email: string, password: string) => api.post('/v1/hub/auth/login', { email, password }),
  verifyEmail: (token: string) => api.post('/v1/hub/auth/verify-email', { token }),
  resendVerification: (email: string) => api.post('/v1/hub/auth/resend-verification', { email }),
}
export const me = {
  getBusinesses: () => api.get('/v1/hub/me/businesses'),
  joinBusiness: (code: string, token?: string) =>
    api.post(`/v1/hub/businesses/${code}/join`, token ? { token } : {}),
}
export function getBusinesses(params?: any) { return api.get('/v1/hub/businesses', { params }) }
export function createBusiness(data: { name: string; description?: string; repo_url?: string }) {
  return api.post('/v1/hub/businesses', data)
}
export function patchBusinessProfile(code: string, data: { name?: string; description?: string }) {
  return api.patch(`/v1/hub/businesses/${encodeURIComponent(code)}/profile`, data)
}
export function getWorkers(params?: any) { return api.get('/v1/hub/workers', { params }) }
export function getLocks(params?: any) { return api.get('/v1/hub/locks', { params }) }
export function getEvents(params?: any) { return api.get('/v1/hub/events', { params }) }
export function searchPlaybooks(params?: any) { return api.get('/v1/hub/playbooks/search', { params }) }
export function getDAG(code: string) { return api.get(`/v1/hub/dag/${code}`) }
export function listTeamDocs(code: string) { return api.get(`/v1/hub/businesses/${code}/docs`) }
export function listWorkerTemplates(code: string) { return api.get(`/v1/hub/businesses/${code}/worker-templates`) }

// Community marketplace
export function listCommunityWorkers(params?: any) { return api.get('/v1/hub/community/workers', { params }) }
export function getCommunityWorker(id: number) { return api.get(`/v1/hub/community/workers/${id}`) }
export function publishCommunityWorker(data: any) { return api.post('/v1/hub/community/workers', data) }
export function installCommunityWorker(id: number, businessCode: string) { return api.post(`/v1/hub/community/workers/${id}/install`, { business_code: businessCode }) }
export function getCommunityWorkerReviews(id: number) { return api.get(`/v1/hub/community/workers/${id}/reviews`) }

// Link requests
export function getLinkRequests(code: string) { return api.get(`/v1/hub/businesses/${code}/link-requests`) }
export function reviewLinkRequest(code: string, id: number, action: string) { return api.post(`/v1/hub/businesses/${code}/link-requests/${id}/review`, { action }) }
export function createLinkRequest(code: string, deviceInfo?: string) {
  return api.post(`/v1/hub/businesses/${code}/link-requests`, { device_info: deviceInfo || '' })
}

// Invites & members
export function inviteMember(code: string, email: string, role = 'member') {
  return api.post(`/v1/hub/businesses/${code}/invite`, { email, role })
}
export function listInvites(code: string) {
  return api.get(`/v1/hub/businesses/${code}/invites`)
}
export function revokeInvite(code: string, id: number) {
  return api.post(`/v1/hub/businesses/${code}/invites/${id}/revoke`)
}
export function acceptInvite(token: string) {
  return api.post('/v1/hub/invites/accept', { token })
}
export function listMembers(code: string) {
  return api.get(`/v1/hub/businesses/${code}/members`)
}

// Branch index
export function listBranches(code: string) {
  return api.get(`/v1/hub/repos/${code}/branches`)
}
export function refreshBranches(code: string, data?: { repo_url?: string; default_only?: boolean }) {
  return api.post(`/v1/hub/repos/${code}/branches/refresh`, data || {})
}

// Requirements
export function listRequirements(code: string, status?: string) {
  return api.get(`/v1/hub/requirements/${code}`, { params: status ? { status } : {} })
}
export function getRequirement(code: string, id: number) {
  return api.get(`/v1/hub/requirements/${code}/${id}`)
}
export function createRequirement(code: string, data: { title: string; description?: string }) {
  return api.post(`/v1/hub/requirements/${code}`, data)
}
export function updateRequirement(code: string, id: number, data: { title?: string; description?: string }) {
  return api.patch(`/v1/hub/requirements/${code}/${id}`, data)
}
export function submitRequirement(code: string, id: number, comment?: string) {
  return api.post(`/v1/hub/requirements/${code}/${id}/submit`, { comment })
}
export function acceptRequirement(code: string, id: number, comment?: string) {
  return api.post(`/v1/hub/requirements/${code}/${id}/accept`, { comment })
}
export function rejectRequirement(code: string, id: number, comment?: string) {
  return api.post(`/v1/hub/requirements/${code}/${id}/reject`, { comment })
}
export function cancelRequirement(code: string, id: number, comment?: string) {
  return api.post(`/v1/hub/requirements/${code}/${id}/cancel`, { comment })
}
export function listRequirementComments(code: string, id: number) {
  return api.get(`/v1/hub/requirements/${code}/${id}/comments`)
}
export function createRequirementComment(code: string, id: number, data: { body: string; decision?: string }) {
  return api.post(`/v1/hub/requirements/${code}/${id}/comments`, data)
}

export default api
