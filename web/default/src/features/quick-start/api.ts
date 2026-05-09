import { api } from '@/lib/api'
import type { QuickStartResponse } from './types'

export async function getQuickStartContent() {
  try {
    const res = await api.get<QuickStartResponse>('/api/quickstart')
    return res.data
  } catch {
    // Backward compatibility for deployments using underscored endpoint.
    const res = await api.get<QuickStartResponse>('/api/quick_start')
    return res.data
  }
}
