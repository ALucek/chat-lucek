import type { RunNode } from './api';

export type TreeNode = RunNode & { children: TreeNode[] };

// buildTree links nodes into a forest by parent_id, preserving array order.
export function buildTree(nodes: RunNode[]): TreeNode[] {
  const wrapped = nodes.map((n) => ({ ...n, children: [] as TreeNode[] }));
  const byId = new Map(wrapped.map((n) => [n.id, n]));
  const roots: TreeNode[] = [];
  for (const n of wrapped) {
    const parent = n.parent_id ? byId.get(n.parent_id) : undefined;
    if (parent) parent.children.push(n);
    else roots.push(n);
  }
  return roots;
}

const TOOL_LABELS: Record<string, string> = {
  run_subagent: 'subagent',
  internet_search: 'search',
  set_todos: 'plan',
};

// toolLabel humanizes a tool name, falling back to the raw name.
export function toolLabel(name: string): string {
  return TOOL_LABELS[name] ?? name;
}

// inputDetail is a one-line digest of a tool's input, no per-tool code.
export function inputDetail(input: unknown): string {
  if (input == null || typeof input !== 'object') return '';
  const values = Object.values(input as Record<string, unknown>);
  if (values.length === 0) return '';
  if (values.length === 1 && typeof values[0] !== 'object')
    return String(values[0]);
  return JSON.stringify(input);
}
