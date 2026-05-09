import { z } from 'zod'
import { createFileRoute } from '@tanstack/react-router'
import { Wallet } from '@/features/wallet'

const walletSearchSchema = z.object({
  show_history: z.boolean().optional(),
  // Alipay synchronous return: triggers active order check on mount
  alipay_return: z.coerce.number().optional(),
  out_trade_no: z.string().optional(),
})

export const Route = createFileRoute('/_authenticated/wallet/')({
  component: RouteComponent,
  validateSearch: walletSearchSchema,
})

function RouteComponent() {
  const { show_history, alipay_return, out_trade_no } = Route.useSearch()
  return (
    <Wallet
      initialShowHistory={show_history}
      alipayReturn={!!alipay_return}
      alipayOutTradeNo={out_trade_no}
    />
  )
}
