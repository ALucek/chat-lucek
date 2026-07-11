import { describe, it, expect } from 'vitest';
import {
  buildTree,
  toolLabel,
  inputDetail,
  latestPlan,
  answerText,
} from './run-log';
import type { RunNode } from './api';

describe('buildTree', () => {
  it('nests children under their parent, preserves order', () => {
    const tree = buildTree([
      { id: 'r1', parent_id: null, type: 'reasoning', text: 'a' },
      { id: 'SA', parent_id: null, type: 'tool', name: 'run_subagent' },
      { id: 's1', parent_id: 'SA', type: 'tool', name: 'internet_search' },
      { id: 't', parent_id: null, type: 'text', text: 'ans' },
    ]);
    expect(tree.map((n) => n.id)).toEqual(['r1', 'SA', 't']);
    const sa = tree.find((n) => n.id === 'SA')!;
    expect(sa.children.map((c) => c.id)).toEqual(['s1']);
  });

  it('handles an orphan parent reference as a root', () => {
    const tree = buildTree([{ id: 'x', parent_id: 'missing', type: 'text' }]);
    expect(tree.map((n) => n.id)).toEqual(['x']);
  });
});

describe('toolLabel', () => {
  it('humanizes known tools, falls back to raw name', () => {
    expect(toolLabel('run_subagent')).toBe('subagent');
    expect(toolLabel('internet_search')).toBe('search');
    expect(toolLabel('set_todos')).toBe('plan');
    expect(toolLabel('some_new_tool')).toBe('some_new_tool');
  });
});

describe('inputDetail', () => {
  it('single field -> its value; multi-field -> compact json; empty -> ""', () => {
    expect(inputDetail({ query: 'hello' })).toBe('hello');
    expect(inputDetail({ a: 1, b: 2 })).toBe('{"a":1,"b":2}');
    expect(inputDetail(undefined)).toBe('');
    expect(inputDetail({})).toBe('');
  });
});

function planNode(id: string, todos: unknown, output?: string): RunNode {
  return {
    id,
    parent_id: null,
    type: 'tool',
    name: 'set_todos',
    input: { todos },
    output,
  };
}

describe('latestPlan', () => {
  it('returns the todos of the last set_todos node', () => {
    const nodes: RunNode[] = [
      planNode('1', [{ description: 'a', progress: 'in progress' }]),
      planNode('2', [
        { description: 'a', progress: 'completed' },
        { description: 'b', progress: 'in progress' },
      ]),
    ];
    expect(latestPlan(nodes)).toEqual([
      { description: 'a', progress: 'completed' },
      { description: 'b', progress: 'in progress' },
    ]);
  });

  it('falls back to parsing output when input is absent', () => {
    const node: RunNode = {
      id: '1',
      parent_id: null,
      type: 'tool',
      name: 'set_todos',
      output: JSON.stringify({
        todos: [{ description: 'x', progress: 'outstanding' }],
      }),
    };
    expect(latestPlan([node])).toEqual([
      { description: 'x', progress: 'outstanding' },
    ]);
  });

  it('returns null when there is no set_todos node', () => {
    const nodes: RunNode[] = [
      {
        id: '1',
        parent_id: null,
        type: 'tool',
        name: 'internet_search',
        input: {},
      },
    ];
    expect(latestPlan(nodes)).toBeNull();
  });

  it('drops malformed items but keeps valid ones', () => {
    const nodes = [
      planNode('1', [
        { description: 'good', progress: 'completed' },
        { description: 'bad status', progress: 'nope' },
        { description: 42, progress: 'outstanding' },
      ]),
    ];
    expect(latestPlan(nodes)).toEqual([
      { description: 'good', progress: 'completed' },
    ]);
  });

  it('a later malformed node does not clobber an earlier valid plan', () => {
    const nodes: RunNode[] = [
      planNode('1', [{ description: 'a', progress: 'completed' }]),
      {
        id: '2',
        parent_id: null,
        type: 'tool',
        name: 'set_todos',
        output: 'not json{',
      },
    ];
    expect(latestPlan(nodes)).toEqual([
      { description: 'a', progress: 'completed' },
    ]);
  });
});

describe('answerText', () => {
  it('joins separate top-level text blocks with a blank line', () => {
    const nodes: RunNode[] = [
      { id: 'a', parent_id: null, type: 'text', text: 'first' },
      { id: 'SA', parent_id: null, type: 'tool', name: 'run_subagent' },
      { id: 'b', parent_id: null, type: 'text', text: 'second' },
    ];
    expect(answerText(nodes)).toBe('first\n\nsecond');
  });

  it('ignores reasoning, tools, and nested text', () => {
    const nodes: RunNode[] = [
      { id: 'r', parent_id: null, type: 'reasoning', text: 'thinking' },
      { id: 'SA', parent_id: null, type: 'tool', name: 'run_subagent' },
      { id: 's', parent_id: 'SA', type: 'text', text: 'nested' },
      { id: 'a', parent_id: null, type: 'text', text: 'answer' },
    ];
    expect(answerText(nodes)).toBe('answer');
  });

  it('drops whitespace-only blocks', () => {
    const nodes: RunNode[] = [
      { id: 'a', parent_id: null, type: 'text', text: 'one' },
      { id: 'ws', parent_id: null, type: 'text', text: '\n ' },
      { id: 'b', parent_id: null, type: 'text', text: 'two' },
    ];
    expect(answerText(nodes)).toBe('one\n\ntwo');
  });
});
