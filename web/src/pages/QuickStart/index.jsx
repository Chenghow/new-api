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

import React, { useEffect, useState } from 'react';
import { API, showError } from '../../helpers';
import { marked } from 'marked';
import { Empty } from '@douyinfe/semi-ui';
import {
  IllustrationConstruction,
  IllustrationConstructionDark,
} from '@douyinfe/semi-illustrations';
import { useTranslation } from 'react-i18next';

const QuickStart = () => {
  const { t } = useTranslation();
  const [quickStart, setQuickStart] = useState('');
  const [quickStartLoaded, setQuickStartLoaded] = useState(false);

  const displayQuickStart = async () => {
    setQuickStart(localStorage.getItem('quickStart') || '');
    const res = await API.get('/api/quickstart');
    const { success, message, data } = res.data;
    if (success) {
      let quickStartContent = data;
      if (!data.startsWith('https://')) {
        quickStartContent = marked.parse(data);
      }
      setQuickStart(quickStartContent);
      localStorage.setItem('quickStart', quickStartContent);
    } else {
      showError(message);
      setQuickStart(t('加载快速开始内容失败...'));
    }
    setQuickStartLoaded(true);
  };

  useEffect(() => {
    displayQuickStart().then();
  }, []);

  const emptyStyle = {
    padding: '24px',
  };

  const customDescription = (
    <div style={{ textAlign: 'center' }}>
      <p>{t('可在设置页面设置快速开始内容，支持 HTML & Markdown')}</p>
    </div>
  );

  return (
    <div className='mt-[60px] px-2'>
      {quickStartLoaded && quickStart === '' ? (
        <div className='flex justify-center items-center h-screen p-8'>
          <Empty
            image={
              <IllustrationConstruction style={{ width: 150, height: 150 }} />
            }
            darkModeImage={
              <IllustrationConstructionDark
                style={{ width: 150, height: 150 }}
              />
            }
            description={t('管理员暂时未设置任何快速开始内容')}
            style={emptyStyle}
          >
            {customDescription}
          </Empty>
        </div>
      ) : (
        <>
          {quickStart.startsWith('https://') ? (
            <iframe
              src={quickStart}
              style={{ width: '100%', height: '100vh', border: 'none' }}
            />
          ) : (
            <div
              style={{ fontSize: 'larger' }}
              dangerouslySetInnerHTML={{ __html: quickStart }}
            ></div>
          )}
        </>
      )}
    </div>
  );
};

export default QuickStart;
