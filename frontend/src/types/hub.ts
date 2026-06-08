export interface Business {
  id: number; code: string; name: string; repo_url: string
  owner_user_id: number; description: string; status: string
  created_at: string; updated_at: string
}
export interface Worker {
  id: number; worker_id: string; version: string
  last_heartbeat_at: string; status: string; host: string; pid: number
}
export interface Lock {
  id: number; resource_key: string; holder_token: string
  holder_worker_id: string; acquired_at: string; expires_at: string; released_at: string
}
export interface Playbook {
  id: number; category: string; title: string; content: string
  tags: string[]; created_by_worker_id: string
}
export interface HubEvent {
  id: number; actor: string; event_type: string
  payload: Record<string, any>; created_at: string
}
