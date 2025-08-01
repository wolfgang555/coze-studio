/*
 * Copyright 2025 coze-dev Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
 
import { type FC, type ReactNode } from 'react';

import { I18n } from '@coze-arch/i18n';
import {
  IllustrationConstruction,
  IllustrationFailure,
} from '@douyinfe/semi-illustrations';

export const ExceptionDisplay: FC<{ title: string; image: ReactNode }> = ({
  title,
  image,
}) => (
  <div className="flex flex-col gap-[16px] justify-center items-center h-[80%]">
    {image}
    <span className="coz-fg-plus text-[16px] font-medium leading-[22px]">
      {title}
    </span>
  </div>
);

/**
 * 加载失败
 */
export const LoadFailedDisplay = () => (
  <ExceptionDisplay
    image={<IllustrationFailure className="h-[140px] w-[140px]" />}
    title={I18n.t('plugin_exception')}
  />
);

/**
 * 无数据
 */
export const NoDataDisplay = () => (
  <ExceptionDisplay
    image={<IllustrationConstruction className="h-[140px] w-[140px]" />}
    title={I18n.t('debug_asyn_task_notask')}
  />
);
