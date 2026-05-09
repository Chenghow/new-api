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

import React, { useEffect } from 'react';
import UsageLogsTable from '../../components/table/usage-logs';
import { API, showError, showSuccess } from '../../helpers';

const Token = () => {
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const outTradeNo = params.get('out_trade_no');
    if (!outTradeNo) return;

    // 从支付宝返回后主动查询订单状态
    API.get(`/api/alipay/check?out_trade_no=${encodeURIComponent(outTradeNo)}`)
      .then((res) => {
        if (res.data && res.data.data === '充值成功') {
          showSuccess('充值成功！额度已到账，请刷新页面查看。');
        } else {
          showError('订单支付处理中，请稍后刷新页面查看额度。');
        }
      })
      .catch(() => {
        showError('订单支付处理中，请稍后刷新页面查看额度。');
      });
  }, []);

  return (
    <div className='mt-[60px] px-2'>
      <UsageLogsTable />
    </div>
  );
};

export default Token;
