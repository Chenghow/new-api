/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useEffect, useRef, useState } from 'react';
import { Banner, Button, Col, Form, Row, Spin, Typography } from '@douyinfe/semi-ui';
import { API, removeTrailingSlash, showError, showSuccess } from '../../../helpers';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

export default function SettingsPaymentGatewayWechat(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    WechatPayEnabled: false,
    WechatPayAppId: '',
    WechatPayMchId: '',
    WechatPayApiV3Key: '',
    WechatPayCertSerialNo: '',
    WechatPayPublicKeyId: '',
    WechatPayPrivateKey: '',
    WechatPayPublicKey: '',
  });
  const [originInputs, setOriginInputs] = useState({});
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        WechatPayEnabled:
          props.options.WechatPayEnabled === 'true' || props.options.WechatPayEnabled === true,
        WechatPayAppId: props.options.WechatPayAppId || '',
        WechatPayMchId: props.options.WechatPayMchId || '',
        WechatPayApiV3Key: '',
        WechatPayCertSerialNo: props.options.WechatPayCertSerialNo || '',
        WechatPayPublicKeyId: props.options.WechatPayPublicKeyId || '',
        WechatPayPrivateKey: '',
        WechatPayPublicKey: '',
      };
      setInputs(currentInputs);
      setOriginInputs({ ...currentInputs });
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const handleFormChange = (values) => {
    setInputs(values);
  };

  const submitWechatSetting = async () => {
    if (!props.options.ServerAddress) {
      showError(t('请先填写服务器地址'));
      return;
    }

    setLoading(true);
    try {
      const options = [];

      if (originInputs.WechatPayEnabled !== inputs.WechatPayEnabled) {
        options.push({
          key: 'WechatPayEnabled',
          value: inputs.WechatPayEnabled ? 'true' : 'false',
        });
      }
      if (inputs.WechatPayAppId && inputs.WechatPayAppId.trim() !== '') {
        options.push({ key: 'WechatPayAppId', value: inputs.WechatPayAppId.trim() });
      }
      if (inputs.WechatPayMchId && inputs.WechatPayMchId.trim() !== '') {
        options.push({ key: 'WechatPayMchId', value: inputs.WechatPayMchId.trim() });
      }
      if (inputs.WechatPayApiV3Key && inputs.WechatPayApiV3Key.trim() !== '') {
        options.push({ key: 'WechatPayApiV3Key', value: inputs.WechatPayApiV3Key.trim() });
      }
      if (inputs.WechatPayCertSerialNo && inputs.WechatPayCertSerialNo.trim() !== '') {
        options.push({ key: 'WechatPayCertSerialNo', value: inputs.WechatPayCertSerialNo.trim() });
      }
      if (inputs.WechatPayPublicKeyId && inputs.WechatPayPublicKeyId.trim() !== '') {
        options.push({ key: 'WechatPayPublicKeyId', value: inputs.WechatPayPublicKeyId.trim() });
      }
      if (inputs.WechatPayPrivateKey && inputs.WechatPayPrivateKey.trim() !== '') {
        options.push({ key: 'WechatPayPrivateKey', value: inputs.WechatPayPrivateKey.trim() });
      }
      if (inputs.WechatPayPublicKey && inputs.WechatPayPublicKey.trim() !== '') {
        options.push({ key: 'WechatPayPublicKey', value: inputs.WechatPayPublicKey.trim() });
      }

      if (options.length === 0) {
        showError(t('没有需要更新的配置'));
        setLoading(false);
        return;
      }

      const requestQueue = options.map((opt) =>
        API.put('/api/option/', { key: opt.key, value: opt.value }),
      );

      const results = await Promise.all(requestQueue);
      const errorResults = results.filter((res) => !res.data.success);
      if (errorResults.length > 0) {
        errorResults.forEach((res) => showError(res.data.message));
      } else {
        showSuccess(t('更新成功'));
        setOriginInputs({
          ...originInputs,
          WechatPayEnabled: inputs.WechatPayEnabled,
          WechatPayAppId: inputs.WechatPayAppId,
          WechatPayMchId: inputs.WechatPayMchId,
          WechatPayCertSerialNo: inputs.WechatPayCertSerialNo,
          WechatPayPublicKeyId: inputs.WechatPayPublicKeyId,
        });
        formApiRef.current?.setValue('WechatPayApiV3Key', '');
        formApiRef.current?.setValue('WechatPayPrivateKey', '');
        formApiRef.current?.setValue('WechatPayPublicKey', '');
        props.refresh?.();
      }
    } catch (error) {
      showError(t('更新失败'));
    }
    setLoading(false);
  };

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        onValueChange={handleFormChange}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Form.Section text={t('微信 Native 支付设置')}>
          <Text>
            {t('采用微信支付 APIv3 Native 扫码支付。用户支付时会展示二维码，适合 PC 端充值场景。')}
          </Text>
          <Banner
            type='info'
            description={`Notify URL：${props.options.ServerAddress ? removeTrailingSlash(props.options.ServerAddress) : t('网站地址')}/api/wechat/notify`}
          />
          <Banner
            type='warning'
            description={t('APIv3Key、商户私钥、微信支付公钥为敏感信息，留空表示不覆盖已有值。')}
          />

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='WechatPayEnabled'
                size='default'
                checkedText='｜'
                uncheckedText='〇'
                label={t('启用微信 Native 支付')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='WechatPayAppId'
                label={t('AppId')}
                placeholder={t('微信 AppId（如 wx...）')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='WechatPayMchId'
                label={t('商户号（MchId）')}
                placeholder={t('微信支付商户号')}
              />
            </Col>
          </Row>

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='WechatPayApiV3Key'
                label={t('APIv3Key（32 字节）')}
                placeholder={t('留空表示不修改')}
                mode='password'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='WechatPayCertSerialNo'
                label={t('商户证书序列号')}
                placeholder={t('商户 API 证书序列号')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='WechatPayPublicKeyId'
                label={t('微信支付公钥 ID')}
                placeholder={t('PUB_KEY_ID_xxxxx')}
              />
            </Col>
          </Row>

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='WechatPayPrivateKey'
                label={t('商户 API 私钥（PEM）')}
                placeholder={t('粘贴私钥 PEM 内容，留空表示不修改')}
                autosize={{ minRows: 4, maxRows: 8 }}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='WechatPayPublicKey'
                label={t('微信支付公钥（PEM）')}
                placeholder={t('粘贴微信支付公钥 PEM 内容，留空表示不修改')}
                autosize={{ minRows: 4, maxRows: 8 }}
              />
            </Col>
          </Row>

          <Button onClick={submitWechatSetting} style={{ marginTop: 8 }}>
            {t('更新微信支付设置')}
          </Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
