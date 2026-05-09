import { createFileRoute } from '@tanstack/react-router'
import { QuickStart } from '@/features/quick-start'

export const Route = createFileRoute('/quick-start/')({
  component: QuickStart,
})
