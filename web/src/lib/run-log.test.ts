import { describe, it, expect } from 'vitest';
import { buildTree, toolLabel, inputDetail } from './run-log';

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
