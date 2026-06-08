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
}
export const me = {
  getBusinesses: () => api.get('/v1/hub/me/businesses'),
  joinBusiness: (code: string) => api.post(`/v1/hub/businesses/${code}/join`),
}
export function getBusinesses(params?: any) { return api.get('/v1/hub/businesses', { params }) }
export function createBusiness(data: any) { return api.post('/v1/hub/businesses', data) }
export function getWorkers(params?: any) { return api.get('/v1/hub/workers', { params }) }
export function getLocks(params?: any) { return api.get('/v1/hub/locks', { params }) }
export function getEvents(params?: any) { return api.get('/v1/hub/events', { params }) }
export function searchPlaybooks(params?: any) { return api.get('/v1/hub/playbooks/search', { params }) }
export function getDAG(code: string) { return api.get(`/v1/hub/dag/${code}`) }

// Community marketplace
export function listCommunityWorkers(params?: any) { return api.get('/v1/hub/community/workers', { params }) }
export function getCommunityWorker(id: number) { return api.get(`/v1/hub/community/workers/${id}`) }
export function publishCommunityWorker(data: any) { return api.post('/v1/hub/community/workers', data) }
export function installCommunityWorker(id: number, businessCode: string) { return api.post(`/v1/hub/community/workers/${id}/install`, { business_code: businessCode }) }
export function getCommunityWorkerReviews(id: number) { return api.get(`/v1/hub/community/workers/${id}/reviews`) }

export default api
