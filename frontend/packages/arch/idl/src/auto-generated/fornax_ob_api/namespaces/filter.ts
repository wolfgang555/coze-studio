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
 
// THIS IS AN AUTOGENERATED FILE. DO NOT EDIT THIS FILE DIRECTLY.
/* eslint-disable */
/* tslint:disable */
// @ts-nocheck

import * as flow_devops_fornaxob_common from './flow_devops_fornaxob_common';

export type Int64 = string | number;

export interface FilterField {
  field_name?: string;
  field_type?: string;
  values?: Array<string>;
  query_type?: string;
  query_and_or?: string;
  sub_filter?: FilterFields;
  /** 临时新增，预期不应该存在 */
  hidden?: boolean;
}

export interface FilterFields {
  query_and_or?: string;
  filter_fields: Array<FilterField>;
}

export interface SpanFilters {
  /** Span 过滤条件 */
  filters?: FilterFields;
  /** 平台类型，不填默认是fornax */
  platform_type?: flow_devops_fornaxob_common.PlatformType;
  /** 查询的 span 标签页类型，不填默认是 root span */
  span_list_type?: flow_devops_fornaxob_common.SpanListType;
}

export interface TaskFilterField {
  field_name?: string;
  field_type?: string;
  values?: Array<string>;
  query_type?: string;
  query_and_or?: string;
  sub_filter?: TaskFilterField;
}

export interface TaskFilterFields {
  query_and_or?: string;
  filter_fields: Array<TaskFilterField>;
}
/* eslint-enable */
