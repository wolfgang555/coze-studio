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
 
import type { FC } from 'react';

import {
  useNodeRender,
  usePlayground,
} from '@flowgram-adapter/free-layout-editor';

import type { CommentEditorModel } from '../model';
import { DragArea } from './drag-area';

interface IBlankArea {
  model: CommentEditorModel;
}

export const BlankArea: FC<IBlankArea> = props => {
  const { model } = props;
  const playground = usePlayground();
  const { selectNode } = useNodeRender();

  return (
    <div
      className="workflow-comment-blank-area h-full w-full"
      onMouseDown={e => {
        e.preventDefault();
        e.stopPropagation();
        model.setFocus(false);
        selectNode(e);
        playground.node.focus(); // 防止节点无法被删除
      }}
      onClick={e => {
        model.setFocus(true);
        model.selectEnd();
      }}
    >
      <DragArea
        className="relative h-full w-full"
        model={model}
        stopEvent={false}
      />
    </div>
  );
};
