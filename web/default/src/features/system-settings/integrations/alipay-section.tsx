import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

export interface AlipaySettingsValues {
  AlipayEnabled: boolean
  AlipayAppId: string
  AlipayPrivateKey: string
  AlipayPublicKey: string
  AlipaySandbox: boolean
}

interface Props {
  defaultValues: AlipaySettingsValues
}

export function AlipaySection({ defaultValues }: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const form = useForm<AlipaySettingsValues>({
    defaultValues,
  })

  useEffect(() => {
    form.reset(defaultValues)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [JSON.stringify(defaultValues)])

  const handleSave = async () => {
    try {
      const values = form.getValues()
      const options: { key: string; value: string }[] = [
        { key: 'AlipayEnabled', value: String(values.AlipayEnabled) },
        { key: 'AlipaySandbox', value: String(values.AlipaySandbox) },
        { key: 'AlipayAppId', value: values.AlipayAppId.trim() },
      ]
      // Sensitive keys: only send if non-empty (blank = keep existing)
      if (values.AlipayPrivateKey.trim())
        options.push({ key: 'AlipayPrivateKey', value: values.AlipayPrivateKey.trim() })
      if (values.AlipayPublicKey.trim())
        options.push({ key: 'AlipayPublicKey', value: values.AlipayPublicKey.trim() })

      for (const opt of options) {
        await updateOption.mutateAsync(opt)
      }
      toast.success(t('Updated successfully'))
    } catch {
      toast.error(t('Update failed'))
    }
  }

  return (
    <SettingsSection
      title={t('Alipay v3 (Direct Connect)')}
      description={t('Configure Alipay direct-connect payment integration')}
    >
      <div className="space-y-6">
        {/* Enable / Sandbox toggles */}
        <div className="flex flex-wrap gap-6">
          <div className="flex items-center gap-3">
            <Switch
              checked={form.watch('AlipayEnabled')}
              onCheckedChange={(v) => form.setValue('AlipayEnabled', v)}
            />
            <Label>{t('Enable Alipay v3')}</Label>
          </div>
          <div className="flex items-center gap-3">
            <Switch
              checked={form.watch('AlipaySandbox')}
              onCheckedChange={(v) => form.setValue('AlipaySandbox', v)}
            />
            <Label>{t('Sandbox mode')}</Label>
          </div>
        </div>

        {/* App ID */}
        <div className="grid gap-6 md:grid-cols-2">
          <div className="space-y-2">
            <Label>{t('Alipay App ID')}</Label>
            <Input
              placeholder="2021001234567890"
              autoComplete="off"
              {...form.register('AlipayAppId')}
            />
          </div>
        </div>

        {/* Private / Public keys */}
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>{t('App private key (RSA2)')}</Label>
            <Textarea
              rows={5}
              placeholder={t('Leave blank to keep existing value')}
              autoComplete="off"
              {...form.register('AlipayPrivateKey')}
            />
            <p className="text-muted-foreground text-sm">
              {t('Leave blank unless rotating the key')}
            </p>
          </div>
          <div className="space-y-2">
            <Label>{t('Alipay public key')}</Label>
            <Textarea
              rows={5}
              placeholder={t('Leave blank to keep existing value')}
              autoComplete="off"
              {...form.register('AlipayPublicKey')}
            />
            <p className="text-muted-foreground text-sm">
              {t('Leave blank unless rotating the key')}
            </p>
          </div>
        </div>

        {/* Notify URL */}
        <Alert>
          <AlertDescription className="text-sm font-mono">
            {'<ServerAddress>/api/alipay/notify'}
          </AlertDescription>
        </Alert>
        <p className="text-muted-foreground text-sm">
          {t('Set this URL as the Alipay asynchronous notification address in your Alipay Open Platform console')}
        </p>

        <Button onClick={handleSave} disabled={updateOption.isPending}>
          {updateOption.isPending ? t('Saving...') : t('Save Alipay settings')}
        </Button>
      </div>
    </SettingsSection>
  )
}
