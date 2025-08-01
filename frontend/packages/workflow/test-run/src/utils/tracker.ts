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
 
import { nanoid } from 'nanoid';

export type ExtraType = Record<string, any>;

/**
 * cache 类型
 */
export interface CacheMapType {
  start: number;
  disabled: boolean;
  extra?: ExtraType;
}

export class Tracker {
  cache = new Map<string, CacheMapType>();

  start(extra?: ExtraType) {
    const key = nanoid();
    const start = performance.now();
    const prev = this.cache.get(key);
    const value = {
      start,
      extra,
      // 如果已经存在，则永久禁用该事件上报
      disabled: !!prev,
    };
    this.cache.set(key, value);
    return key;
  }
  end(key: string): null | (CacheMapType & { duration: number }) {
    const prev = this.cache.get(key);
    if (!prev || prev.disabled) {
      return null;
    }
    const duration = performance.now() - prev.start;
    // 成功上报重置状态，等待下一次上报
    this.cache.delete(key);
    return { ...prev, duration };
  }
}
