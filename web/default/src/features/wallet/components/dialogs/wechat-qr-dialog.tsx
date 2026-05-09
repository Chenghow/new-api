import { useEffect, useRef, useCallback } from 'react'
import { QRCodeSVG } from 'qrcode.react'
import { CheckCircle2, Loader2, RefreshCcw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

interface WechatQrDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  qrUrl: string
  orderId: string
  onPaymentSuccess: () => void
}

const POLL_INTERVAL_MS = 3000

export function WechatQrDialog({
  open,
  onOpenChange,
  qrUrl,
  orderId,
  onPaymentSuccess,
}: WechatQrDialogProps) {
  const { t } = useTranslation()
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const stopPolling = useCallback(() => {
    if (pollRef.current) {
      clearInterval(pollRef.current)
      pollRef.current = null
    }
  }, [])

  const startPolling = useCallback(() => {
    stopPolling()
    if (!orderId) return

    pollRef.current = setInterval(async () => {
      try {
        const res = await api.get(
          `/api/wechat/check?out_trade_no=${encodeURIComponent(orderId)}`
        )
        const { message } = res.data as { message?: string }
        if (message === 'success') {
          stopPolling()
          onOpenChange(false)
          toast.success(t('WeChat Pay successful'))
          onPaymentSuccess()
        }
      } catch {
        // ignore transient polling errors
      }
    }, POLL_INTERVAL_MS)
  }, [orderId, stopPolling, onOpenChange, onPaymentSuccess, t])

  useEffect(() => {
    if (open && orderId) {
      startPolling()
    } else {
      stopPolling()
    }
    return () => stopPolling()
  }, [open, orderId, startPolling, stopPolling])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-sm:w-[calc(100vw-1.5rem)] sm:max-w-sm'>
        <DialogHeader>
          <DialogTitle>{t('WeChat Pay')}</DialogTitle>
          <DialogDescription>
            {t('Scan the QR code with WeChat to complete payment')}
          </DialogDescription>
        </DialogHeader>

        <div className='flex flex-col items-center gap-4 py-4'>
          {qrUrl ? (
            <>
              <div className='rounded-xl border p-4 shadow-sm'>
                <QRCodeSVG value={qrUrl} size={200} />
              </div>

              <div className='text-muted-foreground flex items-center gap-2 text-sm'>
                <Loader2 className='h-4 w-4 animate-spin' />
                {t('Waiting for payment...')}
              </div>

              <div className='text-muted-foreground text-center text-xs'>
                {t('Payment will be confirmed automatically after scanning')}
              </div>
            </>
          ) : (
            <div className='text-muted-foreground py-8 text-sm'>
              {t('QR code unavailable')}
            </div>
          )}
        </div>

        <div className='flex gap-2'>
          <Button
            variant='outline'
            className='flex-1'
            onClick={() => startPolling()}
          >
            <RefreshCcw className='mr-2 h-4 w-4' />
            {t('Refresh')}
          </Button>
          <Button
            variant='outline'
            className='flex-1'
            onClick={() => {
              stopPolling()
              onOpenChange(false)
            }}
          >
            {t('Cancel')}
          </Button>
          <Button
            className='flex-1'
            onClick={() => {
              stopPolling()
              onOpenChange(false)
              onPaymentSuccess()
              toast.success(t('Payment confirmed'))
            }}
          >
            <CheckCircle2 className='mr-2 h-4 w-4' />
            {t("I've paid")}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
