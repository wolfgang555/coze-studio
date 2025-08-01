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

import * as datasetv2 from './datasetv2';
import * as common from './common';

export type Int64 = string | number;

export interface EvaluationSet {
  /** 主键&外键 */
  id?: Int64;
  app_id?: number;
  space_id?: Int64;
  /** 基础信息 */
  name?: string;
  description?: string;
  status?: datasetv2.DatasetStatus;
  /** 规格限制 */
  spec?: datasetv2.DatasetSpec;
  /** 功能开关 */
  features?: datasetv2.DatasetFeatures;
  /** 数据条数 */
  item_count?: Int64;
  /** 是否有未提交的修改 */
  change_uncommitted?: boolean;
  /** 版本信息
版本详情信息 */
  evaluation_set_version?: EvaluationSetVersion;
  /** 最新的版本号 */
  latest_version?: string;
  /** 下一个的版本号 */
  next_version_num?: Int64;
  /** 系统信息 */
  base_info?: common.BaseInfo;
}

export interface EvaluationSetItem {
  /** 主键&外键
主键，随版本变化 */
  id?: Int64;
  app_id?: number;
  space_id?: Int64;
  evaluation_set_id?: Int64;
  schema_id?: Int64;
  /** 数据在当前数据集内的唯一 ID，不随版本发生改变 */
  item_id?: Int64;
  /** 数据插入的幂等 key */
  item_key?: string;
  /** 轮次数据内容 */
  turns?: Array<Turn>;
  /** 系统信息 */
  base_info?: common.BaseInfo;
}

/** EvaluationSetSchema 评测集 Schema，包含字段的类型限制等信息 */
export interface EvaluationSetSchema {
  /** 主键&外键 */
  id?: Int64;
  app_id?: number;
  space_id?: Int64;
  evaluation_set_id?: Int64;
  /** 数据集字段约束 */
  field_schemas?: Array<FieldSchema>;
  /** 系统信息 */
  base_info?: common.BaseInfo;
}

export interface EvaluationSetVersion {
  /** 主键&外键 */
  id?: Int64;
  app_id?: number;
  space_id?: Int64;
  evaluation_set_id?: Int64;
  /** 版本信息
展示的版本号，SemVer2 三段式 */
  version?: string;
  /** 后端记录的数字版本号，从 1 开始递增 */
  version_num?: Int64;
  /** 版本描述 */
  description?: string;
  /** schema */
  evaluation_set_schema?: EvaluationSetSchema;
  /** 数据条数 */
  item_count?: Int64;
  /** 系统信息 */
  base_info?: common.BaseInfo;
}

export interface FieldData {
  key?: string;
  name?: string;
  content?: common.Content;
}

export interface FieldSchema {
  /** 唯一键 */
  key?: string;
  /** 展示名称 */
  name?: string;
  /** 描述 */
  description?: string;
  /** 类型，如 文本，图片，etc. */
  content_type?: string;
  /** 默认渲染格式，如 code, json, etc.mai */
  default_display_format?: datasetv2.FieldDisplayFormat;
  /** 当前列的状态 */
  status?: datasetv2.FieldStatus;
  /** [20,50) 内容格式限制相关
文本内容格式限制，格式为 JSON schema，协议参考 https://json-schema.org/specification */
  text_schema?: string;
  /** 多模态规格限制 */
  multi_model_spec?: datasetv2.MultiModalSpec;
  /** 用户是否不可见 */
  hidden?: boolean;
}

export interface Turn {
  /** 轮次ID，如果是单轮评测集，id=0 */
  id?: Int64;
  /** 字段数据 */
  field_data_list?: Array<FieldData>;
}
/* eslint-enable */
