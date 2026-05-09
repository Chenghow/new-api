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

export interface WechatPaySettingsValues {
  WechatPayEnabled: boolean
  WechatPayAppId: string
  WechatPayMchId: string
  WechatPayApiV3Key: string
  WechatPayCertSerialNo: string
  WechatPayPublicKeyId: string
  WechatPayPrivateKey: string
  WechatPayPublicKey: string
}

interface Props {
  defaultValues: WechatPaySettingsValues
}

export function WechatPaySection({ defaultValues }: Props) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const form = useForm<WechatPaySettingsValues>({
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
        { key: 'WechatPayEnabled', value: String(values.WechatPayEnabled) },
        { key: 'WechatPayAppId', value: values.WechatPayAppId.trim() },
        { key: 'WechatPayMchId', value: values.WechatPayMchId.trim() },
        { key: 'WechatPayCertSerialNo', value: values.WechatPayCertSerialNo.trim() },
        { key: 'WechatPayPublicKeyId', value: values.WechatPayPublicKeyId.trim() },
      ]
      // Sensitive keys: only send if non-empty (blank = keep existing)
      if (values.WechatPayApiV3Key.trim())
        options.push({ key: 'WechatPayApiV3Key', value: values.WechatPayApiV3Key.trim() })
      if (values.WechatPayPrivateKey.trim())
        options.push({ key: 'WechatPayPrivateKey', value: values.WechatPayPrivateKey.trim() })
      if (values.WechatPayPublicKey.trim())
        options.push({ key: 'WechatPayPublicKey', value: values.WechatPayPublicKey.trim() })

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
      title={t('WeChat Native Pay')}
      description={t('Configure WeChat Pay APIv3 native payment integration')}
    >
      <div className="space-y-6">
        {/* Enable toggle */}
        <div className="flex items-center gap-3">
          <Switch
            checked={form.watch('WechatPayEnabled')}
            onCheckedChange={(v) => form.setValue('WechatPayEnabled', v)}
          />
          <Label>{t('Enable WeChat Pay')}</Label>
        </div>

        {/* AppId / MchId */}
        <div className="grid gap-6 md:grid-cols-2">
          <div className="space-y-2">
            <Label>{t('App ID (appid)')}</Label>
            <Input
              placeholder="wx1234567890abcdef"
              autoComplete="off"
              {...form.register('WechatPayAppId')}
            />
            <p className="text-muted-foreground text-sm">
              {t('WeChat Official Account or Mini Program App ID')}
            </p>
          </div>
          <div className="space-y-2">
            <Label>{t('Merchant ID (mchid)')}</Label>
            <Input
              placeholder="1234567890"
              autoComplete="off"
              {...form.register('WechatPayMchId')}
            />
          </div>
        </div>

        {/* ApiV3Key / CertSerialNo */}
        <div className="grid gap-6 md:grid-cols-2">
          <div className="space-y-2">
            <Label>{t('APIv3 key')}</Label>
            <Input
              type="password"
              placeholder={t('Leave blank to keep existing value')}
              autoComplete="new-password"
              {...form.register('WechatPayApiV3Key')}
            />
            <p className="text-muted-foreground text-sm">
              {t('32-character key set in WeChat Pay console')}
            </p>
          </div>
          <div className="space-y-2">
            <Label>{t('Certificate serial number')}</Label>
            <Input
              placeholder="3C8FB4D3C8FB4D..."
              autoComplete="off"
              {...form.register('WechatPayCertSerialNo')}
            />
          </div>
        </div>

        {/* PublicKeyId */}
        <div className="grid gap-6 md:grid-cols-2">
          <div className="space-y-2">
            <Label>{t('Public key ID')}</Label>
            <Input
              placeholder="PUB_KEY_ID_..."
              autoComplete="off"
              {...form.register('WechatPayPublicKeyId')}
            />
            <p className="text-muted-foreground text-sm">
              {t('Required when using public key mode instead of certificate')}
            </p>
          </div>
        </div>

        {/* Private / Public keys */}
        <div className="space-y-4">
          <div className="space-y-2">
            <Label>{t('Merchant private key (PEM)')}</Label>
            <Textarea
              rows={5}
              placeholder={t('Leave blank to keep existing value')}
              autoComplete="off"
              {...form.register('WechatPayPrivateKey')}
            />
            <p className="text-muted-foreground text-sm">
              {t('Leave blank unless rotating the key')}
            </p>
          </div>
          <div className="space-y-2">
            <Label>{t('Platform public key (PEM)')}</Label>
            <Textarea
              rows={5}
              placeholder={t('Leave blank to keep existing value')}
              autoComplete="off"
              {...form.register('WechatPayPublicKey')}
            />
            <p className="text-muted-foreground text-sm">
              {t('Leave blank unless rotating the key')}
            </p>
          </div>
        </div>

        {/* Notify URL */}
        <Alert>
          <AlertDescription className="text-sm font-mono">
            {'<ServerAddress>/api/wechat/notify'}
          </AlertDescription>
        </Alert>
        <p className="text-muted-foreground text-sm">
          {t('Set this URL as the WeChat Pay callback notification address in WeChat Pay console')}
        </p>

        <Button onClick={handleSave} disabled={updateOption.isPending}>
          {updateOption.isPending ? t('Saving...') : t('Save WeChat Pay settings')}
        </Button>
      </div>
    </SettingsSection>
  )
}
