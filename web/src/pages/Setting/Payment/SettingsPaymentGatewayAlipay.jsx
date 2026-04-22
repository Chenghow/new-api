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

export default function SettingsPaymentGatewayAlipay(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    AlipayEnabled: false,
    AlipaySandbox: false,
    AlipayAppId: '',
    AlipayPrivateKey: '',
    AlipayPublicKey: '',
  });
  const [originInputs, setOriginInputs] = useState({});
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const currentInputs = {
        AlipayEnabled:
          props.options.AlipayEnabled === 'true' || props.options.AlipayEnabled === true,
        AlipaySandbox:
          props.options.AlipaySandbox === 'true' || props.options.AlipaySandbox === true,
        AlipayAppId: props.options.AlipayAppId || '',
        AlipayPrivateKey: '',
        AlipayPublicKey: '',
      };
      setInputs(currentInputs);
      setOriginInputs({ ...currentInputs });
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const handleFormChange = (values) => {
    setInputs(values);
  };

  const submitAlipaySetting = async () => {
    if (!props.options.ServerAddress) {
      showError(t('请先填写服务器地址'));
      return;
    }

    setLoading(true);
    try {
      const options = [];

      if (originInputs.AlipayEnabled !== inputs.AlipayEnabled) {
        options.push({
          key: 'AlipayEnabled',
          value: inputs.AlipayEnabled ? 'true' : 'false',
        });
      }
      if (originInputs.AlipaySandbox !== inputs.AlipaySandbox) {
        options.push({
          key: 'AlipaySandbox',
          value: inputs.AlipaySandbox ? 'true' : 'false',
        });
      }
      if (inputs.AlipayAppId && inputs.AlipayAppId !== '') {
        options.push({ key: 'AlipayAppId', value: inputs.AlipayAppId.trim() });
      }
      if (inputs.AlipayPrivateKey && inputs.AlipayPrivateKey.trim() !== '') {
        options.push({ key: 'AlipayPrivateKey', value: inputs.AlipayPrivateKey.trim() });
      }
      if (inputs.AlipayPublicKey && inputs.AlipayPublicKey.trim() !== '') {
        options.push({ key: 'AlipayPublicKey', value: inputs.AlipayPublicKey.trim() });
      }

      if (options.length === 0) {
        showError(t('没有需要更新的配置'));
        setLoading(false);
        return;
      }

      const requestQueue = options.map((opt) =>
        API.put('/api/option/', {
          key: opt.key,
          value: opt.value,
        }),
      );

      const results = await Promise.all(requestQueue);
      const errorResults = results.filter((res) => !res.data.success);
      if (errorResults.length > 0) {
        errorResults.forEach((res) => {
          showError(res.data.message);
        });
      } else {
        showSuccess(t('更新成功'));
        setOriginInputs({
          ...originInputs,
          AlipayEnabled: inputs.AlipayEnabled,
          AlipaySandbox: inputs.AlipaySandbox,
          AlipayAppId: inputs.AlipayAppId,
        });
        formApiRef.current?.setValue('AlipayPrivateKey', '');
        formApiRef.current?.setValue('AlipayPublicKey', '');
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
        <Form.Section text={t('支付宝 v3 设置')}>
          <Text>
            {t('采用支付宝开放平台网页支付（alipay.trade.page.pay）。建议先开启沙箱验证，再切到正式环境。')}
          </Text>
          <Banner
            type='info'
            description={`Notify URL：${props.options.ServerAddress ? removeTrailingSlash(props.options.ServerAddress) : t('网站地址')}/api/alipay/notify`}
          />
          <Banner
            type='warning'
            description={t('私钥和支付宝公钥支持粘贴 PEM（可含换行），为空则不会覆盖已有值。')}
          />

          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='AlipayEnabled'
                size='default'
                checkedText='｜'
                uncheckedText='〇'
                label={t('启用支付宝 v3')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='AlipaySandbox'
                size='default'
                checkedText='｜'
                uncheckedText='〇'
                label={t('使用沙箱环境')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='AlipayAppId'
                label={t('AppId')}
                placeholder={t('支付宝开放平台应用 AppId')}
              />
            </Col>
          </Row>

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='AlipayPrivateKey'
                label={t('应用私钥（RSA2）')}
                placeholder={t('粘贴应用私钥 PEM 内容，留空表示不修改')}
                autosize={{ minRows: 4, maxRows: 8 }}
              />
            </Col>
            <Col xs={24} sm={24} md={12} lg={12} xl={12}>
              <Form.TextArea
                field='AlipayPublicKey'
                label={t('支付宝公钥')}
                placeholder={t('粘贴支付宝公钥 PEM 内容，留空表示不修改')}
                autosize={{ minRows: 4, maxRows: 8 }}
              />
            </Col>
          </Row>

          <Button onClick={submitAlipaySetting}>{t('更新支付宝设置')}</Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
