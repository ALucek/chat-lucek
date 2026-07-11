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

// answerText joins a reply's top-level text blocks with a blank line.
export function answerText(nodes: RunNode[]): string {
  return nodes
    .filter((n) => n.type === 'text' && !n.parent_id)
    .map((n) => (n.text ?? '').trim())
    .filter(Boolean)
    .join('\n\n');
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

export type PlanProgress = 'completed' | 'in progress' | 'outstanding';

export interface PlanTodo {
  description: string;
  progress: PlanProgress;
}

const PROGRESS: readonly PlanProgress[] = [
  'completed',
  'in progress',
  'outstanding',
];

// coerceTodo returns a validated todo, or null if the raw item is malformed.
function coerceTodo(raw: unknown): PlanTodo | null {
  if (!raw || typeof raw !== 'object') return null;
  const { description, progress } = raw as Record<string, unknown>;
  if (typeof description !== 'string') return null;
  if (!PROGRESS.includes(progress as PlanProgress)) return null;
  return { description, progress: progress as PlanProgress };
}

// coercePlan pulls a non-empty valid todo list from a { todos } payload.
function coercePlan(payload: unknown): PlanTodo[] | null {
  if (!payload || typeof payload !== 'object') return null;
  const raw = (payload as { todos?: unknown }).todos;
  if (!Array.isArray(raw)) return null;
  const todos = raw.map(coerceTodo).filter((t): t is PlanTodo => t !== null);
  return todos.length > 0 ? todos : null;
}

// planFromNode reads a node's todos from input, falling back to output.
function planFromNode(node: RunNode): PlanTodo[] | null {
  const fromInput = coercePlan(node.input);
  if (fromInput) return fromInput;
  if (typeof node.output !== 'string') return null;
  try {
    return coercePlan(JSON.parse(node.output));
  } catch {
    return null;
  }
}

// latestPlan returns the todos from the last set_todos node, or null if none.
export function latestPlan(nodes: RunNode[]): PlanTodo[] | null {
  let plan: PlanTodo[] | null = null;
  for (const node of nodes) {
    if (node.name !== 'set_todos') continue;
    const next = planFromNode(node);
    if (next) plan = next;
  }
  return plan;
}
