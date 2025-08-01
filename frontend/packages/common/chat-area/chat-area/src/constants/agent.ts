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
 
import { AgentType } from '@coze-arch/bot-api/developer_api';

// TODO: 这里为啥没i18n需要做
// 节点类型名称映射关系
export const agentTypeNameMap: { [key in AgentType]: string | undefined } = {
  [AgentType.LLM_Agent]: 'Agent',
  [AgentType.Bot_Agent]: 'Bot',
  [AgentType.Global_Agent]: 'Bot',
  [AgentType.Start_Agent]: 'Bot',
  [AgentType.Task_Agent]: 'Bot',
};
